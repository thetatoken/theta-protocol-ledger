package common

import (
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

const (
	// CfgConsensusMaxEpochLength defines the maxium length of an epoch.
	CfgConsensusMaxEpochLength = "consensus.maxEpochLength"
	// CfgConsensusMessageQueueSize defines the capacity of consensus message queue.
	CfgConsensusMessageQueueSize = "consensus.messageQueueSize"
	// CfgP2PName sets the ID of local node in P2P network.
	CfgP2PName = "p2p.name"
	// CfgP2PPort sets the port used by P2P network.
	CfgP2PPort = "p2p.port"
	// CfgP2PMessageQueueSize sets the message queue size for network interface.
	CfgP2PMessageQueueSize = "p2p.messageQueueSize"
	// CfgLogDebug sets the log level.
	CfgLogDebug = "log.debug"
)

func init() {
	viper.SetDefault(CfgConsensusMaxEpochLength, 2)
	viper.SetDefault(CfgConsensusMessageQueueSize, 512)

	viper.SetDefault(CfgP2PMessageQueueSize, 512)
	viper.SetDefault(CfgP2PName, "Anonymous")
	viper.SetDefault(CfgP2PPort, 50001)

	viper.SetDefault(CfgLogDebug, false)

	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("No config file is loaded")
	}
	log.SetLevel(log.DebugLevel)
}
