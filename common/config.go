package common

import (
	"fmt"
	"os"
	"path"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"

	log "github.com/sirupsen/logrus"
)

const (
	// CfgChainID defines the chain ID.
	CfgChainID = "chain.ID"

	// CfgConsensusMaxEpochLength defines the maxium length of an epoch.
	CfgConsensusMaxEpochLength = "consensus.maxEpochLength"
	// CfgConsensusMessageQueueSize defines the capacity of consensus message queue.
	CfgConsensusMessageQueueSize = "consensus.messageQueueSize"

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
	// CfgLogDebug sets the log level.
	CfgLogDebug = "log.debug"
)

// InitialConfig is the default configuartion produced by init command.
const InitialConfig = `# Theta configuration
p2p:
  port: 5000
  seeds: 127.0.0.1:6000,127.0.0.1:7000
`

func init() {
	viper.SetDefault(CfgChainID, "localchain")

	viper.SetDefault(CfgConsensusMaxEpochLength, 2)
	viper.SetDefault(CfgConsensusMessageQueueSize, 512)

	viper.SetDefault(CfgSyncMessageQueueSize, 512)

	viper.SetDefault(CfgP2PMessageQueueSize, 512)
	viper.SetDefault(CfgP2PName, "Anonymous")
	viper.SetDefault(CfgP2PPort, 50001)
	viper.SetDefault(CfgP2PSeeds, "")

	viper.SetDefault(CfgLogDebug, false)

	log.SetLevel(log.DebugLevel)
}

// GetDefaultConfigPath returns the default config path.
func GetDefaultConfigPath() string {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return path.Join(home, ".ukulele")
}

// WriteInitialConfig writes initial config file to file system.
func WriteInitialConfig(filePath string) error {
	return WriteFileAtomic(filePath, []byte(InitialConfig), 0600)
}
