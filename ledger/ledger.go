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
	mempool   *mp.Mempool

	state    *st.LedgerState
	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(consensus core.ConsensusEngine, mempool *mp.Mempool) *Ledger {
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

	_, res := ledger.executor.ExecuteTx(tx, true) // Sanity check only
	return res
}

// DeliverBlockTxs executes and returns a list of transactions,
// which will be used to assemble the next block
func (ledger *Ledger) DeliverBlockTxs() (blockRawTxs []common.Bytes, res result.Result) {
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
		ledger.executor.ExecuteTx(tx, false)
	}
	ledger.mempool.Update(regularRawTxs) // clear txs from the mempool

	return blockRawTxs, result.OK
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
	vaMgr := ledger.consensus.GetValidatorManager()
	proposer := vaMgr.GetProposerForEpoch(epoch)
	validators := vaMgr.GetValidatorSetForEpoch(epoch).Validators()

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
		BlockHeight: exec.GetCurrentBlockHeight(),
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
