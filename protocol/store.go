package protocol

import (
	"github.com/bytom/database/storage"
	"github.com/bytom/protocol/bc"
	"github.com/bytom/protocol/bc/types"
	"github.com/bytom/protocol/state"
)

// Store provides storage interface for blockchain data
type Store interface {
	//根据Hash判断区块是否存在
	BlockExist(*bc.Hash) bool
	//根据Hash获取一个区块
	GetBlock(*bc.Hash) (*types.Block, error)
	//获取store的存储状态
	GetStoreStatus() *BlockStoreState
	//根据hash获取该块中所有交易的状态
	GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error)
	//缓存与输入交易相关的所有的utxo
	GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error
	//根据hash获取该块内所有的utxo
	GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)
	//从db中读取所有区块的头部信息并加载到缓存中
	LoadBlockIndex(uint64) (*state.BlockIndex, error)
	//存储块和交易状态
	SaveBlock(*types.Block, *bc.TransactionStatus) error
	//设置主链的状态,当节点第一次启动时节点会根据key为blockStore的内容判断是否初始化主链
	SaveChainStatus(*state.BlockNode, *state.UtxoViewpoint) error
}

// BlockStoreState represents the core's db status
type BlockStoreState struct {
	Height uint64
	Hash   *bc.Hash
}
