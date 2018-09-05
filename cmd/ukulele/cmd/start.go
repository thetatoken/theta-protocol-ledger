package cmd

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/node"
	"github.com/thetatoken/ukulele/p2p/messenger"
	"github.com/thetatoken/ukulele/store"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Theta node.",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func start() {
	port := viper.GetInt(common.CfgP2PPort)

	// Parse seeds and filter out empty item.
	f := func(c rune) bool {
		return c == ','
	}
	peerSeeds := strings.FieldsFunc(viper.GetString(common.CfgP2PSeeds), f)

	network := newMessenger(peerSeeds, port)

	checkpoint, err := consensus.LoadCheckpoint(path.Join(cfgPath, "checkpoint.json"))
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to load checkpoint")
	}
	validators := checkpoint.Validators
	chainID := checkpoint.ChainID
	rootEpoch := checkpoint.StartingEpoch
	rootHash, err := hex.DecodeString(checkpoint.StartingHash)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to parse checkpoint hash")
	}

	store := store.NewMemKVStore()
	root := &blockchain.Block{}
	root.ChainID = chainID
	root.Epoch = rootEpoch
	root.Hash = rootHash

	params := &node.Params{
		Store:      store,
		ChainID:    chainID,
		Root:       root,
		Validators: consensus.NewTestValidatorSet(validators),
		Network:    network,
	}
	n := node.NewNode(params)
	n.Start(context.Background())

	n.Wait()
}

func loadOrCreateKey() *ecdsa.PrivateKey {
	filepath := path.Join(cfgPath, "key")
	privKey, err := crypto.LoadECDSA(filepath)
	if err == nil {
		return privKey
	}
	log.WithFields(log.Fields{"err": err}).Warning("Failed to load private key. Generating new key")
	privKey, err = crypto.GenerateKey()
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to generate private key")
	}
	err = crypto.SaveECDSA(filepath, privKey)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to save private key")
	}
	return privKey
}

func newMessenger(seedPeerNetAddresses []string, port int) *messenger.Messenger {
	privKey := loadOrCreateKey()
	log.WithFields(log.Fields{
		"pubKey":  fmt.Sprintf("%X", crypto.FromECDSAPub(&privKey.PublicKey)),
		"address": fmt.Sprintf("%X", crypto.PubkeyToAddress(privKey.PublicKey)),
	}).Info("Using key")
	msgrConfig := messenger.GetDefaultMessengerConfig()
	msgrConfig.SetAddressBookFilePath(path.Join(cfgPath, "addrbook.json"))
	messenger, err := messenger.CreateMessenger(privKey.PublicKey, seedPeerNetAddresses, port, msgrConfig)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to create PeerDiscoveryManager instance")
	}
	return messenger
}
