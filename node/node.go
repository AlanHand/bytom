package node

import (
	"context"
	"errors"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"

	"github.com/prometheus/prometheus/util/flock"
	log "github.com/sirupsen/logrus"
	cmn "github.com/tendermint/tmlibs/common"
	dbm "github.com/tendermint/tmlibs/db"
	browser "github.com/toqueteos/webbrowser"

	"github.com/bytom/accesstoken"
	"github.com/bytom/account"
	"github.com/bytom/api"
	"github.com/bytom/asset"
	"github.com/bytom/blockchain/pseudohsm"
	"github.com/bytom/blockchain/txfeed"
	cfg "github.com/bytom/config"
	"github.com/bytom/consensus"
	"github.com/bytom/database/leveldb"
	"github.com/bytom/env"
	"github.com/bytom/mining/cpuminer"
	"github.com/bytom/mining/miningpool"
	"github.com/bytom/mining/tensority"
	"github.com/bytom/netsync"
	"github.com/bytom/protocol"
	"github.com/bytom/protocol/bc"
	w "github.com/bytom/wallet"
)

const (
	webHost           = "http://127.0.0.1"
	maxNewBlockChSize = 1024
)

type Node struct {
	cmn.BaseService

	// config
	config *cfg.Config

	syncManager *netsync.SyncManager

	//bcReactor    *bc.BlockchainReactor
	wallet       *w.Wallet
	accessTokens *accesstoken.CredentialStore
	api          *api.API
	chain        *protocol.Chain
	txfeed       *txfeed.Tracker
	cpuMiner     *cpuminer.CPUMiner
	miningPool   *miningpool.MiningPool
	miningEnable bool
}
//初始化node运行环境
func NewNode(config *cfg.Config) *Node {
	//获取全局上下文context
	ctx := context.Background()
	//将根目录先锁住
	if err := lockDataDirectory(config); err != nil {
		cmn.Exit("Error: " + err.Error())
	}
	//初始化日志文件
	initLogFile(config)
	//初始化网络参数,确认是主网,测试网还是私链的一种
	initActiveNetParams(config)
	//初始化命令行参数
	initCommonConfig(config)

	// Get store 数据库只能是memdb内存数据库或者leveldb
	if config.DBBackend != "memdb" && config.DBBackend != "leveldb" {
		cmn.Exit(cmn.Fmt("Param db_backend [%v] is invalid, use leveldb or memdb", config.DBBackend))
	}
	// 初始化core数据库
	coreDB := dbm.NewDB("core", config.DBBackend, config.DBDir())
	store := leveldb.NewStore(coreDB)

	//初始化accesstoken数据库
	tokenDB := dbm.NewDB("accesstoken", config.DBBackend, config.DBDir())
	accessTokens := accesstoken.NewStore(tokenDB)

	//初始化交易池
	txPool := protocol.NewTxPool(store)
	chain, err := protocol.NewChain(store, txPool)
	if err != nil {
		cmn.Exit(cmn.Fmt("Failed to create chain structure: %v", err))
	}

	var accounts *account.Manager = nil
	var assets *asset.Registry = nil
	var wallet *w.Wallet = nil
	var txFeed *txfeed.Tracker = nil

	//初始化txfeeds数据库
	txFeedDB := dbm.NewDB("txfeeds", config.DBBackend, config.DBDir())
	txFeed = txfeed.NewTracker(txFeedDB, chain)

	if err = txFeed.Prepare(ctx); err != nil {
		log.WithField("error", err).Error("start txfeed")
		return nil
	}

	//初始化keystore
	hsm, err := pseudohsm.New(config.KeysDir())
	if err != nil {
		cmn.Exit(cmn.Fmt("initialize HSM failed: %v", err))
	}

	if !config.Wallet.Disable {
		walletDB := dbm.NewDB("wallet", config.DBBackend, config.DBDir())
		accounts = account.NewManager(walletDB, chain)
		assets = asset.NewRegistry(walletDB, chain)
		wallet, err = w.NewWallet(walletDB, accounts, assets, hsm, chain)
		if err != nil {
			log.WithField("error", err).Error("init NewWallet")
		}

		// trigger rescan wallet
		if config.Wallet.Rescan {
			wallet.RescanBlocks()
		}
	}
	newBlockCh := make(chan *bc.Hash, maxNewBlockChSize)

	//初始化网络节点同步管理
	syncManager, _ := netsync.NewSyncManager(config, chain, txPool, newBlockCh)

	// get transaction from txPool and send it to syncManager and wallet
	go newPoolTxListener(txPool, syncManager, wallet)

	//初始化pprof，pprof用于输出性能指标，需要制定--prof_laddr参数来开启，在文章开头我们已经开启该功能
	// run the profile server
	profileHost := config.ProfListenAddress
	if profileHost != "" {
		// Profiling bytomd programs.see (https://blog.golang.org/profiling-go-programs)
		// go tool pprof http://profileHose/debug/pprof/heap
		go func() {
			if err = http.ListenAndServe(profileHost, nil); err != nil {
				cmn.Exit(cmn.Fmt("Failed to register tcp profileHost: %v", err))
			}
		}()
	}

	//初始化节点 , 填充节点所需的所有参数环境
	node := &Node{
		config:       config,
		syncManager:  syncManager,
		accessTokens: accessTokens,
		wallet:       wallet,
		chain:        chain,
		txfeed:       txFeed,
		miningEnable: config.Mining,
	}

	// 初始化挖矿
	node.cpuMiner = cpuminer.NewCPUMiner(chain, accounts, txPool, newBlockCh)
	node.miningPool = miningpool.NewMiningPool(chain, accounts, txPool, newBlockCh)

	node.BaseService = *cmn.NewBaseService(nil, "Node", node)

	if config.Simd.Enable {
		tensority.UseSIMD = true
	}

	return node
}

