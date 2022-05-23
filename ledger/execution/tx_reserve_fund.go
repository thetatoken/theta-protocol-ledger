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

var _ TxExecutor = (*ReserveFundTxExecutor)(nil)

// ------------------------------- ReserveFundTx Transaction -----------------------------------

// ReserveFundTxExecutor implements the TxExecutor interface
type ReserveFundTxExecutor struct {
	state *st.LedgerState
}

// NewReserveFundTxExecutor creates a new instance of ReserveFundTxExecutor
func NewReserveFundTxExecutor(state *st.LedgerState) *ReserveFundTxExecutor {
	return &ReserveFundTxExecutor{
		state: state,
	}
}

func (exec *ReserveFundTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
	tx := transaction.(*types.ReserveFundTx)

	// Validate source, basic
	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get input account
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return result.Error("Failed to get the source account: %v", tx.Source.Address)
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source, blockHeight)
	if res.IsError() {
		logger.Debugf(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
		return res
	}

	coins := tx.Source.Coins.NoNil()

	if !coins.IsPositive() {
		return result.Error("Amount of reserved fund not specified").
			WithErrorCode(result.CodeReservedFundNotSpecified)
	}

	if coins.ThetaWei.Cmp(types.Zero) != 0 {
		return result.Error("Cannot reserve Theta as service fund!").
			WithErrorCode(result.CodeInvalidFundToReserve)
	}

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	fund := tx.Source.Coins
	collateral := tx.Collateral
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence

	minimalBalance := fund.Plus(collateral).Plus(tx.Fee)
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("ReserveFund: Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("Insufficient fund: Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientFund)
	}

	err := sourceAccount.CheckReserveFund(collateral, fund, duration, reserveSequence)
	if err != nil {
		return result.Error(err.Error()).WithErrorCode(result.CodeReserveFundCheckFailed)
	}

	return result.OK
}

func (exec *ReserveFundTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.ReserveFundTx)

	sourceAddress := tx.Source.Address
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return common.Hash{}, result.Error("Failed to get the source account")
	}

	collateral := tx.Collateral
	fund := tx.Source.Coins
	resourceIDs := tx.ResourceIDs
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence
	endBlockHeight := exec.state.Height() + duration

	sourceAccount.ReserveFund(collateral, fund, resourceIDs, endBlockHeight, reserveSequence)
	if !chargeFee(sourceAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *ReserveFundTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.ReserveFundTx)
	return &core.TxInfo{
		Address:           tx.Source.Address,
		Sequence:          tx.Source.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *ReserveFundTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.ReserveFundTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
