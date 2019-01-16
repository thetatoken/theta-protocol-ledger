package netsync

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/treestore"
)

var (
	genesisValidatorAddrs = []string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
	}
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
	svHashes := make(map[common.Hash]bool)
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
					h := sv.Save()
					svHashes[h] = true
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

	for _, blockPair := range metadata.BlocksWithValidatorChange {
		_, ok := svHashes[blockPair.First.StateHash]
		if ok {
			delete(svHashes, blockPair.First.StateHash)
		} else {
			return nil, fmt.Errorf("Storeview missing for block %v", blockPair.First.StateHash)
		}
	}
	if len(svHashes) != 0 {
		return nil, fmt.Errorf("Can't find matching state hash for storeview")
	}

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

	var validatorSet *core.ValidatorSet
	for i, blockPair := range metadata.BlocksWithValidatorChange {
		if !blockPair.First.Status.IsDirectlyFinalized() || !blockPair.Second.Status.IsFinalized() || blockPair.Second.Parent != blockPair.First.Hash() {
			return false
		}

		var block *core.BlockHeader
		if i > 0 {
			block = metadata.BlocksWithValidatorChange[i-1].First.BlockHeader
		}
		validatorSet = getValidatorSet(block, db)
		validateBlock(blockPair.First.BlockHeader, validatorSet, voteSetMap)
		validateBlock(blockPair.Second.BlockHeader, validatorSet, voteSetMap)
	}
	validateBlock(&metadata.Blockheader, nil, voteSetMap)

	return true
}

func getValidatorSet(block *core.BlockHeader, db database.Database) *core.ValidatorSet {
	if block == nil {
		validators := []core.Validator{}
		for _, addr := range genesisValidatorAddrs {
			raw, _ := hex.DecodeString(addr)
			pubKey, _ := crypto.PublicKeyFromBytes(raw)
			address := pubKey.Address()
			stake := new(big.Int).Mul(new(big.Int).SetUint64(5), core.MinValidatorStakeDeposit) //TODO: decide stake
			validators = append(validators, core.Validator{Address: address, Stake: stake})
		}
		validatorSet := core.NewValidatorSet()
		validatorSet.SetValidators(validators)
		return validatorSet
	}

	sv := state.NewStoreView(block.Height, block.StateHash, db)
	vcp := sv.GetValidatorCandidatePool()
	return consensus.GetValidatorSetFromVCP(vcp)
}

func validateBlock(block *core.BlockHeader, validatorSet *core.ValidatorSet, voteSetMap map[common.Hash]*core.VoteSet) bool {
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
