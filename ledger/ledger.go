package ledger

import (
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/kvstore"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	mp "github.com/thetatoken/ukulele/mempool"
	"github.com/thetatoken/ukulele/store/database"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "ledger"})

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
	mempool   *mp.Mempool

	mu       *sync.RWMutex // Lock for accessing ledger state.
	state    *st.LedgerState
	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(chainID string, db database.Database, consensus core.ConsensusEngine, valMgr core.ValidatorManager, mempool *mp.Mempool) *Ledger {
	state := st.NewLedgerState(chainID, db)
	executor := exec.NewExecutor(state, consensus, valMgr)
	ledger := &Ledger{
		consensus: consensus,
		valMgr:    valMgr,
		mempool:   mempool,
		mu:        &sync.RWMutex{},
		state:     state,
		executor:  executor,
	}
	return ledger
}

// State returns the state of the ledger
func (ledger *Ledger) State() *st.LedgerState {
	return ledger.state
}

// GetScreenedSnapshot returns a snapshot of screened ledger state to query about accounts, etc.
func (ledger *Ledger) GetScreenedSnapshot() (*st.StoreView, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	return ledger.state.Screened().Copy()
}

// GetDeliveredSnapshot returns a snapshot of delivered ledger state to query about accounts, etc.
func (ledger *Ledger) GetDeliveredSnapshot() (*st.StoreView, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	return ledger.state.Delivered().Copy()
}

// GetFinalizedSnapshot returns a snapshot of finalized ledger state to query about accounts, etc.
func (ledger *Ledger) GetFinalizedSnapshot() (*st.StoreView, error) {
	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	return ledger.state.Finalized().Copy()
}

// GetFinalizedValidatorCandidatePool returns the validator candidate pool of the latest DIRECTLY finalized block
func (ledger *Ledger) GetFinalizedValidatorCandidatePool(blockHash common.Hash) (*core.ValidatorCandidatePool, error) {
	db := ledger.state.DB()
	store := kvstore.NewKVStore(db)

	for i := 0; !blockHash.IsEmpty(); i++ {
		block, err := findBlock(store, blockHash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, fmt.Errorf("Block is nil for hash %v", blockHash)
		}

		//if i >= 2 { // Start checking direct finalization only from the grandparent block
		if block.Status.IsDirectlyFinalized() { // the latest DIRECTLY finalized block found
			stateRoot := block.BlockHeader.StateHash
			storeView := st.NewStoreView(block.Height, stateRoot, db)
			vcp := storeView.GetValidatorCandidatePool()
			return vcp, nil
		}
		//}
		blockHash = block.Parent
	}

	return nil, fmt.Errorf("Failed to find a directly finalized ancestor block for %v", blockHash)
}

