package ledger

import (
	"encoding/hex"

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

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
	mempool   *mp.Mempool

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
		state:     state,
		executor:  executor,
	}
	return ledger
}

// ScreenTx screens the given transaction
func (ledger *Ledger) ScreenTx(rawTx common.Bytes) result.Result {
	var tx types.Tx
	tx, err := types.TxFromBytes(rawTx)
	if err != nil {
		return result.Error("Error decoding tx: %v", err)
	}

	if ledger.shouldSkipCheckTx(tx) {
		return result.Error("Unauthorized transaction, should skip").
			WithErrorCode(result.CodeUnauthorizedTx)
	}

	_, res := ledger.executor.ScreenTx(tx)
	return res
}

// ProposeBlockTxs collects and executes a list of transactions, which will be used to assemble the next blockl
// It also clears these transactions from the mempool.
func (ledger *Ledger) ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result) {

	// Add special transactions
	rawTxCandidates := []common.Bytes{}
	ledger.addSpecialTransactions(&rawTxCandidates)

	// Add regular transactions submitted by the clients
	regularRawTxs := ledger.mempool.Reap(core.MaxNumRegularTxsPerBlock)
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
			log.Errorf("Transaction check failed: errMsg = %v, tx = %v", res.Message, tx)
			continue
		}
		blockRawTxs = append(blockRawTxs, rawTxCandidate)
	}

	stateRootHash = ledger.state.Checked().Hash()
	ledger.mempool.Update(regularRawTxs) // clear txs from the mempool

	return stateRootHash, blockRawTxs, result.OK
}

// ApplyBlockTxs applies the given block transactions. If any of the transactions failed, it returns
// an error immediately. If all the transactions execute successfully, it then validates the state
// root hash. If the states root hash matches the expected value, it clears the transactions from the mempool
func (ledger *Ledger) ApplyBlockTxs(blockRawTxs []common.Bytes, expectedStateRoot common.Hash) result.Result {
	currHeight := ledger.state.Height()
	currStateRoot := ledger.state.Delivered().Hash()
	for _, rawTx := range blockRawTxs {
		tx, err := types.TxFromBytes(rawTx)
		if err != nil {
			ledger.ResetState(currHeight, currStateRoot)
			return result.Error("Failed to parse transaction: %v", hex.EncodeToString(rawTx))
		}
		_, res := ledger.executor.ExecuteTx(tx)
		if res.IsError() {
			ledger.ResetState(currHeight, currStateRoot)
			return res
		}
	}

	newStateRoot := ledger.state.Delivered().Hash()
	if newStateRoot != expectedStateRoot {
		ledger.ResetState(currHeight, currStateRoot)
		return result.Error("State root mismatch! root: %v, exptected: %v",
			hex.EncodeToString(newStateRoot[:]),
			hex.EncodeToString(expectedStateRoot[:]))
	}

	ledger.state.Commit() // commit to persistent storage

	ledger.mempool.Update(blockRawTxs) // clear txs from the mempool

	return result.OK
}

// ResetState sets the ledger state with the designated root
func (ledger *Ledger) ResetState(height uint64, rootHash common.Hash) result.Result {
	success := ledger.state.ResetState(height, rootHash)
	if !success {
		return result.Error("Failed to set state root: %v", hex.EncodeToString(rootHash[:]))
	}
	return result.OK
}

// Query returns the account query results
func (ledger *Ledger) Query() {
	// TODO: implementation..
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

// addSpecialTransactions adds special transactions (e.g. coinbase transaction, slash transaction) to the block
func (ledger *Ledger) addSpecialTransactions(rawTxs *[]common.Bytes) {
	epoch := ledger.consensus.GetEpoch()
	proposer := ledger.valMgr.GetProposerForEpoch(epoch)
	validators := ledger.valMgr.GetValidatorSetForEpoch(epoch).Validators()

	ledger.addCoinbaseTx(&proposer, &validators, rawTxs)
	ledger.addSlashTxs(&proposer, &validators, rawTxs)
}

// addCoinbaseTx adds a Coinbase transaction
func (ledger *Ledger) addCoinbaseTx(proposer *core.Validator, validators *[]core.Validator, rawTxs *[]common.Bytes) {
	proposerAddress := proposer.Address()
	proposerPubKey := proposer.PublicKey()
	proposerTxIn := types.TxInput{
		Address: proposerAddress,
		PubKey:  &proposerPubKey,
	}

	validatorAddresses := make([]common.Address, len(*validators))
	for idx, validator := range *validators {
		validatorAddress := validator.Address()
		validatorAddresses[idx] = validatorAddress
	}
	accountRewardMap := exec.CalculateReward(ledger.state, validatorAddresses)

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
		log.Errorf("Failed to add coinbase transaction: %v", err)
		return
	}
	coinbaseTx.SetSignature(proposerAddress, signature)
	coinbaseTxBytes, err := types.TxToBytes(coinbaseTx)
	if err != nil {
		log.Errorf("Failed to add coinbase transaction: %v", err)
		return
	}

	*rawTxs = append(*rawTxs, coinbaseTxBytes)
	log.Debugf("Adding coinbase transction: tx: %v, bytes: %v", coinbaseTx, hex.EncodeToString(coinbaseTxBytes))
}

// addsSlashTx adds Slash transactions
func (ledger *Ledger) addSlashTxs(proposer *core.Validator, validators *[]core.Validator, rawTxs *[]common.Bytes) {
	proposerAddress := proposer.Address()
	proposerPubKey := proposer.PublicKey()
	proposerTxIn := types.TxInput{
		Address: proposerAddress,
		PubKey:  &proposerPubKey,
	}

	slashIntents := ledger.state.GetSlashIntents()
	for _, slashIntent := range slashIntents {
		slashTx := &types.SlashTx{
			Proposer:        proposerTxIn,
			SlashedAddress:  slashIntent.Address,
			ReserveSequence: slashIntent.ReserveSequence,
			SlashProof:      slashIntent.Proof,
		}

		signature, err := ledger.signTransaction(slashTx)
		if err != nil {
			log.Errorf("Failed to add slash transaction: %v", err)
			continue
		}
		slashTx.SetSignature(proposerAddress, signature)
		slashTxBytes, err := types.TxToBytes(slashTx)
		if err != nil {
			log.Errorf("Failed to add slash transaction: %v", err)
			continue
		}

		*rawTxs = append(*rawTxs, slashTxBytes)
		log.Debugf("Adding slash transction: tx: %v, bytes: %v", slashTx, hex.EncodeToString(slashTxBytes))
	}
	ledger.state.ClearSlashIntents()
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
