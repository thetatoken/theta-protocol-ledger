package snapshot

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	cns "github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
	"github.com/thetatoken/theta/store/treestore"
)

func ExportSnapshot(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain) error {
	metadata := &core.SnapshotMetadata{}

	stub := consensus.GetSummary()
	lastFinalizedBlock, err := chain.FindBlock(stub.LastFinalizedBlock)
	if err != nil {
		logger.Errorf("Failed to get block %v, %v", stub.LastFinalizedBlock, err)
		return err
	}

	sv := state.NewStoreView(lastFinalizedBlock.Height, lastFinalizedBlock.BlockHeader.StateHash, db)

	var genesisBlockHeader *core.BlockHeader
	kvStore := kvstore.NewKVStore(db)
	hl := sv.GetStakeTransactionHeightList().Heights
	for _, height := range hl {
		// check kvstore first
		blockTrio := &core.SnapshotBlockTrio{}
		blockTrioKey := []byte(core.BlockTrioStoreKeyPrefix + strconv.FormatUint(height, 10))
		err := kvStore.Get(blockTrioKey, blockTrio)
		if err == nil {
			metadata.ProofTrios = append(metadata.ProofTrios, *blockTrio)
			if height == core.GenesisBlockHeight {
				genesisBlockHeader = &blockTrio.Second.Header
			}
			continue
		}

		if height == core.GenesisBlockHeight {
			blocks := chain.FindBlocksByHeight(core.GenesisBlockHeight)
			genesisBlock := blocks[0]
			genesisBlockHeader = genesisBlock.BlockHeader
			metadata.ProofTrios = append(metadata.ProofTrios,
				core.SnapshotBlockTrio{
					First:  core.SnapshotFirstBlock{},
					Second: core.SnapshotSecondBlock{Header: *genesisBlock.BlockHeader},
					Third:  core.SnapshotThirdBlock{},
				})
		} else {
			blocks := chain.FindBlocksByHeight(height)
			foundDirectlyFinalizedBlock := false
			for _, block := range blocks {
				if block.Status.IsDirectlyFinalized() {
					var child, grandChild core.BlockHeader
					b, err := getFinalizedChild(block, chain)
					if err != nil {
						return err
					}
					if b != nil {
						child = *b.BlockHeader
						b, err = getFinalizedChild(b, chain)
						if err != nil {
							return err
						}
						if b != nil {
							grandChild = *b.BlockHeader
						} else {
							return fmt.Errorf("Can't find finalized grandchild block. " +
								"Likely the last finalized block also contains stake change transactions. " +
								"Please try again in 30 seconds.")
						}
					} else {
						return fmt.Errorf("Can't find finalized child block. " +
							"Likely the last finalized block also contains stake change transactions. " +
							"Please try again in 30 seconds.")
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
					metadata.ProofTrios = append(metadata.ProofTrios,
						core.SnapshotBlockTrio{
							First:  core.SnapshotFirstBlock{Header: *block.BlockHeader, Proof: *vcpProof},
							Second: core.SnapshotSecondBlock{Header: child},
							Third:  core.SnapshotThirdBlock{Header: grandChild},
						})
					foundDirectlyFinalizedBlock = true
					break
				}
			}
			if !foundDirectlyFinalizedBlock {
				return fmt.Errorf("Finalized block not found for height %v", height)
			}
		}
	}

	parentBlock, err := chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return fmt.Errorf("Failed to find last finalized block's parent, %v", err)
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, chain)
	if err != nil {
		return fmt.Errorf("Failed to find last finalized block's committed child, %v", err)
	}

	if lastFinalizedBlock.HCC.BlockHash != parentBlock.Hash() {
		return fmt.Errorf("Parent block hash mismatch: %v vs %v", lastFinalizedBlock.HCC.BlockHash, parentBlock.Hash())
	}

	if childBlock.HCC.BlockHash != lastFinalizedBlock.Hash() {
		return fmt.Errorf("Finalized block hash mismatch: %v vs %v", childBlock.HCC.BlockHash, lastFinalizedBlock.Hash())
	}

	st := cns.NewState(kvstore.NewKVStore(db), chain)
	childVoteSet, err := st.GetVoteSetByBlock(childBlock.Hash())
	if err != nil {
		return fmt.Errorf("Failed to get child block's votes, %v", err)
	}

	vcpProof, err := proveVCP(parentBlock, db)
	if err != nil {
		return fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.TailTrio = core.SnapshotBlockTrio{
		First:  core.SnapshotFirstBlock{Header: *parentBlock.BlockHeader, Proof: *vcpProof},
		Second: core.SnapshotSecondBlock{Header: *lastFinalizedBlock.BlockHeader},
		Third:  core.SnapshotThirdBlock{Header: *childBlock.BlockHeader, VoteSet: childVoteSet},
	}

	currentTime := time.Now().UTC()
	file, err := os.Create("theta_snapshot-" + sv.Hash().String() + "-" + strconv.FormatUint(sv.Height(), 10) + "-" + currentTime.Format("2006-01-02"))
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return err
	}

	genesisSV := state.NewStoreView(genesisBlockHeader.Height, genesisBlockHeader.StateHash, db)
	writeStoreView(genesisSV, false, writer, db)
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
			logger.Errorf("Failed to get block %v", err)
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
			logger.Errorf("Failed to get block %v", err)
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
				logger.Errorf("Failed to parse account for %v", []byte(v))
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
