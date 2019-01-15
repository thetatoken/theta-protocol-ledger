package netsync

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/treestore"
)

func LoadSnapshot(filePath string, db database.Database) (*core.SnapshotMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)

	metadata, err := readMetadata(reader)
	if err != nil {
		log.Errorf("Failed to load snapshot block")
		return nil, err
	}

	var hash common.Hash
	var sv *state.StoreView
	var account *types.Account
	var accountStorage *treestore.TreeStore
	svSeq := -1
	// svHashes := make(map[common.Hash]bool)
	for {
		record, err := readRecord(reader)
		if err != nil {
			if err == io.EOF {
				if accountStorage != nil {
					accountStorage.Commit()
				}
				if sv != nil {
					hash = sv.Save()
				}
				break
			}
			log.Errorf("Failed to read snapshot record")
			return nil, err
		}

		if len(record.R) == 0 {
			if record.S != svSeq {
				if sv != nil {
					sv.Save()
					// h := sv.Save()
					// svHashes[h] = true
				}
				sv = state.NewStoreView(metadata.Blockheader.Height, common.Hash{}, db)
				svSeq = record.S
			}
			sv.Set(record.K, record.V)
		} else {
			if account == nil || !bytes.Equal(account.Root.Bytes(), record.R) {
				root, err := accountStorage.Commit()
				if err != nil {
					log.Errorf("Failed to commit account storage %v", account.Root)
					return nil, err
				}
				if account != nil && bytes.Compare(account.Root.Bytes(), root.Bytes()) != 0 {
					return nil, fmt.Errorf("Account storage root doesn't match %v != %v", account.Root.Bytes(), root.Bytes())
				}

				// reset temporary account and account storage
				account = &types.Account{}
				err = types.FromBytes([]byte(record.V), account)
				if err != nil {
					log.Errorf("Failed to parse account for %v", []byte(record.V))
					return nil, err
				}
				accountStorage = treestore.NewTreeStore(common.Hash{}, db)
			}
			accountStorage.Set(record.K, record.V)
		}
	}

	// for _, block := range metadata.BlocksWithValidatorChange {
	// 	_, ok := svHashes[block.StateHash]
	// 	if ok {
	// 		delete(svHashes, block.StateHash)
	// 		// } else {
	// 		// 	return nil, fmt.Errorf("Storeview missing for block %v", block.StateHash)
	// 	}
	// }
	// if len(svHashes) != 0 {
	// 	return nil, fmt.Errorf("Can't find matching state hash for storeview")
	// }

	if !validateSnapshot(metadata, hash, db) {
		return nil, fmt.Errorf("Snapshot validation failed")
	}
	return metadata, nil
}

func validateSnapshot(metadata *core.SnapshotMetadata, hash common.Hash, db database.Database) bool {
	if bytes.Compare(metadata.Blockheader.StateHash.Bytes(), hash.Bytes()) != 0 {
		return false
	}

	voteSetMap := map[common.Hash]*core.VoteSet{}
	for _, vote := range metadata.Votes {
		if _, ok := voteSetMap[vote.Block]; !ok {
			voteSetMap[vote.Block] = core.NewVoteSet()
		}
		voteSetMap[vote.Block].AddVote(vote)
	}

	for _, blockPair := range metadata.BlocksWithValidatorChange {
		if !blockPair.First.Status.IsDirectlyFinalized() || !blockPair.Second.Status.IsDirectlyFinalized() || blockPair.Second.Parent != blockPair.First.Hash() {
			return false
		}
		validateBlock(blockPair.First.BlockHeader, voteSetMap, db)
		validateBlock(blockPair.Second.BlockHeader, voteSetMap, db)
	}
	validateBlock(&metadata.Blockheader, voteSetMap, db)

	return true
}

func validateBlock(block *core.BlockHeader, voteSetMap map[common.Hash]*core.VoteSet, db database.Database) bool {
	sv := state.NewStoreView(block.Height, block.StateHash, db)
	vcp := sv.GetValidatorCandidatePool()
	maxNumValidators := viper.GetInt(common.CfgConsensusMaxNumValidators)
	topStakeHolders := vcp.GetTopStakeHolders(maxNumValidators)

	validatorSet := core.NewValidatorSet()
	for _, stakeHolder := range topStakeHolders {
		valAddr := stakeHolder.Holder.Hex()
		valStake := stakeHolder.TotalStake()
		validator := core.NewValidator(valAddr, valStake)
		validatorSet.AddValidator(validator)
	}
	if !validatorSet.HasMajority(voteSetMap[block.Hash()]) {
		return false
	}
	for _, vote := range voteSetMap[block.Hash()].Votes() {
		if !vote.Validate().IsOK() {
			return false
		}
		if bytes.Compare(vote.Block.Bytes(), block.Hash().Bytes()) != 0 {
			return false
		}
		_, err := validatorSet.GetValidator(vote.ID)
		if err != nil {
			return false
		}
	}
	return true
}

func readMetadata(reader *bufio.Reader) (*core.SnapshotMetadata, error) {
	metadata := &core.SnapshotMetadata{}
	sizeBytes := make([]byte, 8)
	_, err := reader.Read(sizeBytes)
	if err != nil {
		return metadata, err
	}
	size := bstoi(sizeBytes)
	metadataBytes := make([]byte, size)
	_, err = reader.Read(metadataBytes)
	if err != nil {
		return metadata, err
	}
	err = rlp.DecodeBytes(metadataBytes, metadata)
	return metadata, err
}

func readRecord(reader *bufio.Reader) (*core.SnapshotRecord, error) {
	record := &core.SnapshotRecord{}
	sizeBytes := make([]byte, 8)
	_, err := reader.Read(sizeBytes)
	if err != nil {
		return record, err
	}
	size := bstoi(sizeBytes)
	recordBytes := make([]byte, size)
	_, err = reader.Read(recordBytes)
	if err != nil {
		return record, err
	}
	err = rlp.DecodeBytes(recordBytes, record)
	return record, err
}

func bstoi(arr []byte) (val uint64) {
	for i := 0; i < 8; i++ {
		val = val + uint64(arr[i])*uint64(math.Pow10(i))
	}
	return
}
