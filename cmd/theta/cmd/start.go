package cmd

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/node"
	msg "github.com/thetatoken/theta/p2p/messenger"
	msgl "github.com/thetatoken/theta/p2pl/messenger"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/snapshot"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/version"
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
	var networkOld *msg.Messenger
	var network *msgl.Messenger
	var err error

	privKey, err := loadOrCreateKey()
	if err != nil {
		log.Fatalf("Failed to load or create key: %v", err)
	}

	// Open database
	dbPath := viper.GetString(common.CfgDataPath)
	if dbPath == "" {
		dbPath = cfgPath
	}

	mainDBPath := path.Join(dbPath, "db", "main")
	refDBPath := path.Join(dbPath, "db", "ref")
	db, err := backend.NewLDBDatabase(mainDBPath, refDBPath,
		viper.GetInt(common.CfgStorageLevelDBCacheSize),
		viper.GetInt(common.CfgStorageLevelDBHandles))

	if err != nil {
		log.Fatalf("Failed to connect to the db. main: %v, ref: %v, err: %v",
			mainDBPath, refDBPath, err)
	}

	// load snapshot
	if len(snapshotPath) == 0 {
		snapshotPath = path.Join(cfgPath, "snapshot")
	}

	var root *core.Block
	var snapshotBlockHeader *core.BlockHeader
	dbSnapshotHeader := &core.BlockHeader{}
	skipLoadSnapshot := false

	// Read last verified snapshot header from db and compare with current snapshot
	raw, err := db.Get([]byte("/snapshot_blockheader"))
	if err == nil {
		err = rlp.DecodeBytes(raw, dbSnapshotHeader)
		if err == nil {
			snapshotBlockHeader = snapshot.LoadSnapshotCheckpointHeader(snapshotPath)
			if snapshotBlockHeader.Hash() == dbSnapshotHeader.Hash() {
				// snapshot has already been loaded into db
				skipLoadSnapshot = true
			}
		}
	}
	if skipLoadSnapshot && !viper.GetBool(common.CfgForceValidateSnapshot) {
		log.Println("Skip validating snapshot")
	} else {
		snapshotBlockHeader, err = snapshot.ValidateSnapshot(snapshotPath, chainImportDirPath, chainCorrectionPath)
		if err != nil {
			log.Fatalf("Snapshot validation failed, err: %v", err)
		}

		raw, err := rlp.EncodeToBytes(snapshotBlockHeader)
		if err == nil {
			err = db.Put([]byte("/snapshot_blockheader"), raw)
			if err != nil {
				log.Errorf("Failed to save snapshot validation result: %v", err)
			}
		}
	}

	root = &core.Block{BlockHeader: snapshotBlockHeader}

	viper.Set(common.CfgGenesisChainID, root.ChainID)

	// Parse seeds and filter out empty item.
	f := func(c rune) bool {
		return c == ','
	}

	// trap Ctrl+C and call cancel on the context
	ctx, cancel := context.WithCancel(context.Background())

	p2pOpt := common.P2POptEnum(viper.GetInt(common.CfgP2POpt))
	if p2pOpt != common.P2POptOld {
		port := viper.GetInt(common.CfgP2PLPort)
		peerSeeds := strings.FieldsFunc(viper.GetString(common.CfgLibP2PSeeds), f)
		seedPeerOnly := viper.GetBool(common.CfgP2PSeedPeerOnly)
		network = newMessenger(privKey, peerSeeds, port, seedPeerOnly, ctx)
	}
	if p2pOpt != common.P2POptLibp2p {
		portOld := viper.GetInt(common.CfgP2PPort)
		peerSeedsOld := strings.FieldsFunc(viper.GetString(common.CfgP2PSeeds), f)
		networkOld = newMessengerOld(privKey, peerSeedsOld, portOld, ctx)
	}

	params := &node.Params{
		ChainID:             root.ChainID,
		PrivateKey:          privKey,
		Root:                root,
		NetworkOld:          networkOld,
		Network:             network,
		DB:                  db,
		SnapshotPath:        snapshotPath,
		ChainImportDirPath:  chainImportDirPath,
		ChainCorrectionPath: chainCorrectionPath,
	}

	n := node.NewNode(params)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	done := make(chan struct{})
	go func() {
		<-c
		signal.Stop(c)
		cancel()
		network.Stop()
		// Wait at most 5 seconds before forcefully shutting down.
		<-time.After(time.Duration(5) * time.Second)
		close(done)
	}()

	n.Start(ctx)

	if viper.GetBool(common.CfgProfEnabled) {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if viper.GetBool(common.CfgForceGCEnabled) {
		go memoryCleanupRoutine()
	}

	go func() {
		n.Wait()
		close(done)
	}()

	<-done
	log.Infof("")
	log.Infof("Graceful exit.")
	printExitBanner()
}

