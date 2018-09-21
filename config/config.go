package config

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	// CommonConfig means config object
	CommonConfig *Config
)

type Config struct {
	// Top level options use an anonymous struct
	BaseConfig `mapstructure:",squash"`			//匿名结构体,表示BaseConfig中的变量可以通过Config直接使用
	// Options for services
	P2P    *P2PConfig     `mapstructure:"p2p"`   //这里使用的是结构体中的tag,通过反射可以获取这个mapstructure对应的值
	Wallet *WalletConfig  `mapstructure:"wallet"`
	Auth   *RPCAuthConfig `mapstructure:"auth"`
	Web    *WebConfig     `mapstructure:"web"`
	Simd   *SimdConfig    `mapstructure:"simd"`
}

// Default configurable parameters.
func DefaultConfig() *Config {
	return &Config{
		BaseConfig: DefaultBaseConfig(),	 //node基础相关配置
		P2P:        DefaultP2PConfig(),		 //p2p网络相关配置
		Wallet:     DefaultWalletConfig(),	 //钱包相关配置
		Auth:       DefaultRPCAuthConfig(),	 //验证相关配置
		Web:        DefaultWebConfig(),		 //web相关配置
		Simd:       DefaultSimdConfig(),	 //web可配置网络参数
	}
}

// Set the RootDir for all Config structs
func (cfg *Config) SetRoot(root string) *Config {
	cfg.BaseConfig.RootDir = root
	cfg.P2P.RootDir = root
	return cfg
}

//-----------------------------------------------------------------------------
// BaseConfig
type BaseConfig struct {
	// The root directory for all data.
	// This should be set in viper so it can unmarshal into this struct
	RootDir string `mapstructure:"home"`

	//The ID of the network to json
	ChainID string `mapstructure:"chain_id"`

	//log level to set
	LogLevel string `mapstructure:"log_level"`

	// A JSON file containing the private key to use as a validator in the consensus protocol
	PrivateKey string `mapstructure:"private_key"`

	// A custom human readable name for this node
	Moniker string `mapstructure:"moniker"`

	// TCP or UNIX socket address for the profiling server to listen on
	ProfListenAddress string `mapstructure:"prof_laddr"`

	// If this node is many blocks behind the tip of the chain, FastSync
	// allows them to catchup quickly by downloading blocks in parallel
	// and verifying their commits
	FastSync bool `mapstructure:"fast_sync"`

	Mining bool `mapstructure:"mining"`

	FilterPeers bool `mapstructure:"filter_peers"` // false

	// What indexer to use for transactions
	TxIndex string `mapstructure:"tx_index"`

	// Database backend: leveldb | memdb
	DBBackend string `mapstructure:"db_backend"`

	// Database directory
	DBPath string `mapstructure:"db_dir"`

	// Keystore directory
	KeysPath string `mapstructure:"keys_dir"`

	// remote HSM url
	HsmUrl string `mapstructure:"hsm_url"`

	ApiAddress string `mapstructure:"api_addr"`

	VaultMode bool `mapstructure:"vault_mode"`

	Time time.Time

	// log file name
	LogFile string `mapstructure:"log_file"`
}

// Default configurable base parameters.
func DefaultBaseConfig() BaseConfig {
	return BaseConfig{
		Moniker:           "anonymous",
		ProfListenAddress: "",
		FastSync:          true,
		FilterPeers:       false,
		Mining:            false,
		TxIndex:           "kv",
		DBBackend:         "leveldb", //使用leveldb数据库
		DBPath:            "data",
		KeysPath:          "keystore",
		HsmUrl:            "",
	}
}

func (b BaseConfig) DBDir() string {
	return rootify(b.DBPath, b.RootDir)
}

func (b BaseConfig) KeysDir() string {
	return rootify(b.KeysPath, b.RootDir)
}

// P2PConfig
type P2PConfig struct {
	RootDir          string `mapstructure:"home"`
	ListenAddress    string `mapstructure:"laddr"`
	Seeds            string `mapstructure:"seeds"`
	SkipUPNP         bool   `mapstructure:"skip_upnp"`
	AddrBook         string `mapstructure:"addr_book_file"`
	AddrBookStrict   bool   `mapstructure:"addr_book_strict"`
	PexReactor       bool   `mapstructure:"pex"`
	MaxNumPeers      int    `mapstructure:"max_num_peers"`
	HandshakeTimeout int    `mapstructure:"handshake_timeout"`
	DialTimeout      int    `mapstructure:"dial_timeout"`
}

// Default configurable p2p parameters.
func DefaultP2PConfig() *P2PConfig {
	return &P2PConfig{
		ListenAddress:    "tcp://0.0.0.0:46656",
		AddrBook:         "addrbook.json",
		AddrBookStrict:   true,
		SkipUPNP:         false,
		MaxNumPeers:      50,
		HandshakeTimeout: 30,
		DialTimeout:      3,
		PexReactor:       true,
	}
}

func (p *P2PConfig) AddrBookFile() string {
	return rootify(p.AddrBook, p.RootDir)
}

//-----------------------------------------------------------------------------
type WalletConfig struct {
	Disable  bool   `mapstructure:"disable"`
	Rescan   bool   `mapstructure:"rescan"`
	MaxTxFee uint64 `mapstructure:"max_tx_fee"`
}

type RPCAuthConfig struct {
	Disable bool `mapstructure:"disable"`
}

type WebConfig struct {
	Closed bool `mapstructure:"closed"`
}

type SimdConfig struct {
	Enable bool `mapstructure:"enable"`
}

// Default configurable rpc's auth parameters.
func DefaultRPCAuthConfig() *RPCAuthConfig {
	return &RPCAuthConfig{
		Disable: false,
	}
}

// Default configurable web parameters.
func DefaultWebConfig() *WebConfig {
	return &WebConfig{
		Closed: false,
	}
}

// Default configurable wallet parameters.
func DefaultWalletConfig() *WalletConfig {
	return &WalletConfig{
		Disable:  false,
		Rescan:   false,
		MaxTxFee: uint64(1000000000),
	}
}

// Default configurable web parameters.
func DefaultSimdConfig() *SimdConfig {
	return &SimdConfig{
		Enable: false,
	}
}

//-----------------------------------------------------------------------------
// Utils

// helper function to make config creation independent of root dir
func rootify(path, root string) string {
	//判断是否绝对路径,像这样 c://
	if filepath.IsAbs(path) {
		return path
	}
	//不是根路径的话合并到root路径下面,作为根路径返回
	return filepath.Join(root, path)
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home == "" {
		return "./.bytom"
	}
	switch runtime.GOOS {
	case "darwin":
		// In order to be compatible with old data path,
		// copy the data from the old path to the new path
		oldPath := filepath.Join(home, "Library", "Bytom")
		newPath := filepath.Join(home, "Library", "Application Support", "Bytom")
		if !isFolderNotExists(oldPath) && isFolderNotExists(newPath) {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Errorf("DefaultDataDir: %v", err)
				return oldPath
			}
		}
		return newPath
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Bytom")
	default:
		return filepath.Join(home, ".bytom")
	}
}

func isFolderNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}


func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
