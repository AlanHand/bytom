package protocol

import (
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/config"
	"github.com/bytom/errors"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

const maxProcessBlockChSize = 1024

// Chain provides functions for working with the Bytom block chain.
type Chain struct {
	index          *state.BlockIndex
	orphanManage   *OrphanManage
	txPool         *TxPool
	store          Store
	processBlockCh chan *processBlockMsg

	cond     sync.Cond
	bestNode *state.BlockNode
}

// NewChain returns a new Chain using store as the underlying storage.
// 创建一条新链,加载创世区块
func NewChain(store Store, txPool *TxPool) (*Chain, error) {
	//初始化链结构体
	c := &Chain{
		orphanManage:   NewOrphanManage(),
		txPool:         txPool,
		store:          store,
		// 协程通道,通道数据为块数据消息,通道大小可容纳1024个块
		processBlockCh: make(chan *processBlockMsg, maxProcessBlockChSize),
	}
	//互斥锁
	c.cond.L = new(sync.Mutex)
	// 当我们第一次启动比原链节点时，store.GetStoreStatus会从db中获取存储状态，
	// 获取存储状态的过程是从LevelDB中查询key为blockStore的数据，如果查询出错则认为是第一次运行比原链节点，
	// 那么就需要初始化
	storeStatus := store.GetStoreStatus()
	if storeStatus == nil {
		//初始化链,加载创世区块
		if err := c.initChainStatus(); err != nil {
			return nil, err
		}
		//获取存储状态
		storeStatus = store.GetStoreStatus()
	}

	var err error
	if c.index, err = store.LoadBlockIndex(storeStatus.Height); err != nil {
		return nil, err
	}

	c.bestNode = c.index.GetNode(storeStatus.Hash)
	//设置主链
	c.index.SetMainChain(c.bestNode)
	// 获取通道的区块数据消息
	go c.blockProcesser()
	return c, nil
}

// 初始化主链,加载创世区块
func (c *Chain) initChainStatus() error {
	// 获取创世区块
	genesisBlock := config.GenesisBlock()
	// 设置创世区块中所有的交易状态
	txStatus := bc.NewTransactionStatus()
	for i := range genesisBlock.Transactions {
		if err := txStatus.SetStatus(i, false); err != nil {
			return err
		}
	}

	// 存储创世区块到leveldb中
	if err := c.store.SaveBlock(genesisBlock, txStatus); err != nil {
		return err
	}
	// 临时小部分utxo状态集合
	utxoView := state.NewUtxoViewpoint()
	bcBlock := types.MapBlock(genesisBlock)
	if err := utxoView.ApplyBlock(bcBlock, txStatus); err != nil {
		return err
	}

	//选择最佳链作为主链
	node, err := state.NewBlockNode(&genesisBlock.BlockHeader, nil)
	if err != nil {
		return err
	}
	//保存最新主链状态
	return c.store.SaveChainStatus(node, utxoView)
}

// BestBlockHeight returns the current height of the blockchain.
func (c *Chain) BestBlockHeight() uint64 {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return c.bestNode.Height
}

// BestBlockHash return the hash of the chain tail block
func (c *Chain) BestBlockHash() *bc.Hash {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	return &c.bestNode.Hash
}

// BestBlockHeader returns the chain tail block
func (c *Chain) BestBlockHeader() *types.BlockHeader {
	node := c.index.BestNode()
	return node.BlockHeader()
}

// InMainChain checks wheather a block is in the main chain
func (c *Chain) InMainChain(hash bc.Hash) bool {
	return c.index.InMainchain(hash)
}

// CalcNextSeed return the seed for the given block
func (c *Chain) CalcNextSeed(preBlock *bc.Hash) (*bc.Hash, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return nil, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextSeed(), nil
}

// CalcNextBits return the seed for the given block
func (c *Chain) CalcNextBits(preBlock *bc.Hash) (uint64, error) {
	node := c.index.GetNode(preBlock)
	if node == nil {
		return 0, errors.New("can't find preblock in the blockindex")
	}
	return node.CalcNextBits(), nil
}

// This function must be called with mu lock in above level
func (c *Chain) setState(node *state.BlockNode, view *state.UtxoViewpoint) error {
	if err := c.store.SaveChainStatus(node, view); err != nil {
		return err
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	c.index.SetMainChain(node)
	c.bestNode = node

	log.WithFields(log.Fields{"height": c.bestNode.Height, "hash": c.bestNode.Hash}).Debug("chain best status has been update")
	c.cond.Broadcast()
	return nil
}

// BlockWaiter returns a channel that waits for the block at the given height.
func (c *Chain) BlockWaiter(height uint64) <-chan struct{} {
	ch := make(chan struct{}, 1)
	go func() {
		c.cond.L.Lock()
		defer c.cond.L.Unlock()
		for c.bestNode.Height < height {
			c.cond.Wait()
		}
		ch <- struct{}{}
	}()

	return ch
}

// GetTxPool return chain txpool.
func (c *Chain) GetTxPool() *TxPool {
	return c.txPool
}
