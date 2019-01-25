package rpc

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
	"github.com/thetatoken/theta/store/treestore"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
)

const (
	LatestSnapshot = "theta_snapshot-latest"
)

type GenSnapshotArgs struct {
}

type GenSnapshotResult struct {
}

func (t *ThetaRPCService) GenSnapshot(args *GenSnapshotArgs, result *GenSnapshotResult) error {
	metadata := &core.SnapshotMetadata{}

	db := t.ledger.State().DB()
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

	hl := sv.GetStakeTransactionHeightList().Heights
	for _, height := range hl {
		if height >= lastFinalizedBlock.Height-1 {
			break
		}
		blocks := t.chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsDirectlyFinalized() {
				var child, grandChild core.BlockHeader
				b, err := getFinalizedChild(block, t.chain)
				if err != nil {
					return err
				}
				if b != nil {
					child = *b.BlockHeader
					b, err = getFinalizedChild(b, t.chain)
					if err != nil {
						return err
					}
					if b != nil {
						grandChild = *b.BlockHeader
					} else {
						return fmt.Errorf("Can't find finalized child block")
					}
				} else {
					return fmt.Errorf("Can't find finalized child block")
				}

				if child.HCC.BlockHash != block.Hash() || grandChild.HCC.BlockHash != child.Hash() {
					return fmt.Errorf("Invalid block HCC link for validator set changes")
				}
				if grandChild.HCC.Votes.IsEmpty() {
					return fmt.Errorf("Missing block HCC votes for validator set changes")
				}
				for _, vote := range grandChild.HCC.Votes.Votes() {
					if vote.Block != child.Hash() {
						return fmt.Errorf("Invalid block HCC votes for validator set changes")
					}
				}

				storeView := state.NewStoreView(block.Height, block.StateHash, db)
				vcpProof, err := proveVCP(storeView, state.ValidatorCandidatePoolKey())
				if err != nil {
					return fmt.Errorf("Failed to get VCP Proof")
				}
				metadata.BlockTrios = append(metadata.BlockTrios, core.SnapshotBlockTrio{First: core.SnapshotFirstBlock{Header: *block.BlockHeader, Proof: *vcpProof}, Second: child, Third: core.SnapshotThirdBlock{Header: grandChild, Votes: grandChild.HCC.Votes.Votes()}})
				break
			}
		}
	}

	parentBlock, err := t.chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return err
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, t.chain)
	if err != nil {
		return err
	}
	var votes []core.Vote
	if childBlock.BlockHeader.HCC.Votes.IsEmpty() {
		st := consensus.NewState(kvstore.NewKVStore(db), t.chain)
		voteSet, err := st.GetVoteSetByBlock(lastFinalizedBlock.Hash())
		if err != nil {
			return err
		}
		votes = voteSet.Votes()
	} else {
		votes = childBlock.BlockHeader.HCC.Votes.Votes()
	}

	storeView := state.NewStoreView(parentBlock.Height, parentBlock.StateHash, db)
	vcpProof, err := proveVCP(storeView, state.ValidatorCandidatePoolKey())
	if err != nil {
		return fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.BlockTrios = append(metadata.BlockTrios, core.SnapshotBlockTrio{First: core.SnapshotFirstBlock{Header: *parentBlock.BlockHeader, Proof: *vcpProof}, Second: *lastFinalizedBlock.BlockHeader, Third: core.SnapshotThirdBlock{Header: *childBlock.BlockHeader, Votes: votes}})

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

	// for _, trio := range metadata.BlockTrios {
	// 	storeView := state.NewStoreView(trio.First.Height, trio.First.StateHash, db)
	// 	writeStoreView(storeView, false, writer, db)
	// }
	writeStoreView(sv, true, writer, db)

	return nil
}

func proveVCP(sv *state.StoreView, vcpKey []byte) (*core.VCPProof, error) {
	vp := &core.VCPProof{}
	err := sv.ProveVCP(vcpKey, vp)
	if err != nil {
		return nil, err
	}
	return vp, nil
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

func getAtLeastCommittedChild(block *core.ExtendedBlock, chain *blockchain.Chain) (*core.ExtendedBlock, error) {
	for _, h := range block.Children {
		b, err := chain.FindBlock(h)
		if err != nil {
			log.Errorf("Failed to get block %v", err)
			return nil, err
		}
		if b.Status.IsFinalized() || b.Status.IsCommitted() {
			return b, nil
		}
	}
	return nil, nil
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

func writeVCPBranch(sv *state.StoreView, writer *bufio.Writer) {
	height := itobs(sv.Height())
	err := writeRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(state.ValidatorCandidatePoolKey(), func(k, v common.Bytes) bool {
		err = writeRecord(writer, k, v)
		if err != nil {
			panic(err)
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
	record := core.SnapshotTrieRecord{K: k, V: v}
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
	return arr
}
