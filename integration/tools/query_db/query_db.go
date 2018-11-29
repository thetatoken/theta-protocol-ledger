package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/store/database/backend"
	"github.com/thetatoken/ukulele/store/kvstore"
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
	configPathPtr := flag.String("config", "", "path to ukuele config home")
	queryTypePtr := flag.String("type", "block", "type of object to query")
	hashStrPtr := flag.String("hash", "", "hash of the object")
	heightStrPtr := flag.String("height", "", "block height")

	flag.Parse()

	configPath := *configPathPtr
	queryType := *queryTypePtr
	hashStr := *hashStrPtr
	heightStr := *heightStrPtr

	checkpoint, err := consensus.LoadCheckpoint(path.Join(configPath, "genesis"))
	handleError(err)
	mainDBPath := path.Join(configPath, "db", "main")
	refDBPath := path.Join(configPath, "db", "ref")
	db, err := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)
	root := checkpoint.FirstBlock
	store := kvstore.NewKVStore(db)
	chain := blockchain.NewChain(root.ChainID, store, root)

	if queryType == "block" {
		if hashStr != "" {
			hash := common.HexToHash(hashStr)
			block, err := chain.FindBlock(hash)
			handleError(err)
			jsonStr, err := json.MarshalIndent(block, "", "  ")
			handleError(err)
			fmt.Printf("%s\n", jsonStr)
			os.Exit(0)
		}

		if heightStr != "" {
			height, err := strconv.ParseUint(heightStr, 10, 64)
			handleError(err)
			blocks := chain.FindBlocksByHeight(height)
			jsonStr, err := json.MarshalIndent(blocks, "", "  ")
			handleError(err)
			fmt.Printf("%s\n", jsonStr)
			os.Exit(0)
		}
	}

	printUsage()
}
