package execution

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
)

var _ TxExecutor = (*SendTxExecutor)(nil)

// ------------------------------- Send Transaction -----------------------------------

// SendTxExecutor implements the TxExecutor interface
type SendTxExecutor struct {
	state *st.LedgerState
}

// NewSendTxExecutor creates a new instance of SendTxExecutor
func NewSendTxExecutor(state *st.LedgerState) *SendTxExecutor {
	return &SendTxExecutor{
		state: state,
	}
}

func (exec *SendTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
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

	if len(tx.Inputs) == 0 || len(tx.Outputs) == 0 {
		return result.Error("Invalid sendTx, Inputs and/or Outputs are empty")
	}

	numAccountsAffected := uint64(len(tx.Inputs) + len(tx.Outputs))
	if numAccountsAffected > types.MaxAccountsAffectedPerTx {
		return result.Error("Trasaction modifying too many accounts. At most %v accounts are allowed per transaction",
			types.MaxAccountsAffectedPerTx)
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

	blockHeight := view.Height() + 1
	if blockHeight >= common.HeightEnableSmartContract {
		for _, outAcc := range accounts {
			if outAcc.IsASmartContract() {
				return result.Error(
					fmt.Sprintf("Sending Theta/TFuel to a smart contract (%v) through a SendTx transaction is not allowed", outAcc.Address))
			}
		}
	}

	// Validate inputs and outputs, advanced
	signBytes := tx.SignBytes(chainID)
	inTotal, res := validateInputsAdvanced(accounts, signBytes, tx.Inputs, blockHeight)
	if res.IsError() {
		return res
	}

	if minTxFee, success := sanityCheckForSendTxFee(tx.Fee, numAccountsAffected, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	outTotal := sumOutputs(tx.Outputs)
	outPlusFees := outTotal
	outPlusFees = outTotal.Plus(tx.Fee)
	if !inTotal.IsEqual(outPlusFees) {
		return result.Error("Input total (%v) != output total + fees (%v)", inTotal, outPlusFees)
	}

	return result.OK
}

func (exec *SendTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
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

func (exec *SendTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.SendTx)
	return &core.TxInfo{
		Address:           tx.Inputs[0].Address,
		Sequence:          tx.Inputs[0].Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *SendTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.SendTx)
	fee := tx.Fee
	numAccountsAffected := uint64(len(tx.Inputs) + len(tx.Outputs))

	gasSendTxPerAccount := getRegularTxGas(exec.state) / 2
	gasUint64 := gasSendTxPerAccount * numAccountsAffected
	if gasUint64 < 2*gasSendTxPerAccount {
		gasUint64 = 2 * gasSendTxPerAccount // to prevent spamming with invalid transactions, e.g. empty inputs/outputs
	}
	gas := new(big.Int).SetUint64(gasUint64)
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
