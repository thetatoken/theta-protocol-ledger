package rpc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/treestore"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/kvstore"
)

const (
	LatestSnapshot = "theta_snapshot-latest"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCService) GenSnapshot(args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
	metadata := &core.SnapshotMetadata{}

	stub := t.consensus.GetSummary()
	lastFinalizedBlock, err := t.chain.FindBlock(stub.LastFinalizedBlock)
	if err != nil {
		log.Errorf("Failed to get block %v, %v", stub.LastFinalizedBlock, err)
		return err
	}
	sv, err := t.ledger.GetFinalizedSnapshot()
	if err != nil {
		return err
	}
	if sv.Height() != lastFinalizedBlock.Height {
		return fmt.Errorf("Last finalized block height don't match %v != %v", sv.Height(), lastFinalizedBlock.Height)
	}
	if sv.Hash() != lastFinalizedBlock.StateHash {
		return fmt.Errorf("Last finalized block state hash don't match %v != %v", sv.Hash(), lastFinalizedBlock.StateHash)
	}
	metadata.Blockheader = *(lastFinalizedBlock.BlockHeader)

	metadata.Votes = []core.Vote{}
	db := t.ledger.State().DB()
	st := consensus.NewState(kvstore.NewKVStore(db), t.chain)
	addVotes(st, metadata, metadata.Blockheader.Hash())

	hl := sv.GetStakeTransactionHeightList().Heights
	for _, height := range hl {
		if height >= lastFinalizedBlock.Height-1 {
			break
		}
		blocks := t.chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsDirectlyFinalized() {
				var child, grandChild core.ExtendedBlock
				b, err := getFinalizedChild(block, t.chain)
				if err != nil {
					return err
				}
				if b != nil {
					child = *b
					b, err = getFinalizedChild(b, t.chain)
					if err != nil {
						return err
					}
					if b != nil {
						grandChild = *b
					} else {
						return fmt.Errorf("Can't find finalized child block")
					}
				} else {
					return fmt.Errorf("Can't find finalized child block")
				}

				// if child.HCC != block.Hash() || grandChild.HCC != child.Hash() { //TODO: change of HCC struct
				// 	return fmt.Errorf("Invalid block HCC for validator set changes")
				// }

				metadata.BlocksWithValidatorChange = append(metadata.BlocksWithValidatorChange, core.DirectlyFinalizedBlockTrio{First: *block, Second: child, Third: grandChild})
				break
			}
		}
	}

	currentTime := time.Now().UTC()
	file, err := os.Create("theta_snapshot-" + sv.Hash().String() + "-" + strconv.Itoa(int(sv.Height())) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = writeMetadata(writer, metadata)
	if err != nil {
		return err
	}

	for _, pair := range metadata.BlocksWithValidatorChange {
		storeView := state.NewStoreView(pair.First.Height, pair.First.StateHash, db)
		writeStoreView(storeView, false, writer, db)
	}
	writeStoreView(sv, true, writer, db)

	return
}

func getFinalizedChild(block *core.ExtendedBlock, chain *blockchain.Chain) (*core.ExtendedBlock, error) {
	for _, h := range block.Children {
		b, err := chain.FindBlock(h)
		if err != nil {
			log.Errorf("Failed to get block %v", err)
			return nil, err
		}
		if b.Status.IsFinalized() {
			return b, nil
		}
	}
	return nil, nil
}

func addVotes(st *consensus.State, metadata *core.SnapshotMetadata, hash common.Hash) error {
	voteSet, err := st.GetVoteSetByBlock(hash)
	if err != nil {
		log.Errorf("Failed to get vote set for block %v, %v", hash, err)
		return err
	}
	metadata.Votes = append(metadata.Votes, voteSet.Votes()...)
	return nil
}

func writeMetadata(writer *bufio.Writer, metadata *core.SnapshotMetadata) error {
	raw, err := rlp.EncodeToBytes(*metadata)
	if err != nil {
		log.Error("Failed to encode snapshot metadata")
		return err
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write snapshot metadata length")
		return err
	}
	// write metadata itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write snapshot metadata")
		return err
	}

	meta := &core.SnapshotMetadata{}
	rlp.DecodeBytes(raw, meta)

	return nil
}

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer, db database.Database) {
	height := itobs(sv.Height())
	err := writeRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = writeRecord(writer, k, v)
		if err != nil {
			panic(err)
		}
		if needAccountStorage && strings.HasPrefix(k.String(), "ls/a/") {
			err = writeRecord(writer, []byte{core.SVStart}, height)
			if err != nil {
				panic(err)
			}
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				log.Errorf("Failed to parse account for %v", []byte(v))
				panic(err)
			}
			storage := treestore.NewTreeStore(account.Root, db)
			storage.Traverse(nil, func(ak, av common.Bytes) bool {
				err = writeRecord(writer, ak, av)
				if err != nil {
					panic(err)
				}
				return true
			})
			err = writeRecord(writer, []byte{core.SVEnd}, height)
			if err != nil {
				panic(err)
			}
		}
		return true
	})
	err = writeRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func writeRecord(writer *bufio.Writer, k, v common.Bytes) error {
	record := core.SnapshotRecord{K: k, V: v}
	raw, err := rlp.EncodeToBytes(record)
	if err != nil {
		return fmt.Errorf("Failed to encode storage record, %v", err)
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		return fmt.Errorf("Failed to write storage record length, %v", err)
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		return fmt.Errorf("Failed to write storage record, %v", err)
	}
	err = writer.Flush()
	if err != nil {
		return fmt.Errorf("Failed to flush storage record, %v", err)
	}
	return nil
}

func itobs(val uint64) []byte {
	arr := make([]byte, 8)
	binary.LittleEndian.PutUint64(arr, val)
	// for i := 0; i < 8; i++ {
	// 	arr[i] = byte(val % 10)
	// 	val /= 10
	// }
	return arr
}
