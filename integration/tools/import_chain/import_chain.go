package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"

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
	fmt.Println("Usage: query_db -config=<path_to_config_home> -type=block -hash=<hash> -height=<height>")
}

func main() {
	configPathPtr := flag.String("config", "", "path to theta config home")
	snapshotPathPtr := flag.String("snapshot", "", "path to snapshot file")
	chainImportDirPathPtr := flag.String("chain_import", "", "path to chain files directory")

	flag.Parse()

	configPath := *configPathPtr
	snapshotPath := *snapshotPathPtr
	chainImportDirPath := *chainImportDirPathPtr

	mainDBPath := path.Join(configPath, "db", "main")
	refDBPath := path.Join(configPath, "db", "ref")
	db, _ := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)

	root := core.NewBlock()
	// TODO: need to setup root.ChainID
	store := kvstore.NewKVStore(db)
	chain := blockchain.NewChain(root.ChainID, store, root)

	_, err := snapshot.ValidateSnapshot(snapshotPath, chainImportDirPath)
	if err != nil {
		log.Fatalf("Snapshot validation failed, err: %v", err)
	}
	if _, err := snapshot.ImportSnapshot(snapshotPath, chainImportDirPath, chain, db); err != nil {
		log.Fatalf("Failed to load snapshot: %v, err: %v", snapshotPath, err)
	}

	os.Exit(0)
}
