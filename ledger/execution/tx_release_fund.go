package execution

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
)

// ------------------------------- ReleaseFundTx Transaction -----------------------------------

// ReleaseFundTxExecutor implements the TxExecutor interface
type ReleaseFundTxExecutor struct {
}

// NewReleaseFundTxExecutor creates a new instance of ReleaseFundTxExecutor
func NewReleaseFundTxExecutor() *ReleaseFundTxExecutor {
	return &ReleaseFundTxExecutor{}
}

func (exec *ReleaseFundTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
	tx := transaction.(*types.ReleaseFundTx)

	// Validate source, basic
	res := tx.Source.ValidateBasic()
	if res.IsErr() {
		return res
	}

	// Get input account
	sourceAccount, success := getInput(view, tx.Source)
	if success.IsErr() {
		return result.ErrBaseUnknownAddress
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source)
	if res.IsErr() {
		log.Infof(fmt.Sprintf("validateSourceAdvanced failed on %X: %v", tx.Source.Address, res))
		return res.PrependLog("in validateSourceAdvanced()")
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.ErrInternalError.PrependLog("invalid fee")
	}

	minimalBalance := types.Coins{tx.Fee}
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("Source did not have enough balance %X", tx.Source.Address))
		return result.ErrBaseInsufficientFunds.AppendLog(fmt.Sprintf("Source balance is %v, but required minimal balance is %v", sourceAccount.Balance, minimalBalance))
	}

	currentBlockHeight := getCurrentBlockHeight()
	reserveSequence := tx.ReserveSequence
	err := sourceAccount.CheckReleaseFund(currentBlockHeight, reserveSequence)
	if err != nil {
		return result.ErrInternalError.AppendLog(err.Error())
	}

	return result.OK
}

func (exec *ReleaseFundTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) result.Result {
	tx := transaction.(*types.ReleaseFundTx)

	sourceInputs := []types.TxInput{tx.Source}
	accounts, success := getInputs(view, sourceInputs)
	if success.IsErr() {
		// TODO: revisit whether we should panic or just log the error.
		return result.ErrBaseUnknownAddress.AppendLog("Failed to get the source account")
	}
	sourceAddress := tx.Source.Address
	sourceAccount := accounts[string(sourceAddress[:])]

	reserveSequence := tx.ReserveSequence

	currentBlockHeight := getCurrentBlockHeight()
	sourceAccount.ReleaseFund(currentBlockHeight, reserveSequence)
	if !chargeFee(sourceAccount, tx.Fee) {
		return result.ErrInternalError.AppendLog("failed to charge transaction fee")
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return result.NewResultOK(txHash[:], "")
}
