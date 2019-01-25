package rpc

import (
	"bufio"
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

	kvStore := kvstore.NewKVStore(db)
	hl := sv.GetStakeTransactionHeightList().Heights
	for _, height := range hl {
		if height >= lastFinalizedBlock.Height-1 {
			break
		}

		// check kvstore first
		blockTrio := &core.SnapshotBlockTrio{}
		err := kvStore.Get([]byte(core.BlockTrioStoreKeyPrefix+strconv.FormatUint(height, 64)), blockTrio)
		if err == nil {
			metadata.BlockTrios = append(metadata.BlockTrios, *blockTrio)
			continue
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
						return fmt.Errorf("Can't find finalized grandchild block")
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

				vcpProof, err := proveVCP(block, db)
				if err != nil {
					return fmt.Errorf("Failed to get VCP Proof")
				}
				metadata.BlockTrios = append(metadata.BlockTrios,
					core.SnapshotBlockTrio{
						First:  core.SnapshotFirstBlock{Header: *block.BlockHeader, Proof: *vcpProof},
						Second: core.SnapshotSecondBlock{Header: child},
						Third:  core.SnapshotThirdBlock{Header: grandChild},
					})
				break
			} else {
				return fmt.Errorf("Found a non-directly-finalized Stake changing block")
			}
		}
	}

	parentBlock, err := t.chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return fmt.Errorf("Failed to find last finalized block's parent, %v", err)
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, t.chain)
	if err != nil {
		return fmt.Errorf("Failed to find last finalized block's committed child, %v", err)
	}

	if lastFinalizedBlock.HCC.BlockHash != parentBlock.Hash() {
		return fmt.Errorf("Parent block hash mismatch: %v vs %v", lastFinalizedBlock.HCC.BlockHash, parentBlock.Hash())
	}

	if childBlock.HCC.BlockHash != lastFinalizedBlock.Hash() {
		return fmt.Errorf("Finalized block hash mismatch: %v vs %v", childBlock.HCC.BlockHash, lastFinalizedBlock.Hash())
	}

	st := consensus.NewState(kvstore.NewKVStore(db), t.chain)
	childVoteSet, err := st.GetVoteSetByBlock(childBlock.Hash())
	if err != nil {
		return fmt.Errorf("Failed to get child block's votes, %v", err)
	}

	vcpProof, err := proveVCP(parentBlock, db)
	if err != nil {
		return fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.BlockTrios = append(metadata.BlockTrios,
		core.SnapshotBlockTrio{
			First:  core.SnapshotFirstBlock{Header: *parentBlock.BlockHeader, Proof: *vcpProof},
			Second: core.SnapshotSecondBlock{Header: *lastFinalizedBlock.BlockHeader},
			Third:  core.SnapshotThirdBlock{Header: *childBlock.BlockHeader, VoteSet: childVoteSet},
		})

	currentTime := time.Now().UTC()
	file, err := os.Create("theta_snapshot-" + sv.Hash().String() + "-" + strconv.Itoa(int(sv.Height())) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return err
	}

	parentSV := state.NewStoreView(parentBlock.Height, parentBlock.StateHash, db)
	writeStoreView(parentSV, true, writer, db)
	writeStoreView(sv, true, writer, db)

	return nil
}

func proveVCP(block *core.ExtendedBlock, db database.Database) (*core.VCPProof, error) {
	sv := state.NewStoreView(block.Height, block.StateHash, db)
	vcpKey := state.ValidatorCandidatePoolKey()
	vp := &core.VCPProof{}
	err := sv.ProveVCP(vcpKey, vp)
	return vp, err
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

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer, db database.Database) {
	height := core.Itobytes(sv.Height())
	err := core.WriteRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}
	sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
		err = core.WriteRecord(writer, k, v)
		if err != nil {
			panic(err)
		}
		if needAccountStorage && strings.HasPrefix(k.String(), "ls/a/") {
			err = core.WriteRecord(writer, []byte{core.SVStart}, height)
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
				err = core.WriteRecord(writer, ak, av)
				if err != nil {
					panic(err)
				}
				return true
			})
			err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
			if err != nil {
				panic(err)
			}
		}
		return true
	})
	err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}