// newPoolTxListener listener transaction from txPool, and send it to syncManager and wallet
func newPoolTxListener(txPool *protocol.TxPool, syncManager *netsync.SyncManager, wallet *w.Wallet) {
	txMsgCh := txPool.GetMsgCh()
	syncManagerTxCh := syncManager.GetNewTxCh()

	for {
		msg := <-txMsgCh
		switch msg.MsgType {
		case protocol.MsgNewTx:
			syncManagerTxCh <- msg.Tx
			if wallet != nil {
				wallet.AddUnconfirmedTx(msg.TxDesc)
			}
		case protocol.MsgRemoveTx:
			if wallet != nil {
				wallet.RemoveUnconfirmedTx(msg.TxDesc)
			}
		default:
			log.Warn("got unknow message type from the txPool channel")
		}
	}
}

// Lock data directory after daemonization
func lockDataDirectory(config *cfg.Config) error {
	_, _, err := flock.New(filepath.Join(config.RootDir, "LOCK"))
	if err != nil {
		return errors.New("datadir already used by another process")
	}
	return nil
}

func initActiveNetParams(config *cfg.Config) {
	var exist bool
	consensus.ActiveNetParams, exist = consensus.NetParams[config.ChainID]
	if !exist {
		cmn.Exit(cmn.Fmt("chain_id[%v] don't exist", config.ChainID))
	}
}

func initLogFile(config *cfg.Config) {
	if config.LogFile == "" {
		return
	}
	cmn.EnsureDir(filepath.Dir(config.LogFile), 0700)
	file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(file)
	} else {
		log.WithField("err", err).Info("using default")
	}

}

func initCommonConfig(config *cfg.Config) {
	cfg.CommonConfig = config
}

// Lanch web broser or not
func launchWebBrowser(port string) {
	webAddress := webHost + ":" + port
	log.Info("Launching System Browser with :", webAddress)
	if err := browser.Open(webAddress); err != nil {
		log.Error(err.Error())
		return
	}
}

func (n *Node) initAndstartApiServer() {
	n.api = api.NewAPI(n.syncManager, n.wallet, n.txfeed, n.cpuMiner, n.miningPool, n.chain, n.config, n.accessTokens)

	listenAddr := env.String("LISTEN", n.config.ApiAddress)
	env.Parse()
	n.api.StartServer(*listenAddr)
}
//启动节点
func (n *Node) OnStart() error {
	if n.miningEnable {
		if _, err := n.wallet.AccountMgr.GetMiningAddress(); err != nil {
			n.miningEnable = false
			log.Error(err)
		} else {
			//启动挖矿功能
			n.cpuMiner.Start()
		}
	}
	if !n.config.VaultMode {
		//启动p2p网络同步
		n.syncManager.Start()
	}
	//启动http协议的apiserver功能
	n.initAndstartApiServer()
	if !n.config.Web.Closed {
		s := strings.Split(n.config.ApiAddress, ":")
		if len(s) != 2 {
			log.Error("Invalid api address")
		}
		//打开浏览器访问交易页面
		launchWebBrowser(s[1])
	}
	return nil
}

//退出时执行的函数,停止挖矿,停止网络同步
func (n *Node) OnStop() {
	n.BaseService.OnStop()
	if n.miningEnable {
		n.cpuMiner.Stop()
	}
	if !n.config.VaultMode {
		n.syncManager.Stop()
	}
}
//该函数是由tendermint框架启动了监听信号的功能,TrapSignal函数监听了SIGTERM信号，
// bytomd才能成为不退出的守护进程。只有当触发了ctrl+c或kill bytomd_pid才能终止bytomd进程退出。退出时bytomd执行如下操作 OnStop()函数
func (n *Node) RunForever() {
	// Sleep forever and then...
	cmn.TrapSignal(func() {
		n.Stop()
	})
}

func (n *Node) SyncManager() *netsync.SyncManager {
	return n.syncManager
}

func (n *Node) MiningPool() *miningpool.MiningPool {
	return n.miningPool
}
