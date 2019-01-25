package netsync

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"os"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "snapshot"})

var (
	genesisValidatorAddrs = []string{
		"2E833968E5bB786Ae419c4d13189fB081Cc43bab",
	}
)

type SVStack []*state.StoreView

func (s SVStack) push(sv *state.StoreView) SVStack {
	return append(s, sv)
}

func (s SVStack) pop() (SVStack, *state.StoreView) {
	l := len(s)
	if l == 0 {
		return s, nil
	}
	return s[:l-1], s[l-1]
}

func (s SVStack) peek() *state.StoreView {
	l := len(s)
	if l == 0 {
		return nil
	}
	return s[l-1]
}

func LoadSnapshot(filePath string, db database.Database) (*core.BlockHeader, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metadata := core.SnapshotMetadata{}
	err = core.ReadRecord(file, &metadata)
	if err != nil {
		return nil, fmt.Errorf("Failed to load snapshot metadata, %v", err)
	}

	var hash common.Hash
	var sv *state.StoreView
	var account *types.Account
	svStack := make(SVStack, 0)
	for {
		record := core.SnapshotTrieRecord{}
		err := core.ReadRecord(file, &record)
		if err != nil {
			if err == io.EOF {
				if svStack.peek() != nil {
					return nil, fmt.Errorf("Still some storeview unhandled")
				}
				break
			}
			return nil, fmt.Errorf("Failed to read snapshot record, %v", err)
		}

		if bytes.Equal(record.K, []byte{core.SVStart}) {
			height := core.Bytestoi(record.V)
			sv := state.NewStoreView(height, common.Hash{}, db)
			svStack = svStack.push(sv)
		} else if bytes.Equal(record.K, []byte{core.SVEnd}) {
			svStack, sv = svStack.pop()
			if sv == nil {
				return nil, fmt.Errorf("Missing storeview to handle")
			}
			height := core.Bytestoi(record.V)
			if height != sv.Height() {
				return nil, fmt.Errorf("Storeview start and end heights don't match")
			}
			hash = sv.Save()

			if svStack.peek() != nil && height == svStack.peek().Height() {
				// it's a storeview for account storage, verify account
				if bytes.Compare(account.Root.Bytes(), hash.Bytes()) != 0 {
					return nil, fmt.Errorf("Account storage root doesn't match")
				}
			}
			account = nil
		} else {
			sv := svStack.peek()
			if sv == nil {
				return nil, fmt.Errorf("Missing storeview to handle")
			}
			sv.Set(record.K, record.V)

			if account == nil {
				if strings.HasPrefix(record.K.String(), "ls/a/") {
					account = &types.Account{}
					err = types.FromBytes([]byte(record.V), account)
					if err != nil {
						return nil, fmt.Errorf("Failed to parse account, %v", err)
					}
				}
			}
		}
	}

	if err = validateSnapshot(&metadata, hash, db); err != nil {
		return nil, fmt.Errorf("Snapshot validation failed: %v", err)
	}

	kvstore := kvstore.NewKVStore(db)
	blockTrio := metadata.BlockTrios[len(metadata.BlockTrios)-1]

	block := core.Block{BlockHeader: &blockTrio.First.Header}
	ext := core.ExtendedBlock{Block: &block}
	blockHash := blockTrio.First.Header.Hash()
	kvstore.Put(blockHash[:], ext)

	block = core.Block{BlockHeader: &blockTrio.Second.Header}
	ext = core.ExtendedBlock{Block: &block}
	blockHash = blockTrio.First.Header.Hash()
	kvstore.Put(blockHash[:], ext)

	for i, blockTrio := range metadata.BlockTrios {
		if i < len(metadata.BlockTrios)-1 {
			kvstore.Put([]byte(core.BlockTrioStoreKeyPrefix+strconv.FormatUint(blockTrio.First.Header.Height, 64)), blockTrio)
		}
	}

	return &blockTrio.Second.Header, nil
}

