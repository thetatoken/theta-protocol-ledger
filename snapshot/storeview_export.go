package snapshot

import (
	"bufio"
	"bytes"
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
	"github.com/thetatoken/theta/store/treestore"
)

func DumpSV(db database.Database, chain *blockchain.Chain, dumpDir string, height uint64) (string, error) {
	var finalizedBlock *core.ExtendedBlock
	blocks := chain.FindBlocksByHeight(height)
	for _, block := range blocks {
		if block.Status.IsFinalized() {
			finalizedBlock = block
			break
		}
	}
	if finalizedBlock == nil {
		return "", fmt.Errorf("Finalized block not found for height %v", height)
	}

	sv := state.NewStoreView(finalizedBlock.Height, finalizedBlock.StateHash, db)

	heightStr := strconv.FormatUint(height, 10)
	filename := "theta_sv_dump-" + heightStr
	dumpPath := path.Join(dumpDir, filename)
	file, err := os.Create(dumpPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	writeSV(sv, writer, db, heightStr)

	return filename, nil
}

func writeSV(sv *state.StoreView, writer *bufio.Writer, db database.Database, heightStr string) {
	// kvStore := kvstore.NewKVStore(db)
	jsonString := "{\n"
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		jsonString += fmt.Sprintf("\"%v\":%v,\n", common.Bytes2Hex(k), fmtValue(v))
		if bytes.HasPrefix(k, common.Bytes("ls/a")) {
			account := &types.Account{}
			err := types.FromBytes(v, account)
			if err != nil {
				logger.Errorf("Failed to parse account for %v", v)
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
		return fmt.Sprintf("%v", account.JsonString())
	}

	splitRule := types.SplitRule{}
	err = rlp.DecodeBytes(value, &splitRule)
	if err == nil {
		return fmt.Sprintf("%v", splitRule.JsonString())
	}

	vcp := core.ValidatorCandidatePool{}
	err = rlp.DecodeBytes(value, &vcp)
	if err == nil {
		return fmt.Sprintf("%v", vcp.JsonString())
	}

	hl := types.HeightList{}
	err = rlp.DecodeBytes(value, &hl)
	if err == nil {
		return fmt.Sprintf("%v", hl.JsonString())
	}

	bbhie := blockchain.BlockByHeightIndexEntry{}
	err = rlp.DecodeBytes(value, &bbhie)
	if err == nil {
		return fmt.Sprintf("%v", bbhie.JsonString())
	}

	return fmt.Sprintf("\"%v\"", common.Bytes2Hex(value))
}
