package snapshot

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
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
	"github.com/thetatoken/theta/store/trie"
)

func ExportSnapshotV2(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain, snapshotDir string, height uint64) (string, error) {
	var lastFinalizedBlock *core.ExtendedBlock
	if height != 0 {
		blocks := chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsDirectlyFinalized() {
				lastFinalizedBlock = block
				break
			}
		}
		if lastFinalizedBlock == nil {
			return "", fmt.Errorf("Can't find finalized block at height %v", height)
		}
	} else {
		stub := consensus.GetSummary()
		var err error
		lastFinalizedBlock, err = chain.FindBlock(stub.LastFinalizedBlock)
		if err != nil {
			logger.Errorf("Failed to get block %v, %v", stub.LastFinalizedBlock, err)
			return "", err
		}
	}
	sv := state.NewStoreView(lastFinalizedBlock.Height, lastFinalizedBlock.BlockHeader.StateHash, db)

	currentTime := time.Now().UTC()
	filename := "theta_snapshot-" + strconv.FormatUint(sv.Height(), 10) + "-" + sv.Hash().String() + "-" + currentTime.Format("2006-01-02")
	snapshotPath := path.Join(snapshotDir, filename)
	file, err := os.Create(snapshotPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// --------------- Export the Header Section --------------- //

	snapshotHeader := &core.SnapshotHeader{
		Magic:   core.SnapshotHeaderMagic,
		Version: 2,
	}
	err = core.WriteSnapshotHeader(writer, snapshotHeader)
	if err != nil {
		return "", err
	}

	// ------------ Export the Last Checkpoint Section ------------- //

	lastFinalizedBlockHeight := lastFinalizedBlock.Height
	lastCheckpointHeight := common.LastCheckPointHeight(lastFinalizedBlockHeight)
	lastCheckpoint := &core.LastCheckpoint{}

	currHeight := lastFinalizedBlockHeight
	currBlock := lastFinalizedBlock
	for currHeight > lastCheckpointHeight {
		parentHash := currBlock.Parent
		currBlock, err = chain.FindBlock(parentHash)
		if err != nil {
			logger.Errorf("Failed to get intermediate block %v, %v", parentHash.Hex(), err)
			return "", err
		}
		lastCheckpoint.IntermediateHeaders = append(lastCheckpoint.IntermediateHeaders, currBlock.Block.BlockHeader)
		currHeight = currBlock.Height
	}

	lastCheckpointBlock := currBlock
	lastCheckpoint.CheckpointHeader = lastCheckpointBlock.BlockHeader

	err = core.WriteLastCheckpoint(writer, lastCheckpoint)
	if err != nil {
		return "", err
	}

	// -------------- Export the Metadata Section -------------- //

	metadata := &core.SnapshotMetadata{}
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
				genesisBlockHeader = blockTrio.Second.Header
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
					Second: core.SnapshotSecondBlock{Header: genesisBlock.BlockHeader},
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
						return "", err
					}
					if b != nil {
						child = *b.BlockHeader
						b, err = getFinalizedChild(b, chain)
						if err != nil {
							return "", err
						}
						if b != nil {
							grandChild = *b.BlockHeader
						} else {
							return "", fmt.Errorf("Can't find finalized grandchild block. " +
								"Likely the last finalized block also contains stake change transactions. " +
								"Please try again in 30 seconds.")
						}
					} else {
						return "", fmt.Errorf("Can't find finalized child block. " +
							"Likely the last finalized block also contains stake change transactions. " +
							"Please try again in 30 seconds.")
					}

					if child.HCC.BlockHash != block.Hash() || grandChild.HCC.BlockHash != child.Hash() {
						return "", fmt.Errorf("Invalid block HCC link for validator set changes")
					}
					if grandChild.HCC.Votes.IsEmpty() {
						return "", fmt.Errorf("Missing block HCC votes for validator set changes")
					}
					for _, vote := range grandChild.HCC.Votes.Votes() {
						if vote.Block != child.Hash() {
							return "", fmt.Errorf("Invalid block HCC votes for validator set changes")
						}
					}

					vcpProof, err := proveVCP(block, db)
					if err != nil {
						return "", fmt.Errorf("Failed to get VCP Proof")
					}
					metadata.ProofTrios = append(metadata.ProofTrios,
						core.SnapshotBlockTrio{
							First:  core.SnapshotFirstBlock{Header: block.BlockHeader, Proof: *vcpProof},
							Second: core.SnapshotSecondBlock{Header: &child},
							Third:  core.SnapshotThirdBlock{Header: &grandChild},
						})
					foundDirectlyFinalizedBlock = true
					break
				}
			}
			if !foundDirectlyFinalizedBlock {
				return "", fmt.Errorf("Finalized block not found for height %v", height)
			}
		}
	}

	parentBlock, err := chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's parent, %v", err)
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, chain)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's committed child, %v", err)
	}

	if lastFinalizedBlock.HCC.BlockHash != parentBlock.Hash() {
		return "", fmt.Errorf("Parent block hash mismatch: %v vs %v", lastFinalizedBlock.HCC.BlockHash, parentBlock.Hash())
	}

	if childBlock.HCC.BlockHash != lastFinalizedBlock.Hash() {
		return "", fmt.Errorf("Finalized block hash mismatch: %v vs %v", childBlock.HCC.BlockHash, lastFinalizedBlock.Hash())
	}

	childVoteSet := chain.FindVotesByHash(childBlock.Hash())

	vcpProof, err := proveVCP(parentBlock, db)
	if err != nil {
		return "", fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.TailTrio = core.SnapshotBlockTrio{
		First:  core.SnapshotFirstBlock{Header: parentBlock.BlockHeader, Proof: *vcpProof},
		Second: core.SnapshotSecondBlock{Header: lastFinalizedBlock.BlockHeader},
		Third:  core.SnapshotThirdBlock{Header: childBlock.BlockHeader, VoteSet: childVoteSet},
	}

	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return "", err
	}

	// -------------- Export the StoreView Section -------------- //

	// Genesis storeview
	genesisSV := state.NewStoreView(genesisBlockHeader.Height, genesisBlockHeader.StateHash, db)
	writeStoreView(genesisSV, false, writer, db)

	// Last checkpoint storeview
	if lastFinalizedBlock.Height != lastCheckpointHeight {
		lastCheckpointSV := state.NewStoreView(lastCheckpointBlock.Height, lastCheckpointBlock.StateHash, db)
		writeStoreView(lastCheckpointSV, true, writer, db)
	}

	// Parent block storeview
	parentSV := state.NewStoreView(parentBlock.Height, parentBlock.StateHash, db)
	writeStoreView(parentSV, true, writer, db)
	writeStoreView(sv, true, writer, db)

	return filename, nil
}

