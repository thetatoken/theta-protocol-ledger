package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/snapshot"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"
)

func handleError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: import_chain -chain=<chain_id> -config=<path_to_config_home> -snapshot=<path_to_snapshot_file> -chain_import=<path_to chain_files_directory>")
}

func main() {
	chainPtr := flag.String("chain", "", "chain id")
	configPathPtr := flag.String("config", "", "path to theta config home")
	snapshotPathPtr := flag.String("snapshot", "", "path to snapshot file")
	chainImportDirPathPtr := flag.String("chain_import", "", "path to chain files directory")

	flag.Parse()

	chainID := *chainPtr
	configPath := *configPathPtr
	snapshotPath := *snapshotPathPtr
	chainImportDirPath := *chainImportDirPathPtr

	mainDBPath := path.Join(configPath, "db", "main")
	refDBPath := path.Join(configPath, "db", "ref")
	db, _ := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)

	initConfig(configPath)

	root := core.NewBlock()
	if chainID == "" {
		root.ChainID = core.MainnetChainID
	} else {
		root.ChainID = chainID
	}
	store := kvstore.NewKVStore(db)
	chain := blockchain.NewChain(root.ChainID, store, root)

	_, err := snapshot.ValidateSnapshot(snapshotPath, chainImportDirPath, "")
	if err != nil {
		log.Fatalf("Snapshot validation failed, err: %v", err)
	}
	if _, _, err := snapshot.ImportSnapshot(snapshotPath, chainImportDirPath, "", chain, db, nil); err != nil {
		log.Fatalf("Failed to load snapshot: %v, err: %v", snapshotPath, err)
	}

	os.Exit(0)
}

func initConfig(cfgPath string) {
	viper.AddConfigPath(cfgPath)

	// Search config (without extension).
	viper.SetConfigName("config")

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
