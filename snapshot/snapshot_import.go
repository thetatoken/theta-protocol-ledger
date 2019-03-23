package snapshot

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"
	"github.com/thetatoken/theta/store/trie"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "snapshot"})

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

// ImportSnapshot loads the snapshot into the given database
func ImportSnapshot(snapshotFilePath, chainImportDirPath string, db database.Database) (*core.BlockHeader, error) {
	logger.Printf("Loading snapshot from: %v", snapshotFilePath)
	blockHeader, err := loadSnapshot(snapshotFilePath, chainImportDirPath, db)
	if err != nil {
		return nil, err
	}
	logger.Printf("Snapshot loaded successfully.")
	return blockHeader, nil
}

// ValidateSnapshot validates the snapshot using a temporary database
func ValidateSnapshot(snapshotFilePath, chainImportDirPath string) (*core.BlockHeader, error) {
	logger.Printf("Verifying snapshot: %v", snapshotFilePath)

	tmpdbRoot, err := ioutil.TempDir("", "tmpdb")
	if err != nil {
		log.Panicf("Failed to create temporary db for snapshot verification: %v", err)
	}
	mainTmpDBPath := path.Join(tmpdbRoot, "main")
	refTmpDBPath := path.Join(tmpdbRoot, "ref")
	defer func() {
		os.RemoveAll(mainTmpDBPath)
		os.RemoveAll(refTmpDBPath)
	}()

	tmpdb, err := backend.NewLDBDatabase(mainTmpDBPath, refTmpDBPath, 256, 0)

	blockHeader, err := loadSnapshot(snapshotFilePath, chainImportDirPath, tmpdb)
	if err != nil {
		return nil, err
	}
	logger.Printf("Snapshot verified.")

	err = loadChain(chainImportDirPath, tmpdb)
	if err != nil {
		return nil, err
	}

	return blockHeader, nil
}

func loadSnapshot(snapshotFilePath, chainImportDirPath string, db database.Database) (*core.BlockHeader, error) {
	snapshotFile, err := os.Open(snapshotFilePath)
	if err != nil {
		return nil, err
	}
	defer snapshotFile.Close()

	// ------------------------------ Load State ------------------------------ //

	metadata := core.SnapshotMetadata{}
	err = core.ReadRecord(snapshotFile, &metadata)
	if err != nil {
		return nil, fmt.Errorf("Failed to load snapshot metadata, %v", err)
	}
	sv, _, err := loadState(snapshotFile, db)
	if err != nil {
		return nil, err
	}

	// ----------------------------- Validity Checks -------------------------- //

	if err = checkSnapshot(sv, &metadata, db); err != nil {
		return nil, fmt.Errorf("Snapshot state validation failed: %v", err)
	}

	// --------------------- Save Proofs and Tail Blocks  --------------------- //

	kvstore := kvstore.NewKVStore(db)

	for _, blockTrio := range metadata.ProofTrios {
		blockTrioKey := []byte(core.BlockTrioStoreKeyPrefix + strconv.FormatUint(blockTrio.First.Header.Height, 10))
		kvstore.Put(blockTrioKey, blockTrio)
	}

	secondBlockHeader := saveTailBlocks(&metadata, sv, kvstore)
	return secondBlockHeader, nil
}

func loadChain(chainImportDirPath string, db database.Database) error {
	if _, err := os.Stat(chainImportDirPath); !os.IsNotExist(err) {
		fileInfos, err := ioutil.ReadDir(chainImportDirPath)
		if err != nil {
			return fmt.Errorf("Failed to read chain import directory %v: %v", chainImportDirPath, err)
		}

		var chainFileNames []string
		for _, fileInfo := range fileInfos {
			chainFileNames = append(chainFileNames, fileInfo.Name())
		}

		sort.Slice(chainFileNames, func(i, j int) bool {
			start1, _ := getChainBoundary(chainFileNames[i])
			start2, _ := getChainBoundary(chainFileNames[j])
			return start1 < start2
		})

		var currHeight uint64
		var lastBlock *core.ExtendedBlock
		for _, fileName := range chainFileNames {
			start, end := getChainBoundary(fileName)
			if start != currHeight {
				return fmt.Errorf("Missing chain file starting at height %v", currHeight)
			}
			chainFilePath := path.Join(chainImportDirPath, fileName)
			lastBlock, err = loadChainSegment(chainFilePath, start, end, lastBlock, db)
			currHeight = end + 1
		}
	}
	return nil
}