func ExportSnapshotV3(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain, snapshotDir string, height uint64) (string, error) {
	var lastFinalizedBlock *core.ExtendedBlock
	if height != 0 {
		blocks := chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsDirectlyFinalized() {
				lastFinalizedBlock = block
				break
			}
		}
		if lastFinalizedBlock == nil {
			return "", fmt.Errorf("Can't find finalized block at height %v", height)
		}
	} else {
		stub := consensus.GetSummary()
		var err error
		lastFinalizedBlock, err = chain.FindBlock(stub.LastFinalizedBlock)
		if err != nil {
			logger.Errorf("Failed to get block %v, %v", stub.LastFinalizedBlock, err)
			return "", err
		}
	}
	sv := state.NewStoreView(lastFinalizedBlock.Height, lastFinalizedBlock.BlockHeader.StateHash, db)

	currentTime := time.Now().UTC()
	filename := "theta_snapshot-" + strconv.FormatUint(sv.Height(), 10) + "-" + sv.Hash().String() + "-" + currentTime.Format("2006-01-02")
	snapshotPath := path.Join(snapshotDir, filename)
	file, err := os.Create(snapshotPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// --------------- Export the Header Section --------------- //

	snapshotHeader := &core.SnapshotHeader{
		Magic:   core.SnapshotHeaderMagic,
		Version: 3,
	}
	err = core.WriteSnapshotHeader(writer, snapshotHeader)
	if err != nil {
		return "", err
	}

	// ------------ Export the Last Checkpoint Section ------------- //

	lastFinalizedBlockHeight := lastFinalizedBlock.Height
	lastCheckpointHeight := common.LastCheckPointHeight(lastFinalizedBlockHeight)
	lastCheckpoint := &core.LastCheckpoint{}

	currHeight := lastFinalizedBlockHeight
	currBlock := lastFinalizedBlock
	for currHeight > lastCheckpointHeight {
		parentHash := currBlock.Parent
		currBlock, err = chain.FindBlock(parentHash)
		if err != nil {
			logger.Errorf("Failed to get intermediate block %v, %v", parentHash.Hex(), err)
			return "", err
		}
		lastCheckpoint.IntermediateHeaders = append(lastCheckpoint.IntermediateHeaders, currBlock.Block.BlockHeader)
		currHeight = currBlock.Height
	}

	lastCheckpointBlock := currBlock
	lastCheckpoint.CheckpointHeader = lastCheckpointBlock.BlockHeader

	err = core.WriteLastCheckpoint(writer, lastCheckpoint)
	if err != nil {
		return "", err
	}

	// -------------- Export the Metadata Section -------------- //

	metadata := &core.SnapshotMetadata{}
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
				genesisBlockHeader = blockTrio.Second.Header
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
					Second: core.SnapshotSecondBlock{Header: genesisBlock.BlockHeader},
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
						return "", err
					}
					if b != nil {
						child = *b.BlockHeader
						b, err = getFinalizedChild(b, chain)
						if err != nil {
							return "", err
						}
						if b != nil {
							grandChild = *b.BlockHeader
						} else {
							return "", fmt.Errorf("Can't find finalized grandchild block. " +
								"Likely the last finalized block also contains stake change transactions. " +
								"Please try again in 30 seconds.")
						}
					} else {
						return "", fmt.Errorf("Can't find finalized child block. " +
							"Likely the last finalized block also contains stake change transactions. " +
							"Please try again in 30 seconds.")
					}

					if child.HCC.BlockHash != block.Hash() || grandChild.HCC.BlockHash != child.Hash() {
						return "", fmt.Errorf("Invalid block HCC link for validator set changes")
					}
					if grandChild.HCC.Votes.IsEmpty() {
						return "", fmt.Errorf("Missing block HCC votes for validator set changes")
					}
					for _, vote := range grandChild.HCC.Votes.Votes() {
						if vote.Block != child.Hash() {
							return "", fmt.Errorf("Invalid block HCC votes for validator set changes")
						}
					}

					vcpProof, err := proveVCP(block, db)
					if err != nil {
						return "", fmt.Errorf("Failed to get VCP Proof")
					}
					metadata.ProofTrios = append(metadata.ProofTrios,
						core.SnapshotBlockTrio{
							First:  core.SnapshotFirstBlock{Header: block.BlockHeader, Proof: *vcpProof},
							Second: core.SnapshotSecondBlock{Header: &child},
							Third:  core.SnapshotThirdBlock{Header: &grandChild},
						})
					foundDirectlyFinalizedBlock = true
					break
				}
			}
			if !foundDirectlyFinalizedBlock {
				return "", fmt.Errorf("Finalized block not found for height %v", height)
			}
		}
	}

	parentBlock, err := chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's parent, %v", err)
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, chain)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's committed child, %v", err)
	}

	if lastFinalizedBlock.HCC.BlockHash != parentBlock.Hash() {
		return "", fmt.Errorf("Parent block hash mismatch: %v vs %v", lastFinalizedBlock.HCC.BlockHash, parentBlock.Hash())
	}

	if childBlock.HCC.BlockHash != lastFinalizedBlock.Hash() {
		return "", fmt.Errorf("Finalized block hash mismatch: %v vs %v", childBlock.HCC.BlockHash, lastFinalizedBlock.Hash())
	}

	childVoteSet := chain.FindVotesByHash(childBlock.Hash())

	vcpProof, err := proveVCP(parentBlock, db)
	if err != nil {
		return "", fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.TailTrio = core.SnapshotBlockTrio{
		First:  core.SnapshotFirstBlock{Header: parentBlock.BlockHeader, Proof: *vcpProof},
		Second: core.SnapshotSecondBlock{Header: lastFinalizedBlock.BlockHeader},
		Third:  core.SnapshotThirdBlock{Header: childBlock.BlockHeader, VoteSet: childVoteSet},
	}

	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return "", err
	}

	// -------------- Export the StoreView Section -------------- //

	// Genesis storeview
	genesisSV := state.NewStoreView(genesisBlockHeader.Height, genesisBlockHeader.StateHash, db)
	writeStoreViewV3(genesisSV, false, writer, db, common.Hash{})

	// Last checkpoint storeview
	if lastFinalizedBlock.Height != lastCheckpointHeight {
		lastCheckpointSV := state.NewStoreView(lastCheckpointBlock.Height, lastCheckpointBlock.StateHash, db)
		writeStoreViewV3(lastCheckpointSV, false, writer, db, genesisSV.Hash())
	}

	// Parent block storeview
	parentSV := state.NewStoreView(parentBlock.Height, parentBlock.StateHash, db)
	writeStoreViewV3(parentSV, false, writer, db, genesisSV.Hash())
	writeStoreViewV3(sv, true, writer, db, parentSV.Hash())

	return filename, nil
}

