package execution

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
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

func (exec *SendTxExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
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
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v GammaWei",
			types.MinimumTransactionFeeGammaWei).WithErrorCode(result.CodeInvalidFee)
	}

	outTotal := sumOutputs(tx.Outputs)
	outPlusFees := outTotal
	outPlusFees = outTotal.Plus(tx.Fee)
	if !inTotal.IsEqual(outPlusFees) {
		return result.Error("Input total (%v) != output total + fees (%v)", inTotal, outPlusFees)
	}

	return result.OK
}

func (exec *SendTxExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
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

func (exec *SendTxExecutor) calculateFee(transaction types.Tx) (types.Coins, error) {
	tx := transaction.(*types.SendTx)
	fee := tx.Fee
	return fee, nil
}
