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
func NewLedger(consensus core.ConsensusEngine, valMgr core.ValidatorManager, mempool *mp.Mempool) *Ledger {
	return nil // TODO: proper implementation..
}

// CheckTx checks the validity of the given transaction
func (ledger *Ledger) CheckTx(rawTx common.Bytes) result.Result {
	var tx types.Tx
	tx, err := types.TxFromBytes(rawTx)
	if err != nil {
		return result.Error("Error decoding tx: %v", err)
	}

	if ledger.shouldSkipCheckTx(tx) {
		return result.Error("Unauthorized transaction, should skip")
	}

	_, res := ledger.executor.CheckTx(tx)
	return res
}

// ProposeBlockTxs collects and executes a list of transactions, which will be used to assemble the next blockl
// It also clears these transactions from the mempool.
func (ledger *Ledger) ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result) {
	blockRawTxs = []common.Bytes{}

	// Add special transactions
	ledger.addSpecialTransactions(&blockRawTxs)

	// Add regular transactions submitted by the clients
	regularRawTxs := ledger.mempool.Reap(core.MaxNumRegularTxsPerBlock)
	for _, regularRawTx := range regularRawTxs {
		tx, err := types.TxFromBytes(regularRawTx)
		if err != nil {
			continue
		}
		_, res := ledger.executor.ExecuteTx(tx)
		if res.IsOK() {
			blockRawTxs = append(blockRawTxs, regularRawTx)
		}
	}
	stateRootHash = ledger.state.Commit()

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
			ledger.SetRootHash(currHeight, currStateRoot)
			return result.Error("Failed to parse transaction: %v", hex.EncodeToString(rawTx))
		}
		_, res := ledger.executor.ExecuteTx(tx)
		if res.IsError() {
			ledger.SetRootHash(currHeight, currStateRoot)
			return res
		}
	}

	newStateRoot := ledger.state.Delivered().Hash()
	if newStateRoot != expectedStateRoot {
		ledger.SetRootHash(currHeight, currStateRoot)
		return result.Error("State root mismatch! root: %v, exptected: %v",
			hex.EncodeToString(newStateRoot[:]),
			hex.EncodeToString(expectedStateRoot[:]))
	}

	ledger.state.Commit() // commit to persistent storage

	ledger.mempool.Update(blockRawTxs) // clear txs from the mempool

	return result.OK
}

// SetRootHash sets the ledger state with the designated root
func (ledger *Ledger) SetRootHash(height uint32, rootHash common.Hash) result.Result {
	success := ledger.state.SetStateRoot(height, rootHash)
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

	coinbaseTx.SetSignature(proposerAddress, ledger.signTransaction(coinbaseTx))
	coinbaseTxBytes := types.TxToBytes(coinbaseTx)

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
		slashTx.SetSignature(proposerAddress, ledger.signTransaction(slashTx))

		slashTxBytes := types.TxToBytes(slashTx)

		*rawTxs = append(*rawTxs, slashTxBytes)
		log.Debugf("Adding slash transction: tx: %v, bytes: %v", slashTx, hex.EncodeToString(slashTxBytes))
	}
	ledger.state.ClearSlashIntents()
}

// signTransaction signs the given transaction
func (ledger *Ledger) signTransaction(tx types.Tx) *crypto.Signature {
	// chainID := ledger.state.GetChainID()
	// signBytes := tx.SignBytes(chainID)
	signature := crypto.Signature{} // FIXME: use the node's private key to sign the transaction
	return &signature
}