func ExportSnapshotV4(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain, snapshotDir string, height uint64) (string, error) {
	var lastFinalizedBlock *core.ExtendedBlock
	if height != 0 {
		blocks := chain.FindBlocksByHeight(height)
		for _, block := range blocks {
			if block.Status.IsDirectlyFinalized() {
				lastFinalizedBlock = block
				break
			}
		}
		if lastFinalizedBlock == nil {
			return "", fmt.Errorf("Can't find finalized block at height %v", height)
		}
	} else {
		stub := consensus.GetSummary()
		var err error
		lastFinalizedBlock, err = chain.FindBlock(stub.LastFinalizedBlock)
		if err != nil {
			logger.Errorf("Failed to get block %v, %v", stub.LastFinalizedBlock, err)
			return "", err
		}
	}
	sv := state.NewStoreView(lastFinalizedBlock.Height, lastFinalizedBlock.BlockHeader.StateHash, db)

	currentTime := time.Now().UTC()
	filename := "theta_snapshot-" + strconv.FormatUint(sv.Height(), 10) + "-" + sv.Hash().String() + "-" + currentTime.Format("2006-01-02")
	snapshotPath := path.Join(snapshotDir, filename)
	file, err := os.Create(snapshotPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)

	// --------------- Export the Header Section --------------- //

	snapshotHeader := &core.SnapshotHeader{
		Magic:   core.SnapshotHeaderMagic,
		Version: 4,
	}
	err = core.WriteSnapshotHeader(writer, snapshotHeader)
	if err != nil {
		return "", err
	}

	// ------------ Export the Last Checkpoint Section ------------- //

	lastFinalizedBlockHeight := lastFinalizedBlock.Height
	lastCheckpointHeight := common.LastCheckPointHeight(lastFinalizedBlockHeight)
	lastCheckpoint := &core.LastCheckpoint{}

	currHeight := lastFinalizedBlockHeight
	currBlock := lastFinalizedBlock
	for currHeight > lastCheckpointHeight {
		parentHash := currBlock.Parent
		currBlock, err = chain.FindBlock(parentHash)
		if err != nil {
			logger.Errorf("Failed to get intermediate block %v, %v", parentHash.Hex(), err)
			return "", err
		}
		lastCheckpoint.IntermediateHeaders = append(lastCheckpoint.IntermediateHeaders, currBlock.Block.BlockHeader)
		currHeight = currBlock.Height
	}

	lastCheckpointBlock := currBlock

	lastCheckpoint.CheckpointHeader = lastCheckpointBlock.BlockHeader

	err = core.WriteLastCheckpoint(writer, lastCheckpoint)
	if err != nil {
		return "", err
	}

	// -------------- Export the Metadata Section -------------- //

	metadata := &core.SnapshotMetadata{}

	parentBlock, err := chain.FindBlock(lastFinalizedBlock.Parent)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's parent, %v", err)
	}
	childBlock, err := getAtLeastCommittedChild(lastFinalizedBlock, chain)
	if err != nil {
		return "", fmt.Errorf("Failed to find last finalized block's committed child, %v", err)
	}

	if lastFinalizedBlock.HCC.BlockHash != parentBlock.Hash() {
		return "", fmt.Errorf("Parent block hash mismatch: %v vs %v", lastFinalizedBlock.HCC.BlockHash, parentBlock.Hash())
	}

	if childBlock.HCC.BlockHash != lastFinalizedBlock.Hash() {
		return "", fmt.Errorf("Finalized block hash mismatch: %v vs %v", childBlock.HCC.BlockHash, lastFinalizedBlock.Hash())
	}

	childVoteSet := chain.FindVotesByHash(childBlock.Hash())

	vcpProof, err := proveVCP(parentBlock, db)
	if err != nil {
		return "", fmt.Errorf("Failed to get VCP Proof")
	}
	metadata.TailTrio = core.SnapshotBlockTrio{
		First:  core.SnapshotFirstBlock{Header: parentBlock.BlockHeader, Proof: *vcpProof},
		Second: core.SnapshotSecondBlock{Header: lastFinalizedBlock.BlockHeader},
		Third:  core.SnapshotThirdBlock{Header: childBlock.BlockHeader, VoteSet: childVoteSet},
	}

	err = core.WriteMetadata(writer, metadata)
	if err != nil {
		return "", err
	}

	// -------------- Export the StoreView Section -------------- //
	// Last checkpoint storeview
	if lastFinalizedBlock.Height != lastCheckpointHeight {
		lastCheckpointSV := state.NewStoreView(lastCheckpointBlock.Height, lastCheckpointBlock.StateHash, db)
		writeStoreViewV3(lastCheckpointSV, false, writer, db, common.Hash{})
	}

	// Parent block storeview
	parentSV := state.NewStoreView(parentBlock.Height, parentBlock.StateHash, db)
	writeStoreViewV3(parentSV, false, writer, db, common.Hash{})

	writeStoreViewV3(sv, true, writer, db, parentSV.Hash())

	return filename, nil
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
		if needAccountStorage && bytes.HasPrefix(k, []byte("ls/a")) {
			account := &types.Account{}
			err := types.FromBytes([]byte(v), account)
			if err != nil {
				logger.Errorf("Failed to parse account for %v", []byte(v))
				panic(err)
			}
			if account.Root != (common.Hash{}) {
				err = core.WriteRecord(writer, []byte{core.SVStart}, height)
				if err != nil {
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
		}
		return true
	})
	err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}

