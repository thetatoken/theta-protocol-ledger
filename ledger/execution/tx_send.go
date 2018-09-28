package execution

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/ledger/types"
)

var _ TxExecutor = (*SendTxExecutor)(nil)

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
	if res.IsError() {
		return res
	}
	res = validateOutputsBasic(tx.Outputs)
	if res.IsError() {
		return res
	}

	// Get inputs
	accounts, res := getInputs(view, tx.Inputs)
	if res.IsError() {
		return res
	}

	// Get or make outputs.
	accounts, res = getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsError() {
		return res
	}

	// Validate inputs and outputs, advanced
	signBytes := tx.SignBytes(chainID)
	inTotal, res := validateInputsAdvanced(accounts, signBytes, tx.Inputs)
	if res.IsError() {
		return res
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("invalid fee")
	}

	outTotal := sumOutputs(tx.Outputs)
	outPlusFees := outTotal
	fees := types.Coins{tx.Fee}
	outPlusFees = outTotal.Plus(fees)
	if !inTotal.IsEqual(outPlusFees) {
		return result.Error("Input total (%v) != output total + fees (%v)", inTotal, outPlusFees)
	}

	return result.OK
}

func (exec *SendTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.SendTx)

	accounts, res := getInputs(view, tx.Inputs)
	if res.IsError() {
		return common.Hash{}, res
	}

	accounts, res = getOrMakeOutputs(view, accounts, tx.Outputs)
	if res.IsError() {
		return common.Hash{}, res
	}

	adjustByInputs(view, accounts, tx.Inputs)
	adjustByOutputs(view, accounts, tx.Outputs)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}