func validateSnapshot(metadata *core.SnapshotMetadata, hash common.Hash, db database.Database) error {
	if bytes.Compare(metadata.BlockTrios[len(metadata.BlockTrios)-1].Second.Header.StateHash.Bytes(), hash.Bytes()) != 0 {
		return fmt.Errorf("StateHash not matching")
	}

	var validatorSet *core.ValidatorSet
	for i, blockTrio := range metadata.BlockTrios {
		if blockTrio.Second.Header.Parent != blockTrio.First.Header.Hash() || blockTrio.Third.Header.Parent != blockTrio.Second.Header.Hash() {
			return fmt.Errorf("block trio has invalid Parent link")
		}
		if blockTrio.Second.Header.HCC.BlockHash != blockTrio.First.Header.Hash() || blockTrio.Third.Header.HCC.BlockHash != blockTrio.Second.Header.Hash() {
			return fmt.Errorf("block trio has invalid HCC link: %v, %v; %v, %v", blockTrio.First.Header.Hash(), blockTrio.Second.Header.HCC.BlockHash, blockTrio.Second.Header.Hash(), blockTrio.Third.Header.HCC.BlockHash)
		}

		if i > 0 {
			prevBlockTrio := metadata.BlockTrios[i-1]
			validatorSet, _ = getValidatorSet(&prevBlockTrio.First.Header, db, &blockTrio.First.Proof)
		} else {
			validators := []core.Validator{}
			for _, addr := range genesisValidatorAddrs {
				address := common.HexToAddress(addr)
				stake := new(big.Int).Mul(new(big.Int).SetUint64(1), core.MinValidatorStakeDeposit)
				validators = append(validators, core.Validator{Address: address, Stake: stake})
			}
			validatorSet = core.NewValidatorSet()
			validatorSet.SetValidators(validators)
		}

		if err := validateVotes(&blockTrio.Second.Header, validatorSet, blockTrio.Second.Votes); err != nil {
			return fmt.Errorf("Failed to validate voteSet, %v", err)
		}
	}

	lastBlockTrio := metadata.BlockTrios[len(metadata.BlockTrios)-1]
	validateVotes(&lastBlockTrio.Third.Header, validatorSet, lastBlockTrio.Third.Votes)

	return nil
}

func getValidatorSet(block *core.BlockHeader, db database.Database, recoverredVp *core.VCPProof) (*core.ValidatorSet, error) {
	sv := state.NewStoreView(block.Height, block.StateHash, db)
	serializedVCP, err := sv.VerifyProof(sv.Hash(), state.ValidatorCandidatePoolKey(), recoverredVp)
	if err != nil {
		return nil, err
	}

	vcp := &core.ValidatorCandidatePool{}
	err = rlp.DecodeBytes(serializedVCP, vcp)
	if err != nil {
		return nil, err
	}
	return consensus.SelectTopStakeHoldersAsValidators(vcp), nil
}

func getValidatorSetFromSV(block *core.BlockHeader, db database.Database) *core.ValidatorSet {
	if block == nil {
		validators := []core.Validator{}
		for _, addr := range genesisValidatorAddrs {
			address := common.HexToAddress(addr)
			stake := new(big.Int).Mul(new(big.Int).SetUint64(1), core.MinValidatorStakeDeposit)
			validators = append(validators, core.Validator{Address: address, Stake: stake})
		}
		validatorSet := core.NewValidatorSet()
		validatorSet.SetValidators(validators)
		return validatorSet
	}

	sv := state.NewStoreView(block.Height, block.StateHash, db)
	vcp := sv.GetValidatorCandidatePool()
	return consensus.SelectTopStakeHoldersAsValidators(vcp)
}

func validateVotes(block *core.BlockHeader, validatorSet *core.ValidatorSet, votes []core.Vote) error {
	if !validatorSet.HasMajorityVotes(votes) {
		return fmt.Errorf("block doesn't have majority votes")
	}
	for _, vote := range votes {
		// res := vote.Validate()
		// if !res.IsOK() {
		// 	return fmt.Errorf("vote is not valid, %v", res)
		// }
		if vote.Block != block.Hash() {
			return fmt.Errorf("vote is not for corresponding block")
		}
		_, err := validatorSet.GetValidator(vote.ID)
		if err != nil {
			return fmt.Errorf("can't find validator for vote")
		}
	}
	return nil
}
