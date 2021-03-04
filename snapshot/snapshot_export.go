package snapshot

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
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

func ExportSnapshot(db database.Database, consensus *cns.ConsensusEngine, chain *blockchain.Chain, snapshotDir string, height uint64) (string, error) {
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

type kvPair struct {
	k []byte
	v []byte
}

func writeStoreView(sv *state.StoreView, needAccountStorage bool, writer *bufio.Writer, db database.Database) {
	height := core.Itobytes(sv.Height())
	err := core.WriteRecord(writer, []byte{core.SVStart}, height)
	if err != nil {
		panic(err)
	}

	kvs := make(chan *kvPair, 4096)
	prefixes := make(chan []byte)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	wg.Add(256) // # of prefixes
	go func() {
		defer wg.Done()

		for prefix := 0; prefix <= 255; prefix++ {
			prefixes <- []byte{byte(prefix)}
		}
	}()

	// Spawn workers
	for i := 0; i < viper.GetInt(common.CfgSnapshotExportWorker); i++ {
		go func() {
			for {
				prefix, ok := <-prefixes
				if !ok {
					return
				}
				sv.GetStore().Traverse(prefix, func(k, v common.Bytes) bool {
					kvs <- &kvPair{k: k, v: v}
					return true
				})
				wg.Done()
			}
		}()
	}

	wg2 := &sync.WaitGroup{}
	wg2.Add(1)
	go func() {
		defer wg2.Done()

		for {
			kv, ok := <-kvs
			if !ok {
				return
			}

			k := kv.k
			v := kv.v

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
		}
	}()

	wg.Wait()
	close(prefixes)
	close(kvs)
	wg2.Wait()

	err = core.WriteRecord(writer, []byte{core.SVEnd}, height)
	if err != nil {
		panic(err)
	}
	writer.Flush()
}
