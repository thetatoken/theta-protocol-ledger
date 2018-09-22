package execution

import (
	"fmt"

	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
)

// ------------------------------- Send Transaction -----------------------------------

// SendTxExecutor implements the TxExecutor interface
type SendTxExecutor struct {
}

// NewSendTxExecutor creates a new instance of SendTxExecutor
func NewSendTxExecutor() *SendTxExecutor {
	return &SendTxExecutor{}
}

func (exec *SendTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
	tx := transaction.(*types.SendTx)

	// Validate inputs and outputs, basic
	res := validateInputsBasic(tx.Inputs)
	if res.IsErr() {
		return res.PrependLog("in validateInputsBasic()")
	}
	res = validateOutputsBasic(tx.Outputs)
	if res.IsErr() {
		return res.PrependLog("in validateOutputsBasic()")
	}

	// Get inputs
	accounts, res := getInputs(view, tx.Inputs)
	if res.IsErr() {
		return res.PrependLog("in getInputs()")
	}

	// Get or make outputs.
	accounts, res = getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsErr() {
		return res.PrependLog("in getOrMakeOutputs()")
	}

	// Validate inputs and outputs, advanced
	signBytes := tx.SignBytes(chainID)
	inTotal, res := validateInputsAdvanced(accounts, signBytes, tx.Inputs)
	if res.IsErr() {
		return res.PrependLog("in validateInputsAdvanced()")
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.ErrInternalError.PrependLog("invalid fee")
	}

	outTotal := sumOutputs(tx.Outputs)
	outPlusFees := outTotal
	fees := types.Coins{tx.Fee}
	outPlusFees = outTotal.Plus(fees)
	if !inTotal.IsEqual(outPlusFees) {
		return result.ErrBaseInvalidOutput.AppendLog(fmt.Sprintf("Input total (%v) != output total + fees (%v)", inTotal, outPlusFees))
	}

	return result.OK
}

func (exec *SendTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) result.Result {
	tx := transaction.(*types.SendTx)

	accounts, res := getInputs(view, tx.Inputs)
	if res.IsErr() {
		return res.PrependLog("in getInputs()")
	}

	accounts, res = getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsErr() {
		return res.PrependLog("in getOrMakeOutputs()")
	}

	adjustByInputs(view, accounts, tx.Inputs)
	adjustByOutputs(view, accounts, tx.Outputs)

	txHash := types.TxID(chainID, tx)
	return result.NewResultOK(txHash[:], "")
}