func loadChainSegment(filePath string, start, end uint64, lastBlock *core.ExtendedBlock, db database.Database) (*core.ExtendedBlock, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	kvstore := kvstore.NewKVStore(db)

	var count uint64
	for {
		backupBlock := &core.BackupBlock{}
		err := core.ReadRecord(file, backupBlock)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("Failed to read backup record, %v", err)
		}

		block := backupBlock.Block

		if res := block.Validate(); res.IsError() {
			return nil, fmt.Errorf("Block's header is invalid, %v", block.Height)
		}
		if block.TxHash != core.CalculateRootHash(block.Txs) {
			return nil, fmt.Errorf("Block's TxHash doesn't match, %v", block.Height)
		}

		if count == 0 && block.Height != start {
			return nil, fmt.Errorf("First block height doesn't match, %v : %v", block.Height, start)
		}
		count++
		if lastBlock == nil {
			_, err := checkGenesisBlock(block.BlockHeader, db)
			if err != nil {
				return nil, err
			}
		} else {
			if block.Height != lastBlock.Height+1 {
				return nil, fmt.Errorf("Block height missing at %v", lastBlock.Height+1)
			}
			if block.Parent != lastBlock.Hash() {
				return nil, fmt.Errorf("Block has invalid parent at %v", block.Height)
			}
		}
		lastBlock = block

		blockHash := block.Hash()
		existingBlock := core.ExtendedBlock{}
		if kvstore.Get(blockHash[:], &existingBlock) != nil {
			kvstore.Put(blockHash[:], block)
		}
	}
	if lastBlock.Height != end {
		return nil, fmt.Errorf("Last block height doesn't match, %v : %v", lastBlock.Height, end)
	}
	if count != end-start+1 {
		return nil, fmt.Errorf("Expect %v blocks, but actually got %v", end-start+1, count)
	}

	return lastBlock, nil
}

func getChainBoundary(filename string) (start, end uint64) {
	filename = filename[len("theta_chain-"):]
	idx := strings.Index(filename, "-")
	startStr := filename[:idx]
	start, _ = strconv.ParseUint(startStr, 10, 64)
	filename = filename[idx+1:]
	filename = filename[:strings.Index(filename, "-")]
	end, _ = strconv.ParseUint(filename, 10, 64)
	return
}

func loadState(file *os.File, db database.Database) (*state.StoreView, common.Hash, error) {
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
					return nil, common.Hash{}, fmt.Errorf("Still some storeview unhandled")
				}
				break
			}
			return nil, common.Hash{}, fmt.Errorf("Failed to read snapshot record, %v", err)
		}

		if bytes.Equal(record.K, []byte{core.SVStart}) {
			height := core.Bytestoi(record.V)
			sv := state.NewStoreView(height, common.Hash{}, db)
			svStack = svStack.push(sv)
		} else if bytes.Equal(record.K, []byte{core.SVEnd}) {
			svStack, sv = svStack.pop()
			if sv == nil {
				return nil, common.Hash{}, fmt.Errorf("Missing storeview to handle")
			}
			height := core.Bytestoi(record.V)
			if height != sv.Height() {
				return nil, common.Hash{}, fmt.Errorf("Storeview start and end heights don't match")
			}
			hash = sv.Save()

			if svStack.peek() != nil && height == svStack.peek().Height() {
				// it's a storeview for account storage, verify account
				if account.Root != hash {
					return nil, common.Hash{}, fmt.Errorf("Account storage root doesn't match")
				}
			}
			account = nil
		} else {
			sv := svStack.peek()
			if sv == nil {
				return nil, common.Hash{}, fmt.Errorf("Missing storeview to handle")
			}
			sv.Set(record.K, record.V)

			if account == nil {
				if bytes.HasPrefix(record.K, []byte("ls/a")) {
					acct := &types.Account{}
					err = types.FromBytes([]byte(record.V), acct)
					if err != nil {
						return nil, common.Hash{}, fmt.Errorf("Failed to parse account, %v", err)
					}
					if acct.Root != (common.Hash{}) {
						account = acct
					}
				}
			}
		}
	}

	return sv, hash, nil
}

