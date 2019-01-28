package cmd

import (
	"context"
	"fmt"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/netsync"
	"github.com/thetatoken/theta/node"
	"github.com/thetatoken/theta/p2p/messenger"
	"github.com/thetatoken/theta/store/database/backend"
	ks "github.com/thetatoken/theta/wallet/softwallet/keystore"
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
	privKey, err := loadOrCreateKey()
	if err != nil {
		panic(fmt.Sprintf("Failed to load or create key: %v", err))
	}

	network := newMessenger(privKey, peerSeeds, port)
	mainDBPath := path.Join(cfgPath, "db", "main")
	refDBPath := path.Join(cfgPath, "db", "ref")
	db, err := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to the db. main: %v, ref: %v, err: %v",
			mainDBPath, refDBPath, err))
	}

	if len(snapshotPath) == 0 {
		snapshotPath = path.Join(cfgPath, "genesis")
	}
	snapshotBlockHeader, err := netsync.ValidateSnapshot(snapshotPath)
	if err != nil {
		panic(fmt.Sprintf("Snapshot validation failed, err: %v", err))
	}
	root := &core.Block{BlockHeader: snapshotBlockHeader}

	params := &node.Params{
		ChainID:      root.ChainID,
		PrivateKey:   privKey,
		Root:         root,
		Network:      network,
		DB:           db,
		SnapshotPath: snapshotPath,
	}
	n := node.NewNode(params)
	n.Start(context.Background())

	n.Wait()
}

func loadOrCreateKey() (*crypto.PrivateKey, error) {
	keysDir := path.Join(cfgPath, "key")
	keystore, err := ks.NewKeystoreEncrypted(keysDir, ks.StandardScryptN, ks.StandardScryptP)
	if err != nil {
		panic(fmt.Sprintf("Failed to create key store: %v", err))
	}
	addresses, err := keystore.ListKeyAddresses()
	if err != nil {
		panic(fmt.Sprintf("Failed to get key address: %v", err))
	}

	numAddrs := len(addresses)
	if numAddrs > 1 {
		return nil, fmt.Errorf("Multiple encrypted keys detected under %v. Please keep only one key.", path.Join(keysDir, "encrypted"))
	}

	var password string
	var nodeAddrss common.Address
	if numAddrs == 0 {
		fmt.Println("")
		fmt.Println(" --------------------------------------------------------------------------------------------------------------------")
		fmt.Println("| You are launching the Theta Node for the first time. Welcome and please follow the instructions to setup the node. |")
		fmt.Println(" --------------------------------------------------------------------------------------------------------------------")
		fmt.Println("")

		firstPrompt := fmt.Sprintf("Please choose your password for the Theta Node: ")
		firstPassword, err := utils.GetPassword(firstPrompt)
		if err != nil {
			return nil, fmt.Errorf("Failed to get password: %v", err)
		}
		secondPrompt := fmt.Sprintf("Please enter your password again: ")
		secondPassword, err := utils.GetPassword(secondPrompt)
		if err != nil {
			return nil, fmt.Errorf("Failed to get password: %v", err)
		}
		if firstPassword != secondPassword {
			return nil, fmt.Errorf("Passwords do not match")
		}
		fmt.Println("Thanks and please store your password securely. You will need it each time you launch the Theta node.")
		fmt.Println("")
		fmt.Println("Please press enter to continue..")
		utils.GetConfirmation()

		password = firstPassword

		privKey, _, err := crypto.GenerateKeyPair()
		if err != nil {
			return nil, err
		}

		key := ks.NewKey(privKey)
		err = keystore.StoreKey(key, password)
		if err != nil {
			return nil, err
		}
		nodeAddrss = key.Address
	} else {
		fmt.Println("")
		fmt.Println(" ----------------------------------------------------------")
		fmt.Println("|                      Hello, Theta!                       |")
		fmt.Println(" ----------------------------------------------------------")
		fmt.Println("")

		prompt := fmt.Sprintf("Please enter the password to launch the Theta node: ")
		password, err = utils.GetPassword(prompt)
		if err != nil {
			return nil, fmt.Errorf("Failed to get password: %v", err)
		}
		nodeAddrss = addresses[0]
	}

	nodeKey, err := keystore.GetKey(nodeAddrss, password)
	if err != nil {
		return nil, err
	}

	nodePrivKey := nodeKey.PrivateKey
	return nodePrivKey, nil
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