func loadOrCreateKey() (*crypto.PrivateKey, error) {
	keyPath := viper.GetString(common.CfgKeyPath)
	if keyPath == "" {
		keyPath = cfgPath
	}

	keysDir := path.Join(keyPath, "key")
	keystore, err := ks.NewKeystoreEncrypted(keysDir, ks.StandardScryptN, ks.StandardScryptP)
	if err != nil {
		log.Fatalf("Failed to create key store: %v", err)
	}
	addresses, err := keystore.ListKeyAddresses()
	if err != nil {
		log.Fatalf("Failed to get key address: %v", err)
	}

	numAddrs := len(addresses)
	if numAddrs > 1 {
		return nil, fmt.Errorf("Multiple encrypted keys detected under %v. Please keep only one key.", path.Join(keysDir, "encrypted"))
	}

	printWelcomeBanner()

	var password string
	var nodeAddrss common.Address
	if numAddrs == 0 {
		if len(nodePassword) != 0 {
			password = nodePassword
		} else {
			fmt.Println("")
			fmt.Println("You are launching the Theta Node for the first time. Welcome and please follow the instructions to setup the node.")
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

			fmt.Println("")
			fmt.Println("-----------------------------------------------------------------------------------------------------")
			fmt.Println("IMPORTANT: Please store your password securely. You will need it each time you launch the Theta node.")
			fmt.Println("-----------------------------------------------------------------------------------------------------")
			fmt.Println("")

			// fmt.Println("Please press enter to continue..")
			// utils.GetConfirmation()

			password = firstPassword
		}

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

		printCountdown()

	} else {
		prompt := fmt.Sprintf("Please enter the password to launch the Theta node: ")
		if len(nodePassword) != 0 {
			password = nodePassword
		} else {
			password, err = utils.GetPassword(prompt)
		}
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

func newMessenger(privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int, seedPeerOnly bool, ctx context.Context) *msgl.Messenger {
	log.WithFields(log.Fields{
		"pubKey":  fmt.Sprintf("%v", privKey.PublicKey().ToBytes()),
		"address": fmt.Sprintf("%v", privKey.PublicKey().Address()),
	}).Info("Using key:")
	msgrConfig := msgl.GetDefaultMessengerConfig()
	messenger, err := msgl.CreateMessenger(privKey.PublicKey(), seedPeerNetAddresses, port, seedPeerOnly, msgrConfig, true, ctx)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to create Messenger instance.")
	}
	return messenger
}

func newMessengerOld(privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int, ctx context.Context) *msg.Messenger {
	log.WithFields(log.Fields{
		"pubKey":  fmt.Sprintf("%v", privKey.PublicKey().ToBytes()),
		"address": fmt.Sprintf("%v", privKey.PublicKey().Address()),
	}).Info("Using key")
	msgrConfig := msg.GetDefaultMessengerConfig()
	msgrConfig.SetAddressBookFilePath(path.Join(cfgPath, "addrbook.json"))
	messenger, err := msg.CreateMessenger(privKey, seedPeerNetAddresses, port, msgrConfig)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Fatal("Failed to create Messenger instance")
	}
	return messenger
}

func printCountdown() {
	for i := 10; i >= 0; i-- {
		fmt.Printf("\rLaunching Theta to da moon: %d...", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Printf("\n\n")
}

func printWelcomeBanner() {
	fmt.Println("")
	fmt.Println("")
	fmt.Println(" ######################################################### ")
	fmt.Println("#                                                         #")
	fmt.Println("#  _    _      _ _         _______ _          _           #")
	fmt.Println("#  | |  | |    | | |       |__   __| |        | |         #")
	fmt.Println("#  | |__| | ___| | | ___      | |  | |__   ___| |_ __ _   #")
	fmt.Println("#  |  __  |/ _ \\ | |/ _ \\     | |  | '_ \\ / _ \\ __/ _` |  #")
	fmt.Println("#  | |  | |  __/ | | (_) |    | |  | | | |  __/ || (_| |  #")
	fmt.Println("#  |_|  |_|\\___|_|_|\\___/     |_|  |_| |_|\\___|\\__\\__,_|  #")
	fmt.Println("#                                                         #")
	fmt.Println("#                                                         #")
	fmt.Println(" ######################################################### ")
	fmt.Println("")
	fmt.Println("")
	fmt.Printf("Version %v, GitHash %s\nBuilt at %s\n", version.Version, version.GitHash, version.Timestamp)
	fmt.Println("")
}

func printExitBanner() {
	fmt.Println("")
	fmt.Println("")
	fmt.Println(" #################################################### ")
	fmt.Println("#                                                    #")
	fmt.Println("#  ____               _______ _          _           #")
	fmt.Println("#  |  _ \\             |__   __| |        | |         #")
	fmt.Println("#  | |_) |_   _  ___     | |  | |__   ___| |_ __ _   #")
	fmt.Println("#  |  _ <| | | |/ _ \\    | |  | '_ \\ / _ \\ __/ _` |  #")
	fmt.Println("#  | |_) | |_| |  __/    | |  | | | |  __/ || (_| |  #")
	fmt.Println("#  |____/ \\__, |\\___|    |_|  |_| |_|\\___|\\__\\__,_|  #")
	fmt.Println("#          __/ |                                     #")
	fmt.Println("#         |___/                                      #")
	fmt.Println("#                                                    #")
	fmt.Println(" #################################################### ")
	fmt.Println("")
	fmt.Println("")
}

// memoryCleanupRoutine peridically forces memory garbage collection.
func memoryCleanupRoutine() {
	var m runtime.MemStats
	t := time.NewTicker(30 * time.Second)
	for {
		<-t.C

		runtime.ReadMemStats(&m)
		log.Debugf("Memory usage: Alloc = %.3f MiB\tTotalAlloc = %.3f MiB\tSys = %.3f MiB\tNumGC = %v"+
			"\tStackInuse = %.3f MiB\tStackSys = %.3f MiB\tHeapInuse = %.3f MiB\tHeapSys = %.3f MiB\n",
			util.BToMb(m.Alloc), util.BToMb(m.TotalAlloc), util.BToMb(m.Sys), m.NumGC, util.BToMb(m.StackInuse),
			util.BToMb(m.StackSys), util.BToMb(m.HeapInuse), util.BToMb(m.HeapSys))

		runtime.GC()
	}

}
