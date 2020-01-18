package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"
	"github.com/thetatoken/theta/store/treestore"
)

func handleError(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: dump_storeview -config=<path_to_config_home> -height=<height> -state_hash=<state_hash>")
}

func main() {
	configPathPtr := flag.String("config", "", "path to ukuele config home")
	heightPtr := flag.Uint64("height", 0, "height of storeview block")
	stateHashPtr := flag.String("state_hash", "", "hash of state root")
	flag.Parse()
	configPath := *configPathPtr
	height := *heightPtr
	stateHashStr := *stateHashPtr
	heightStr := strconv.FormatUint(height, 10)

	mainDBPath := path.Join(configPath, "db", "main")
	refDBPath := path.Join(configPath, "db", "ref")
	db, err := backend.NewLDBDatabase(mainDBPath, refDBPath, 256, 0)
	handleError(err)

	var sv *state.StoreView
	var filename string
	if len(stateHashStr) != 0 {
		stateHash := common.HexToHash(stateHashStr)
		sv = state.NewStoreView(0, stateHash, db)
		filename = "theta_storeview-" + stateHashStr + ".json"
	} else {
		root := core.NewBlock()
		store := kvstore.NewKVStore(db)
		chain := blockchain.NewChain(root.ChainID, store, root)

		var finalizedBlock *core.ExtendedBlock
		blocks := chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsFinalized() {
				finalizedBlock = block
				break
			}
		}
		if finalizedBlock == nil {
			handleError(fmt.Errorf("Finalized block not found for height %v", height))
		}

		sv = state.NewStoreView(finalizedBlock.Height, finalizedBlock.StateHash, db)
		filename = "theta_storeview-" + heightStr + ".json"
	}

	dumpPath := path.Join(configPath, filename)
	file, err := os.Create(dumpPath)
	if err != nil {
		handleError(err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writeSV(sv, writer, db, heightStr)

	fmt.Printf("Output file: %v\n", filename)
	os.Exit(0)
}

func writeSV(sv *state.StoreView, writer *bufio.Writer, db database.Database, heightStr string) {
	jsonString := "{\n"
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		jsonString += fmt.Sprintf("\"%v\":%v,\n", common.Bytes2Hex(k), fmtValue(v))
		if bytes.HasPrefix(k, common.Bytes("ls/a")) {
			account := &types.Account{}
			err := types.FromBytes(v, account)
			if err != nil {
				panic(err)
			}
			if account.Root != (common.Hash{}) {
				jsonString += fmt.Sprintf("\"%v-storage\": {", common.Bytes2Hex(k))
				storage := treestore.NewTreeStore(account.Root, db)
				storage.Traverse(nil, func(ak, av common.Bytes) bool {
					jsonString += common.Bytes2Hex(ak) + ":" + common.Bytes2Hex(av) + ",\n"
					return true
				})
				jsonString += fmt.Sprintf("\"account\":\"%v\"}", common.Bytes2Hex(k))
			}
		}
		return true
	})
	jsonString += "\"height\": " + heightStr + "\n}"
	writer.WriteString(jsonString)
	writer.Flush()
}

func fmtValue(value common.Bytes) string {
	account := types.Account{}
	err := rlp.DecodeBytes(value, &account)
	if err == nil {
		j, err := json.Marshal(account)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%v", string(j))
	}

	splitRule := types.SplitRule{}
	err = rlp.DecodeBytes(value, &splitRule)
	if err == nil {
		j, err := json.Marshal(splitRule)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%v", string(j))
	}

	vcp := core.ValidatorCandidatePool{}
	err = rlp.DecodeBytes(value, &vcp)
	if err == nil {
		j, err := json.Marshal(vcp)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%v", string(j))
	}

	hl := types.HeightList{}
	err = rlp.DecodeBytes(value, &hl)
	if err == nil {
		j, err := json.Marshal(hl)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%v", string(j))
	}

	bbhie := blockchain.BlockByHeightIndexEntry{}
	err = rlp.DecodeBytes(value, &bbhie)
	if err == nil {
		j, err := json.Marshal(bbhie)
		if err != nil {
			panic(err)
		}
		return fmt.Sprintf("%v", string(j))
	}

	return fmt.Sprintf("\"%v\"", common.Bytes2Hex(value))
}
