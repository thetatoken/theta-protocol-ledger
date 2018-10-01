package execution

import (
	"fmt"
	"strings"

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

func (exec *ReserveFundTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
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
		log.Infof(fmt.Sprintf("validateSourceAdvanced failed on %X: %v", tx.Source.Address, res))
		return res
	}

	for _, coin := range tx.Source.Coins {
		if strings.Compare(coin.Denom, types.DenomGammaWei) != 0 {
			return result.Error("Cannot reserve %s as service fund!", coin.Denom)
		}
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("invalid fee")
	}

	fund := tx.Source.Coins
	collateral := tx.Collateral
	duration := tx.Duration
	reserveSequence := tx.Source.Sequence

	minimalBalance := fund.Plus(collateral).Plus(types.Coins{tx.Fee})
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("Source did not have enough balance %X", tx.Source.Address))
		return result.Error("Source balance is %v, but required minimal balance is %v", sourceAccount.Balance, minimalBalance)
	}

	err := sourceAccount.CheckReserveFund(collateral, fund, duration, reserveSequence)
	if err != nil {
		return result.Error(err.Error())
	}

	return result.OK
}

func (exec *ReserveFundTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) (common.Hash, result.Result) {
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
