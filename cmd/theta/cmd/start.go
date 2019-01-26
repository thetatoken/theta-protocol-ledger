package cmd

import (
	"context"
	"fmt"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/netsync"
	"github.com/thetatoken/theta/node"
	"github.com/thetatoken/theta/p2p/messenger"
	"github.com/thetatoken/theta/store/database/backend"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Theta node.",
	Run:   runStart,
}

func init() {
	RootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) {
	port := viper.GetInt(common.CfgP2PPort)

	// Parse seeds and filter out empty item.
	f := func(c rune) bool {
		return c == ','
	}
	peerSeeds := strings.FieldsFunc(viper.GetString(common.CfgP2PSeeds), f)
	privKey := loadOrCreateKey()

	network := newMessenger(privKey, peerSeeds, port)
	mainDBPath := path.Join(cfgPath, "db", "main")
	refDBPath := path.Join(cfgPath, "db", "ref")
	db, err := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)

	var root *core.Block
	snapshotBlockHeader, err := netsync.LoadSnapshot(snapshotPath, db)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to load snapshot")
	}
	root = &core.Block{BlockHeader: snapshotBlockHeader}

	params := &node.Params{
		ChainID:    root.ChainID,
		PrivateKey: privKey,
		Root:       root,
		Network:    network,
		DB:         db,
	}
	n := node.NewNode(params)
	n.Start(context.Background())

	n.Wait()
}

func loadOrCreateKey() *crypto.PrivateKey {
	filepath := path.Join(cfgPath, "key")
	privKey, err := crypto.PrivateKeyFromFile(filepath)
	if err == nil {
		return privKey
	}
	log.WithFields(log.Fields{"err": err}).Warning("Failed to load private key. Generating new key")
	privKey, _, err = crypto.GenerateKeyPair()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to generate private key")
	}
	err = privKey.SaveToFile(filepath)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to save private key")
	}
	return privKey
}

func newMessenger(privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int) *messenger.Messenger {
	log.WithFields(log.Fields{
		"pubKey":  fmt.Sprintf("%v", privKey.PublicKey().ToBytes()),
		"address": fmt.Sprintf("%v", privKey.PublicKey().Address()),
	}).Info("Using key")
	msgrConfig := messenger.GetDefaultMessengerConfig()
	msgrConfig.SetAddressBookFilePath(path.Join(cfgPath, "addrbook.json"))
	messenger, err := messenger.CreateMessenger(privKey.PublicKey(), seedPeerNetAddresses, port, msgrConfig)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to create PeerDiscoveryManager instance")
	}
	return messenger
}