func checkSnapshot(sv *state.StoreView, metadata *core.SnapshotMetadata, db database.Database) error {
	tailTrio := &metadata.TailTrio
	secondBlock := tailTrio.Second.Header
	expectedStateHash := sv.Hash()
	if bytes.Compare(expectedStateHash.Bytes(), secondBlock.StateHash.Bytes()) != 0 {
		return fmt.Errorf("StateHash not matching: %v vs %s",
			expectedStateHash.Hex(), secondBlock.StateHash.Hex())
	}

	var provenValSet *core.ValidatorSet
	var err error
	if secondBlock.Height != core.GenesisBlockHeight {
		provenValSet, err = checkProofTrios(metadata.ProofTrios, db)
		if err != nil {
			return err
		}
	}

	err = checkTailTrio(sv, provenValSet, tailTrio)
	if err != nil {
		return err
	}

	return nil
}

func checkProofTrios(proofTrios []core.SnapshotBlockTrio, db database.Database) (*core.ValidatorSet, error) {
	logger.Debugf("Check validator set change proofs...")

	var provenValSet *core.ValidatorSet // the proven validator set so far
	var err error
	for idx, blockTrio := range proofTrios {
		first := blockTrio.First
		second := blockTrio.Second
		third := blockTrio.Third
		if idx == 0 {
			// special handling for the genesis block
			provenValSet, err = checkGenesisBlock(&second.Header, db)
			if err != nil {
				return nil, fmt.Errorf("Invalid genesis block: %v", err)
			}
		} else {
			if second.Header.Parent != first.Header.Hash() || third.Header.Parent != second.Header.Hash() {
				return nil, fmt.Errorf("block trio has invalid Parent link")
			}

			if second.Header.HCC.BlockHash != first.Header.Hash() || third.Header.HCC.BlockHash != second.Header.Hash() {
				return nil, fmt.Errorf("block trio has invalid HCC link: %v, %v; %v, %v", first.Header.Hash(), second.Header.HCC.BlockHash,
					second.Header.Hash(), third.Header.HCC.BlockHash)
			}

			// third.Header.HCC.Votes contains the votes for the second block in the trio
			if err := validateVotes(provenValSet, &second.Header, third.Header.HCC.Votes); err != nil {
				return nil, fmt.Errorf("Failed to validate voteSet, %v", err)
			}
			provenValSet, err = getValidatorSetFromVCPProof(first.Header.StateHash, &first.Proof)
			if err != nil {
				return nil, fmt.Errorf("Failed to retrieve validator set from VCP proof: %v", err)
			}
		}

		logger.Debugf("Block height: %v, Currently proven validator set: %v", first.Header.Height, provenValSet)
	}

	return provenValSet, nil
}

func checkTailTrio(sv *state.StoreView, provenValSet *core.ValidatorSet, tailTrio *core.SnapshotBlockTrio) error {
	second := &tailTrio.Second
	third := &tailTrio.Third

	if second.Header.Height == core.GenesisBlockHeight {
		_, err := checkGenesisBlock(&second.Header, sv.GetDB())
		if err != nil {
			return err
		}
	} else {
		validateVotes(provenValSet, &third.Header, third.VoteSet)
		retrievedValSet := getValidatorSetFromSV(sv)
		if !provenValSet.Equals(retrievedValSet) {
			return fmt.Errorf("The latest proven and retrieved validator set does not match")
		}
	}

	return nil
}

