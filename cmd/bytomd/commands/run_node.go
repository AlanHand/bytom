package commands

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/bytom/node"
	"strings"
)

var runNodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the bytomd",
	RunE:  runNode,
}
//初始化node节点运行时所需要的传参
func init() {
	runNodeCmd.Flags().String("prof_laddr", config.ProfListenAddress, "Use http to profile bytomd programs")
	runNodeCmd.Flags().Bool("mining", config.Mining, "Enable mining")

	runNodeCmd.Flags().Bool("simd.enable", config.Simd.Enable, "Enable SIMD mechan for tensority")

	runNodeCmd.Flags().Bool("auth.disable", config.Auth.Disable, "Disable rpc access authenticate")

	runNodeCmd.Flags().Bool("wallet.disable", config.Wallet.Disable, "Disable wallet")
	runNodeCmd.Flags().Bool("wallet.rescan", config.Wallet.Rescan, "Rescan wallet")
	runNodeCmd.Flags().Bool("vault_mode", config.VaultMode, "Run in the offline enviroment")
	runNodeCmd.Flags().Bool("web.closed", config.Web.Closed, "Lanch web browser or not")
	runNodeCmd.Flags().String("chain_id", config.ChainID, "Select network type")

	// log level
	runNodeCmd.Flags().String("log_level", config.LogLevel, "Select log level(debug, info, warn, error or fatal")

	// p2p flags
	runNodeCmd.Flags().String("p2p.laddr", config.P2P.ListenAddress, "Node listen address. (0.0.0.0:0 means any interface, any port)")
	runNodeCmd.Flags().String("p2p.seeds", config.P2P.Seeds, "Comma delimited host:port seed nodes")
	runNodeCmd.Flags().Bool("p2p.skip_upnp", config.P2P.SkipUPNP, "Skip UPNP configuration")
	runNodeCmd.Flags().Bool("p2p.pex", config.P2P.PexReactor, "Enable Peer-Exchange ")
	runNodeCmd.Flags().Int("p2p.max_num_peers", config.P2P.MaxNumPeers, "Set max num peers")
	runNodeCmd.Flags().Int("p2p.handshake_timeout", config.P2P.HandshakeTimeout, "Set handshake timeout")
	runNodeCmd.Flags().Int("p2p.dial_timeout", config.P2P.DialTimeout, "Set dial timeout")

	// log flags
	runNodeCmd.Flags().String("log_file", config.LogFile, "Log output file")

	RootCmd.AddCommand(runNodeCmd)
}

func getLogLevel(level string) log.Level {
	switch strings.ToLower(level) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}

func runNode(cmd *cobra.Command, args []string) error {
	// Set log level by config.LogLevel
	log.SetLevel(getLogLevel(config.LogLevel))

	// 初始化node运行环境
	n := node.NewNode(config)
	// 启动节点
	if _, err := n.Start(); err != nil {
		return fmt.Errorf("Failed to start node: %v", err)
	} else {
		log.Info("Start node ", n.SyncManager().NodeInfo())
	}

	// 监听退出信号，收到ctrl+c操作则退出node。在linux中守进程一般监听SIGTERM信号
	n.RunForever()

	return nil
}
