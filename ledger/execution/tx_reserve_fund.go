package execution

import (
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
)

// ------------------------------- ReserveFundTx Transaction -----------------------------------

// ReserveFundTxExecutor implements the TxExecutor interface
type ReserveFundTxExecutor struct {
}

// NewReserveFundTxExecutor creates a new instance of ReserveFundTxExecutor
func NewReserveFundTxExecutor() *ReserveFundTxExecutor {
	return &ReserveFundTxExecutor{}
}

func (exec *ReserveFundTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
	tx := transaction.(*types.ReserveFundTx)

	// Validate source, basic
	res := tx.Source.ValidateBasic()
	if res.IsErr() {
		return res
	}

	// Get input account
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsErr() {
		return result.ErrBaseUnknownAddress.AppendLog("Failed to get the source account")
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source)
	if res.IsErr() {
		log.Infof(fmt.Sprintf("validateSourceAdvanced failed on %X: %v", tx.Source.Address, res))
		return res.PrependLog("in validateSourceAdvanced()")
	}

	for _, coin := range tx.Source.Coins {
		if strings.Compare(coin.Denom, types.DenomGammaWei) != 0 {
			return result.ErrBaseInvalidInput.AppendLog(fmt.Sprintf("Cannot reserve %s as service fund!", coin.Denom))
		}
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.ErrInternalError.PrependLog("invalid fee")
	}

	fund := tx.Source.Coins
	collateral := tx.Collateral
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence

	minimalBalance := fund.Plus(collateral).Plus(types.Coins{tx.Fee})
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("Source did not have enough balance %X", tx.Source.Address))
		return result.ErrBaseInsufficientFunds.AppendLog(fmt.Sprintf("Source balance is %v, but required minimal balance is %v", sourceAccount.Balance, minimalBalance))
	}

	err := sourceAccount.CheckReserveFund(collateral, fund, duration, reserveSequence)
	if err != nil {
		return result.ErrInternalError.AppendLog(err.Error())
	}

	return result.OK
}

func (exec *ReserveFundTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) result.Result {
	tx := transaction.(*types.ReserveFundTx)

	sourceAddress := tx.Source.Address
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsErr() {
		return result.ErrBaseUnknownAddress.AppendLog("Failed to get the source account")
	}

	collateral := tx.Collateral
	fund := tx.Source.Coins
	resourceIds := tx.ResourceIds
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence
	endBlockHeight := getCurrentBlockHeight() + duration

	sourceAccount.ReserveFund(collateral, fund, resourceIds, endBlockHeight, reserveSequence)
	if !chargeFee(sourceAccount, tx.Fee) {
		return result.ErrInternalError.AppendLog("failed to charge transaction fee")
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return result.NewResultOK(txHash[:], "")
}
