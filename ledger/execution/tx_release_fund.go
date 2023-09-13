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

var _ TxExecutor = (*ReleaseFundTxExecutor)(nil)

// ------------------------------- ReleaseFundTx Transaction -----------------------------------

// ReleaseFundTxExecutor implements the TxExecutor interface
type ReleaseFundTxExecutor struct {
	state *st.LedgerState
}

// NewReleaseFundTxExecutor creates a new instance of ReleaseFundTxExecutor
func NewReleaseFundTxExecutor(state *st.LedgerState) *ReleaseFundTxExecutor {
	return &ReleaseFundTxExecutor{
		state: state,
	}
}

func (exec *ReleaseFundTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
	tx := transaction.(*types.ReleaseFundTx)

	// Validate source, basic
	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get input account
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return result.Error("Unknown address: %v", tx.Source.Address)
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source, blockHeight)
	if res.IsError() {
		logger.Debugf(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
		return res
	}

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	minimalBalance := tx.Fee
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientFund)
	}

	currentBlockHeight := exec.state.Height()
	reserveSequence := tx.ReserveSequence
	err := sourceAccount.CheckReleaseFund(currentBlockHeight, reserveSequence)
	if err != nil {
		return result.Error(err.Error()).WithErrorCode(result.CodeReleaseFundCheckFailed)
	}

	return result.OK
}

func (exec *ReleaseFundTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.ReleaseFundTx)

	sourceInputs := []types.TxInput{tx.Source}
	accounts, success := getInputs(view, sourceInputs)
	if success.IsError() {
		// TODO: revisit whether we should panic or just log the error.
		return common.Hash{}, result.Error("Failed to get the source account")
	}
	sourceAddress := tx.Source.Address
	sourceAccount := accounts[string(sourceAddress[:])]

	reserveSequence := tx.ReserveSequence

	currentBlockHeight := exec.state.Height()
	sourceAccount.ReleaseFund(currentBlockHeight, reserveSequence)
	if !chargeFee(sourceAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *ReleaseFundTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.ReleaseFundTx)
	return &core.TxInfo{
		Address:           tx.Source.Address,
		Sequence:          tx.Source.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *ReleaseFundTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.ReleaseFundTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
