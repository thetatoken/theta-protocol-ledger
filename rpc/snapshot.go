package rpc

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thetatoken/ukulele/store/database"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/kvstore"
	"github.com/thetatoken/ukulele/store/treestore"
)

const (
	LatestSnapshot = "theta_snapshot-latest"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCServer) GenSnapshot(r *http.Request, args *GenSnapshotArgs, result *GenSnapshotResult) (err error) {
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
	var currBlock *core.ExtendedBlock
	for _, height := range hl {
		if currBlock == nil || height > currBlock.Height {
			blocks := t.chain.FindBlocksByHeight(height)
			for _, block := range blocks {
				if block.Status.IsFinalized() {
					var finalizedChind *core.ExtendedBlock
					for {
						for _, h := range block.Children {
							b, err := t.chain.FindBlock(h)
							if err != nil {
								log.Errorf("Failed to get block %v", err)
								return err
							}
							if b.Status.IsFinalized() {
								finalizedChind = b
								break
							}
						}
						if block.Status == core.BlockStatusDirectlyFinalized {
							metadata.BlocksWithValidatorChange = append(metadata.BlocksWithValidatorChange, core.DirectlyFinalizedBlockPair{First: *block, Second: *finalizedChind})
							addVotes(st, metadata, block.Hash())
							addVotes(st, metadata, finalizedChind.Hash())

							currBlock = block
							break
						} else {
							block = finalizedChind
						}
					}
					break
				}
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

	for i, blockPair := range metadata.BlocksWithValidatorChange {
		storeView := state.NewStoreView(blockPair.First.Height, blockPair.First.StateHash, db)
		writeStoreView(storeView, false, writer, db, 2*i)
		storeView = state.NewStoreView(blockPair.Second.Height, blockPair.Second.StateHash, db)
		writeStoreView(storeView, false, writer, db, 2*i+1)
	}

	writeStoreView(sv, true, writer, db, len(metadata.BlocksWithValidatorChange))
	return
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

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer, db database.Database, sequence int) {
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err := writeRecord(writer, k, v, nil, sequence)
		if err != nil {
			panic(err)
		}

		if needAccountStorage && strings.HasPrefix(k.String(), "ls/a/") {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				log.Errorf("Failed to parse account for %v", []byte(v))
				panic(err)
			}
			storage := treestore.NewTreeStore(account.Root, db)
			storage.Traverse(nil, func(ak, av common.Bytes) bool {
				err = writeRecord(writer, ak, av, account.Root.Bytes(), sequence)
				if err != nil {
					panic(err)
				}
				return true
			})
		}
		return true
	})
	writer.Flush()
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

func writeRecord(writer *bufio.Writer, k, v, r common.Bytes, s int) error {
	raw, err := rlp.EncodeToBytes(core.SnapshotRecord{K: k, V: v, R: r, S: s})
	if err != nil {
		log.Error("Failed to encode storage record")
		return err
	}
	// write length first
	_, err = writer.Write(itobs(uint64(len(raw))))
	if err != nil {
		log.Error("Failed to write storage record length")
		return err
	}
	// write record itself
	_, err = writer.Write(raw)
	if err != nil {
		log.Error("Failed to write storage record")
		return err
	}
	return nil
}

func itobs(val uint64) []byte {
	arr := make([]byte, 8)
	for i := 0; i < 8; i++ {
		arr[i] = byte(val % 10)
		val /= 10
	}
	return arr
}
