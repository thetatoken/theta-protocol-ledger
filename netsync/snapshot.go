package netsync

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/kvstore"
)

var (
	genesisValidatorAddrs = []string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
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

	metadata, err := readMetadata(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to load snapshot metadata, %v", err)
	}

	var hash common.Hash
	var sv *state.StoreView
	var account *types.Account
	svHashes := make(map[common.Hash]bool)
	svStack := make(SVStack, 0)
	for {
		record, err := readRecord(file)
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
			height := bstoi(record.V)
			sv := state.NewStoreView(height, common.Hash{}, db)
			svStack = svStack.push(sv)
		} else if bytes.Equal(record.K, []byte{core.SVEnd}) {
			svStack, sv = svStack.pop()
			if sv == nil {
				return nil, fmt.Errorf("Missing storeview to handle")
			}
			height := bstoi(record.V)
			if height != sv.Height() {
				return nil, fmt.Errorf("Storeview start and end heights don't match")
			}
			hash = sv.Save()

			if svStack.peek() != nil && height == svStack.peek().Height() {
				// it's a storeview for account storage, verify account
				if bytes.Compare(account.Root.Bytes(), hash.Bytes()) != 0 {
					return nil, fmt.Errorf("Account storage root doesn't match")
				}
			} else {
				svHashes[hash] = true
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

	var blockTrio core.SnapshotBlockTrio
	for _, blockTrio = range metadata.BlockTrios {
		_, ok := svHashes[blockTrio.First.StateHash]
		if ok {
			delete(svHashes, blockTrio.First.StateHash)
		} else {
			return nil, fmt.Errorf("Storeview missing for block %v", blockTrio.First.StateHash)
		}
	}
	if len(svHashes) != 1 {
		return nil, fmt.Errorf("Can't find matching state hash for storeview")
	}

	if !validateSnapshot(metadata, hash, db) {
		return nil, fmt.Errorf("Snapshot validation failed")
	}

	kvstore := kvstore.NewKVStore(db)
	block := core.Block{BlockHeader: &blockTrio.First}
	ext := core.ExtendedBlock{Block: &block}
	blockHash := blockTrio.First.Hash()
	kvstore.Put(blockHash[:], ext)
	block = core.Block{BlockHeader: &blockTrio.Second}
	ext = core.ExtendedBlock{Block: &block}
	blockHash = blockTrio.First.Hash()
	kvstore.Put(blockHash[:], ext)

	return &blockTrio.Second, nil
}

func validateSnapshot(metadata *core.SnapshotMetadata, hash common.Hash, db database.Database) bool {
	if bytes.Compare(metadata.BlockTrios[len(metadata.BlockTrios)-1].Second.StateHash.Bytes(), hash.Bytes()) != 0 {
		return false
	}

	var validatorSet *core.ValidatorSet
	for i, blockTrio := range metadata.BlockTrios {
		if blockTrio.Second.Parent != blockTrio.First.Hash() || blockTrio.Third.Header.Parent != blockTrio.Second.Hash() {
			return false
		}
		if blockTrio.Second.HCC.BlockHash != blockTrio.First.Hash() || blockTrio.Third.Header.HCC.BlockHash != blockTrio.Second.Hash() {
			return false
		}

		var block *core.BlockHeader
		if i > 0 {
			block = &metadata.BlockTrios[i-1].First
		}
		validatorSet = getValidatorSet(block, db)
		validateVotes(&blockTrio.Second, validatorSet, blockTrio.Third.Votes)
	}

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

func validateVotes(block *core.BlockHeader, validatorSet *core.ValidatorSet, votes []core.Vote) bool {
	if !validatorSet.HasMajorityVotes(votes) {
		return false
	}
	for _, vote := range votes {
		if !vote.Validate().IsOK() {
			return false
		}
		// if vote.Block != block.Hash() {
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

func readMetadata(file *os.File) (*core.SnapshotMetadata, error) {
	metadata := &core.SnapshotMetadata{}
	sizeBytes := make([]byte, 8)
	n, err := io.ReadAtLeast(file, sizeBytes, 8)
	if err != nil {
		return metadata, err
	}
	if n < 8 {
		return nil, fmt.Errorf("Failed to read metadata length")
	}
	size := bstoi(sizeBytes)
	metadataBytes := make([]byte, size)
	n, err = io.ReadAtLeast(file, metadataBytes, int(size))
	if err != nil {
		return metadata, err
	}
	if uint64(n) < size {
		return nil, fmt.Errorf("Failed to read metadata, %v < %v", n, size)
	}
	err = rlp.DecodeBytes(metadataBytes, metadata)
	return metadata, err
}

func readRecord(file *os.File) (*core.SnapshotTrieRecord, error) {
	record := &core.SnapshotTrieRecord{}
	sizeBytes := make([]byte, 8)
	n, err := io.ReadAtLeast(file, sizeBytes, 8)
	if err != nil {
		return nil, err
	}
	if n < 8 {
		return nil, fmt.Errorf("Failed to read record length")
	}
	size := bstoi(sizeBytes)
	recordBytes := make([]byte, size)
	n, err = io.ReadAtLeast(file, recordBytes, int(size))
	if err != nil {
		return nil, err
	}
	if uint64(n) < size {
		return nil, fmt.Errorf("Failed to read record, %v < %v", n, size)
	}
	err = rlp.DecodeBytes(recordBytes, record)
	return record, err
}

func bstoi(arr []byte) (val uint64) {
	// for i := 0; i < 8; i++ {
	// 	val = val + uint64(arr[i])*uint64(math.Pow10(i))
	// }
	val = binary.LittleEndian.Uint64(arr)
	return
}