func checkGenesisBlock(block *core.BlockHeader, db database.Database) (*core.ValidatorSet, error) {
	if block.Height != core.GenesisBlockHeight {
		return nil, fmt.Errorf("Invalid genesis block height: %v", block.Height)
	}

	var expectedGenesisHash string
	if block.ChainID == core.MainnetChainID {
		expectedGenesisHash = core.MainnetGenesisBlockHash
	} else {
		expectedGenesisHash = viper.GetString(common.CfgGenesisHash)
	}

	logger.Infof("Expected genesis hash: %v", expectedGenesisHash)
	logger.Infof("Acutal   genesis hash: %v", block.Hash().Hex())

	if block.Hash() != common.HexToHash(expectedGenesisHash) {
		return nil, fmt.Errorf("Genesis block hash mismatch, expected: %v, calculated: %v",
			expectedGenesisHash, block.Hash().Hex())
	}

	// now that the block hash matches with the expected genesis block hash,
	// the block and its state trie is considerred valid. We can retrieve the
	// genesis validator set from its state trie
	gsv := state.NewStoreView(block.Height, block.StateHash, db)

	genesisValidatorSet := getValidatorSetFromSV(gsv)

	return genesisValidatorSet, nil
}

func getValidatorSetFromVCPProof(stateHash common.Hash, recoverredVp *core.VCPProof) (*core.ValidatorSet, error) {
	serializedVCP, _, err := trie.VerifyProof(stateHash, state.ValidatorCandidatePoolKey(), recoverredVp)
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

func getValidatorSetFromSV(sv *state.StoreView) *core.ValidatorSet {
	vcp := sv.GetValidatorCandidatePool()
	return consensus.SelectTopStakeHoldersAsValidators(vcp)
}

func validateVotes(validatorSet *core.ValidatorSet, block *core.BlockHeader, voteSet *core.VoteSet) error {
	if !validatorSet.HasMajority(voteSet) {
		return fmt.Errorf("block doesn't have majority votes")
	}
	for _, vote := range voteSet.Votes() {
		res := vote.Validate()
		if !res.IsOK() {
			return fmt.Errorf("vote is not valid, %v", res)
		}
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

func saveTailBlocks(metadata *core.SnapshotMetadata, sv *state.StoreView, kvstore store.Store) *core.BlockHeader {
	tailBlockTrio := &metadata.TailTrio
	firstBlock := core.Block{BlockHeader: &tailBlockTrio.First.Header}
	secondBlock := core.Block{BlockHeader: &tailBlockTrio.Second.Header}
	hl := sv.GetStakeTransactionHeightList()

	if secondBlock.Height != core.GenesisBlockHeight {
		firstExt := core.ExtendedBlock{
			Block:              &firstBlock,
			Status:             core.BlockStatusTrusted, // HCC links between all three blocks
			Children:           []common.Hash{secondBlock.Hash()},
			HasValidatorUpdate: hl.Contains(firstBlock.Height),
		}
		firstBlockHash := firstBlock.BlockHeader.Hash()

		existingFirstExt := core.ExtendedBlock{}
		if kvstore.Get(firstBlockHash[:], &existingFirstExt) != nil {
			kvstore.Put(firstBlockHash[:], firstExt)
		}
	}

	secondExt := core.ExtendedBlock{
		Block:              &secondBlock,
		Status:             core.BlockStatusTrusted,
		Children:           []common.Hash{},
		HasValidatorUpdate: hl.Contains(secondBlock.Height),
	}
	secondBlockHash := secondBlock.BlockHeader.Hash()

	existingSecondExt := core.ExtendedBlock{}
	if kvstore.Get(secondBlockHash[:], &existingSecondExt) != nil {
		kvstore.Put(secondBlockHash[:], secondExt)
	}

	if secondExt.Height != core.GenesisBlockHeight && secondExt.HasValidatorUpdate {
		// TODO: this would lead to mismatch between the proven and retrieved validator set,
		//       need to handle this case properly
		logger.Warnf("The second block in the tail trio contains validator update, may cause valSet mismatch, height: %v", secondBlock.Height)
	}

	return secondBlock.BlockHeader
}