func findBlock(store store.Store, blockHash common.Hash) (*core.ExtendedBlock, error) {
	var block core.ExtendedBlock
	err := store.Get(blockHash[:], &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// ScreenTx screens the given transaction
func (ledger *Ledger) ScreenTx(rawTx common.Bytes) (txInfo *core.TxInfo, res result.Result) {
	var tx types.Tx
	tx, err := types.TxFromBytes(rawTx)
	if err != nil {
		return nil, result.Error("Error decoding tx: %v", err)
	}

	if ledger.shouldSkipCheckTx(tx) {
		return nil, result.Error("Unauthorized transaction, should skip").
			WithErrorCode(result.CodeUnauthorizedTx)
	}

	ledger.mu.RLock()
	defer ledger.mu.RUnlock()

	_, res = ledger.executor.ScreenTx(tx)
	if res.IsError() {
		return nil, res
	}

	txInfo, res = ledger.executor.GetTxInfo(tx)
	if res.IsError() {
		return nil, res
	}

	return txInfo, res
}

// ProposeBlockTxs collects and executes a list of transactions, which will be used to assemble the next blockl
// It also clears these transactions from the mempool.
func (ledger *Ledger) ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result) {
	// Must always acquire locks in following order to avoid deadlock: mempool, ledger.
	// Otherwise, could cause deadlock since mempool.InsertTransaction() also first acquires the mempool, and then the ledger lock
	ledger.mempool.Lock()
	defer ledger.mempool.Unlock()

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	view := ledger.state.Checked()

	// Add special transactions
	rawTxCandidates := []common.Bytes{}
	ledger.addSpecialTransactions(view, &rawTxCandidates)

	// Add regular transactions submitted by the clients
	regularRawTxs := ledger.mempool.ReapUnsafe(core.MaxNumRegularTxsPerBlock)
	for _, regularRawTx := range regularRawTxs {
		rawTxCandidates = append(rawTxCandidates, regularRawTx)
	}

	blockRawTxs = []common.Bytes{}
	for _, rawTxCandidate := range rawTxCandidates {
		tx, err := types.TxFromBytes(rawTxCandidate)
		if err != nil {
			continue
		}
		_, res := ledger.executor.CheckTx(tx)
		if res.IsError() {
			logger.Errorf("Transaction check failed: errMsg = %v, tx = %v", res.Message, tx)
			continue
		}
		blockRawTxs = append(blockRawTxs, rawTxCandidate)
	}

	ledger.handleDelayedStateUpdates(view)

	stateRootHash = view.Hash()

	return stateRootHash, blockRawTxs, result.OK
}

// ApplyBlockTxs applies the given block transactions. If any of the transactions failed, it returns
// an error immediately. If all the transactions execute successfully, it then validates the state
// root hash. If the states root hash matches the expected value, it clears the transactions from the mempool
func (ledger *Ledger) ApplyBlockTxs(blockRawTxs []common.Bytes, expectedStateRoot common.Hash) result.Result {
	// Must always acquire locks in following order to avoid deadlock: mempool, ledger.
	// Otherwise, could cause deadlock since mempool.InsertTransaction() also first acquires the mempool, and then the ledger lock
	ledger.mempool.Lock()
	defer ledger.mempool.Unlock()

	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	view := ledger.state.Delivered()

	currHeight := view.Height()
	currStateRoot := view.Hash()

	for _, rawTx := range blockRawTxs {
		tx, err := types.TxFromBytes(rawTx)
		if err != nil {
			ledger.resetState(currHeight, currStateRoot)
			return result.Error("Failed to parse transaction: %v", hex.EncodeToString(rawTx))
		}
		_, res := ledger.executor.ExecuteTx(tx)
		if res.IsError() {
			ledger.resetState(currHeight, currStateRoot)
			return res
		}
	}

	ledger.handleDelayedStateUpdates(view)

	newStateRoot := view.Hash()
	if newStateRoot != expectedStateRoot {
		ledger.resetState(currHeight, currStateRoot)
		return result.Error("State root mismatch! root: %v, exptected: %v",
			hex.EncodeToString(newStateRoot[:]),
			hex.EncodeToString(expectedStateRoot[:]))
	}

	ledger.state.Commit() // commit to persistent storage

	ledger.mempool.UpdateUnsafe(blockRawTxs) // clear txs from the mempool

	return result.OK
}

// ResetState sets the ledger state with the designated root
func (ledger *Ledger) ResetState(height uint64, rootHash common.Hash) result.Result {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	return ledger.resetState(height, rootHash)
}

// FinalizeState sets the ledger state with the finalized root
func (ledger *Ledger) FinalizeState(height uint64, rootHash common.Hash) result.Result {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	res := ledger.state.Finalize(height, rootHash)
	if res.IsError() {
		return result.Error("Failed to finalize state root: %v", hex.EncodeToString(rootHash[:]))
	}
	return result.OK
}

// resetState sets the ledger state with the designated root
func (ledger *Ledger) resetState(height uint64, rootHash common.Hash) result.Result {
	logger.Debugf("Reseting state to height %v, hash %v\n", height, rootHash.Hex())

	res := ledger.state.ResetState(height, rootHash)
	if res.IsError() {
		return result.Error("Failed to set state root: %v", hex.EncodeToString(rootHash[:]))
	}
	return result.OK
}

// CheckTx() should skip all the transactions that can only be initiated by the validators
// i.e., if a regular user submits a coinbaseTx or slashTx, it should be skipped so it will not
// get into the mempool
func (ledger *Ledger) shouldSkipCheckTx(tx types.Tx) bool {
	switch tx.(type) {
	case *types.CoinbaseTx:
		return true
	case *types.SlashTx:
		return true
	default:
		return false
	}
}

// handleDelayedStateUpdates handles delayed state updates, e.g. stake return, where the stake
// is returned only after X blocks of its corresponding StakeWithdraw transaction
func (ledger *Ledger) handleDelayedStateUpdates(view *st.StoreView) {
	ledger.handleStakeReturn(view)
}

func (ledger *Ledger) handleStakeReturn(view *st.StoreView) {
	vcp := view.GetValidatorCandidatePool()
	if vcp == nil {
		return
	}

	currentHeight := view.Height()
	returnedStakes := vcp.ReturnStakes(currentHeight)

	for _, returnedStake := range returnedStakes {
		if !returnedStake.Withdrawn || currentHeight < returnedStake.ReturnHeight {
			panic(fmt.Sprintf("Cannot return stake: withdrawn = %v, returnHeight = %v, currentHeight = %v",
				returnedStake.Withdrawn, returnedStake.ReturnHeight, currentHeight))
		}
		sourceAddress := returnedStake.Source
		sourceAccount := view.GetAccount(sourceAddress)
		if sourceAccount == nil {
			panic(fmt.Sprintf("Failed to retrieve source account for stake return: %v", sourceAddress))
		}
		returnedCoins := types.Coins{
			ThetaWei: returnedStake.Amount,
			GammaWei: types.Zero,
		}
		sourceAccount.Balance = sourceAccount.Balance.Plus(returnedCoins)
		view.SetAccount(sourceAddress, sourceAccount)
	}
	view.UpdateValidatorCandidatePool(vcp)
}

// addSpecialTransactions adds special transactions (e.g. coinbase transaction, slash transaction) to the block
func (ledger *Ledger) addSpecialTransactions(view *st.StoreView, rawTxs *[]common.Bytes) {
	extBlk := ledger.consensus.GetLastFinalizedBlock()
	epoch := ledger.consensus.GetEpoch()
	proposer := ledger.valMgr.GetProposer(extBlk.Hash(), epoch)
	validators := ledger.valMgr.GetValidatorSet(extBlk.Hash()).Validators()

	ledger.addCoinbaseTx(view, &proposer, &validators, rawTxs)
	ledger.addSlashTxs(view, &proposer, &validators, rawTxs)
}

// addCoinbaseTx adds a Coinbase transaction
func (ledger *Ledger) addCoinbaseTx(view *st.StoreView, proposer *core.Validator, validators *[]core.Validator, rawTxs *[]common.Bytes) {
	proposerAddress := proposer.Address
	proposerTxIn := types.TxInput{
		Address: proposerAddress,
	}

	validatorAddresses := make([]common.Address, len(*validators))
	for idx, validator := range *validators {
		validatorAddress := validator.Address
		validatorAddresses[idx] = validatorAddress
	}
	accountRewardMap := exec.CalculateReward(view, validatorAddresses)

	coinbaseTxOutputs := []types.TxOutput{}
	for accountAddressStr, accountReward := range accountRewardMap {
		var accountAddress common.Address
		copy(accountAddress[:], accountAddressStr)
		coinbaseTxOutputs = append(coinbaseTxOutputs, types.TxOutput{
			Address: accountAddress,
			Coins:   accountReward,
		})
	}

	coinbaseTx := &types.CoinbaseTx{
		Proposer:    proposerTxIn,
		Outputs:     coinbaseTxOutputs,
		BlockHeight: ledger.state.Height(),
	}

	signature, err := ledger.signTransaction(coinbaseTx)
	if err != nil {
		logger.Errorf("Failed to add coinbase transaction: %v", err)
		return
	}
	coinbaseTx.SetSignature(proposerAddress, signature)
	coinbaseTxBytes, err := types.TxToBytes(coinbaseTx)
	if err != nil {
		logger.Errorf("Failed to add coinbase transaction: %v", err)
		return
	}

	*rawTxs = append(*rawTxs, coinbaseTxBytes)
	logger.Debugf("Adding coinbase transction: tx: %v, bytes: %v", coinbaseTx, hex.EncodeToString(coinbaseTxBytes))
}

// addsSlashTx adds Slash transactions
func (ledger *Ledger) addSlashTxs(view *st.StoreView, proposer *core.Validator, validators *[]core.Validator, rawTxs *[]common.Bytes) {
	proposerAddress := proposer.Address
	proposerTxIn := types.TxInput{
		Address: proposerAddress,
	}

	slashIntents := view.GetSlashIntents()
	for _, slashIntent := range slashIntents {
		slashTx := &types.SlashTx{
			Proposer:        proposerTxIn,
			SlashedAddress:  slashIntent.Address,
			ReserveSequence: slashIntent.ReserveSequence,
			SlashProof:      slashIntent.Proof,
		}

		signature, err := ledger.signTransaction(slashTx)
		if err != nil {
			logger.Errorf("Failed to add slash transaction: %v", err)
			continue
		}
		slashTx.SetSignature(proposerAddress, signature)
		slashTxBytes, err := types.TxToBytes(slashTx)
		if err != nil {
			logger.Errorf("Failed to add slash transaction: %v", err)
			continue
		}

		*rawTxs = append(*rawTxs, slashTxBytes)
		logger.Debugf("Adding slash transction: tx: %v, bytes: %v", slashTx, hex.EncodeToString(slashTxBytes))
	}
	view.ClearSlashIntents()
}

// signTransaction signs the given transaction
func (ledger *Ledger) signTransaction(tx types.Tx) (*crypto.Signature, error) {
	chainID := ledger.state.GetChainID()
	signBytes := tx.SignBytes(chainID)
	signature, err := ledger.consensus.PrivateKey().Sign(signBytes)
	if err != nil {
		return nil, err
	}
	return signature, nil
}
