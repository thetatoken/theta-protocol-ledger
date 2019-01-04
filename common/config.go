package common

import (
	"github.com/spf13/viper"
)

const (
	// CfgChainID defines the chain ID.
	CfgChainID = "chain.ID"

	// CfgConsensusMaxEpochLength defines the maxium length of an epoch.
	CfgConsensusMaxEpochLength = "consensus.maxEpochLength"
	// CfgConsensusMinProposalWait defines the minimal interval between proposals.
	CfgConsensusMinProposalWait = "consensus.minProposalWait"
	// CfgConsensusMessageQueueSize defines the capacity of consensus message queue.
	CfgConsensusMessageQueueSize = "consensus.messageQueueSize"
	// CfgConsensusMaxNumValidators defines the max number validators allowed
	CfgConsensusMaxNumValidators = "consensus.maxNumValidators"

	// CfgSyncMessageQueueSize defines the capacity of Sync Manager message queue.
	CfgSyncMessageQueueSize = "sync.messageQueueSize"

	// CfgP2PName sets the ID of local node in P2P network.
	CfgP2PName = "p2p.name"
	// CfgP2PPort sets the port used by P2P network.
	CfgP2PPort = "p2p.port"
	// CfgP2PSeeds sets the boostrap peers.
	CfgP2PSeeds = "p2p.seeds"
	// CfgP2PMessageQueueSize sets the message queue size for network interface.
	CfgP2PMessageQueueSize = "p2p.messageQueueSize"

	// CfgRPCEnabled sets whether to run RPC service.
	CfgRPCEnabled = "rpc.enabled"
	// CfgRPCPort sets the port of RPC service.
	CfgRPCPort = "rpc.port"
	// CfgRPCMaxConnections limits concurrent connections accepted by RPC server.
	CfgRPCMaxConnections = "rpc.maxConnections"

	// CfgLogLevels sets the log level.
	CfgLogLevels = "log.levels"
	// CfgLogPrintSelfID determines whether to print node's ID in log (Useful in simulation when
	// there are more than one node running).
	CfgLogPrintSelfID = "log.printSelfID"
)

// InitialConfig is the default configuartion produced by init command.
const InitialConfig = `# Theta configuration
p2p:
  port: 5000
  seeds: 127.0.0.1:6000,127.0.0.1:7000
`

func init() {
	viper.SetDefault(CfgChainID, "localchain")

	viper.SetDefault(CfgConsensusMaxEpochLength, 5)
	viper.SetDefault(CfgConsensusMinProposalWait, 2)
	viper.SetDefault(CfgConsensusMessageQueueSize, 512)
	viper.SetDefault(CfgConsensusMaxNumValidators, 7)

	viper.SetDefault(CfgSyncMessageQueueSize, 512)

	viper.SetDefault(CfgRPCEnabled, false)
	viper.SetDefault(CfgP2PMessageQueueSize, 512)
	viper.SetDefault(CfgP2PName, "Anonymous")
	viper.SetDefault(CfgP2PPort, 50001)
	viper.SetDefault(CfgP2PSeeds, "")

	viper.SetDefault(CfgRPCPort, "16888")
	viper.SetDefault(CfgRPCMaxConnections, 200)

	viper.SetDefault(CfgLogLevels, "*:debug")
	viper.SetDefault(CfgLogPrintSelfID, false)
}

// WriteInitialConfig writes initial config file to file system.
func WriteInitialConfig(filePath string) error {
	return WriteFileAtomic(filePath, []byte(InitialConfig), 0600)
}
