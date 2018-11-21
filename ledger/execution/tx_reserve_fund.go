package execution

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
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

func (exec *ReserveFundTxExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	tx := transaction.(*types.ReserveFundTx)

	// Validate source, basic
	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get input account
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return result.Error("Failed to get the source account")
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source)
	if res.IsError() {
		log.Infof(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
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

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v GammaWei",
			types.MinimumTransactionFeeGammaWei).WithErrorCode(result.CodeInvalidFee)
	}

	fund := tx.Source.Coins
	collateral := tx.Collateral
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence

	minimalBalance := fund.Plus(collateral).Plus(tx.Fee)
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientFund)
	}

	err := sourceAccount.CheckReserveFund(collateral, fund, duration, reserveSequence)
	if err != nil {
		return result.Error(err.Error()).WithErrorCode(result.CodeReserveFundCheckFailed)
	}

	return result.OK
}

func (exec *ReserveFundTxExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
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

func (exec *ReserveFundTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.ReserveFundTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(types.GasReserveFundTx)
	effectiveGasPrice := new(big.Int).Div(fee.GammaWei, gas)
	return effectiveGasPrice
}