func writeStoreViewV3(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer, db database.Database, base common.Hash) {
	writeTrie(sv.Hash(), writer, db, base)

	if needAccountStorage {
		sv.GetStore().Traverse(nil, func(k, v common.Bytes) bool {
			if needAccountStorage && bytes.HasPrefix(k, []byte("ls/a")) {
				account := &types.Account{}
				err := types.FromBytes([]byte(v), account)
				if err != nil {
					logger.Errorf("Failed to parse account for %v", []byte(v))
					panic(err)
				}
				if account.Root != (common.Hash{}) {
					writeTrie(account.Root, writer, db, common.Hash{})
				}
			}
			return true
		})
	}
}

func writeTrie(root common.Hash, writer *bufio.Writer, db database.Database, base common.Hash) {
	tr, err := trie.New(root, trie.NewDatabase(db))
	if err != nil {
		log.Panic(err)
	}
	var it trie.NodeIterator
	if !base.IsEmpty() {
		baseTr, err := trie.New(base, trie.NewDatabase(db))
		if err != nil {
			log.Panic(err)
		}
		it, _ = trie.NewDifferenceIterator(baseTr.NodeIterator(nil), tr.NodeIterator(nil))
	} else {
		it = tr.NodeIterator(nil)
	}
	for it.Next(true) {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			val, err := db.Get(hash.Bytes())
			if err != nil {
				log.Panic(err)
			}
			err = core.WriteRecord(writer, hash.Bytes(), val)
			if err != nil {
				log.Panic(err)
			}
		}
	}
	writer.Flush()
}
