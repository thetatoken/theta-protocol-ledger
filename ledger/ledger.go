package ledger

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"sync"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/kvstore"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	exec "github.com/thetatoken/theta/ledger/execution"
	"github.com/thetatoken/theta/ledger/state"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	mp "github.com/thetatoken/theta/mempool"
	"github.com/thetatoken/theta/store/database"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "ledger"})

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	chain     *blockchain.Chain
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
	mempool   *mp.Mempool

	mu       *sync.RWMutex // Lock for accessing ledger state.
	state    *st.LedgerState
	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(chainID string, db database.Database, chain *blockchain.Chain, consensus core.ConsensusEngine, valMgr core.ValidatorManager, mempool *mp.Mempool) *Ledger {
	state := st.NewLedgerState(chainID, db)
	executor := exec.NewExecutor(state, consensus, valMgr)
	ledger := &Ledger{
		chain:     chain,
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
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	return ledger.state.Screened().Copy()
}

// GetDeliveredSnapshot returns a snapshot of delivered ledger state to query about accounts, etc.
func (ledger *Ledger) GetDeliveredSnapshot() (*st.StoreView, error) {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	return ledger.state.Delivered().Copy()
}

// GetFinalizedSnapshot returns a snapshot of finalized ledger state to query about accounts, etc.
func (ledger *Ledger) GetFinalizedSnapshot() (*st.StoreView, error) {
	ledger.mu.Lock()
	defer ledger.mu.Unlock()

	return ledger.state.Finalized().Copy()
}

// GetFinalizedValidatorCandidatePool returns the validator candidate pool of the latest DIRECTLY finalized block
func (ledger *Ledger) GetFinalizedValidatorCandidatePool(blockHash common.Hash, isNext bool) (*core.ValidatorCandidatePool, error) {
	db := ledger.state.DB()
	store := kvstore.NewKVStore(db)

	var i int
	if isNext {
		i = 1
	} else {
		i = 2
	}
	for ; ; i-- {
		block, err := findBlock(store, blockHash)
		if err != nil {
			return nil, err
		}
		if block == nil {
			return nil, fmt.Errorf("Block is nil for hash %v", blockHash.Hex())
		}

		// Grandparent or root block.
		if i == 0 || block.HCC.BlockHash.IsEmpty() || block.Status.IsTrusted() {
			stateRoot := block.BlockHeader.StateHash
			storeView := st.NewStoreView(block.Height, stateRoot, db)
			vcp := storeView.GetValidatorCandidatePool()
			return vcp, nil
		}
		blockHash = block.HCC.BlockHash
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

	hasValidatorUpdate := false
	for _, rawTx := range blockRawTxs {
		tx, err := types.TxFromBytes(rawTx)
		if err != nil {
			ledger.resetState(currHeight, currStateRoot)
			return result.Error("Failed to parse transaction: %v", hex.EncodeToString(rawTx))
		}
		if _, ok := tx.(*types.DepositStakeTx); ok {
			hasValidatorUpdate = true
		} else if _, ok := tx.(*types.WithdrawStakeTx); ok {
			hasValidatorUpdate = true
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

	return result.OKWith(result.Info{"hasValidatorUpdate": hasValidatorUpdate})
}

// PruneState attempts to prune the state up to the targetEndHeight
func (ledger *Ledger) PruneState(targetEndHeight uint64) error {
	var processedHeight uint64
	db := ledger.State().DB()
	kvStore := kvstore.NewKVStore(db)
	err := kvStore.Get(state.StatePruningProgressKey(), &processedHeight)
	if err != nil {
		processedHeight = ledger.chain.Root().Height
	}

	pruneInterval := uint64(viper.GetInt(common.CfgStorageStatePruningInterval))
	maxHeightsToPrune := 3 * pruneInterval // prune too many heights at once could cause hang, should catchup gradually
	endHeight := processedHeight + maxHeightsToPrune
	if endHeight > targetEndHeight {
		endHeight = targetEndHeight
	}

	startHeight := processedHeight + 1
	if endHeight < startHeight {
		errMsg := fmt.Sprintf("endHeight (%v) < startHeight (%v)", endHeight, startHeight)
		logger.Warnf(errMsg)
		return fmt.Errorf(errMsg)
	}

	lastFinalizedBlock := ledger.consensus.GetLastFinalizedBlock()
	if endHeight >= lastFinalizedBlock.Height {
		errMsg := fmt.Sprintf("Can't prune at height >= %v yet", lastFinalizedBlock.Height)
		logger.Warnf(errMsg)
		return fmt.Errorf(errMsg)
	}

	// Need to save the progress before pruning -- in case the program exits during pruning (e.g. Ctrl+C),
	// the states that are already pruned do not get pruned again
	kvStore.Put(state.StatePruningProgressKey(), endHeight)

	err = ledger.pruneStateForRange(startHeight, endHeight)
	if err != nil {
		logger.Warnf("Unable to pruning state: %v", err)
		return err
	}

	return nil
}

// pruneStateForRange prunes states from startHeight to endHeight (inclusive for both end)
func (ledger *Ledger) pruneStateForRange(startHeight, endHeight uint64) error {
	logger.Infof("Prune state from height %v to %v", startHeight, endHeight)

	db := ledger.State().DB()
	consensus := ledger.consensus
	chain := ledger.chain
	lastFinalizedBlock := consensus.GetLastFinalizedBlock()

	sv := state.NewStoreView(lastFinalizedBlock.Height, lastFinalizedBlock.BlockHeader.StateHash, db)

	stateHashMap := make(map[string]bool)
	kvStore := kvstore.NewKVStore(db)
	hl := sv.GetStakeTransactionHeightList().Heights
	for _, height := range hl {
		// check kvstore first
		blockTrio := &core.SnapshotBlockTrio{}
		blockTrioKey := []byte(core.BlockTrioStoreKeyPrefix + strconv.FormatUint(height, 10))
		err := kvStore.Get(blockTrioKey, blockTrio)
		if err == nil {
			stateHashMap[blockTrio.First.Header.StateHash.String()] = true
			continue
		}

		if height == core.GenesisBlockHeight {
			blocks := chain.FindBlocksByHeight(core.GenesisBlockHeight)
			genesisBlock := blocks[0]
			stateHashMap[genesisBlock.StateHash.String()] = true
		} else {
			blocks := chain.FindBlocksByHeight(height)
			for _, block := range blocks {
				if block.Status.IsDirectlyFinalized() {
					stateHashMap[block.StateHash.String()] = true
					break
				}
			}
		}
	}

	for height := endHeight; height >= startHeight && height > 0; height-- {
		blocks := chain.FindBlocksByHeight(height)
		for idx, block := range blocks {
			if _, ok := stateHashMap[block.StateHash.String()]; !ok {
				if block.HasValidatorUpdate {
					continue
				}

				if block.Status.IsPending() || block.Status.IsInvalid() || block.Status.IsTrusted() {
					continue // This could happen if the block is stored in the chain but its
					// txs were not processed (e.g. an invalid block). In such cases the block
					// is stored in the chain, but its state trie is not saved
				}

				logger.Infof("Prune state, idx: %v, height: %v, StateHash: %v", idx, height, block.StateHash.Hex())
				_, err := db.Get(block.StateHash[:])
				if err != nil {
					logger.Errorf("StateRoot %v not found, skip pruning", block.StateHash.Hex())
					continue
				}

				sv := state.NewStoreView(height, block.StateHash, db)
				err = sv.Prune()
				if err != nil {
					return fmt.Errorf("Failed to prune storeview at height %v, %v", height, err)
				}
			}
		}
	}

	return nil
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
			TFuelWei: types.Zero,
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
	//ledger.addSlashTxs(view, &proposer, &validators, rawTxs)
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
