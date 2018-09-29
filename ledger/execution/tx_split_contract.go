package execution

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
)

var _ TxExecutor = (*SplitContractTxExecutor)(nil)

// ------------------------------- SplitContract Transaction -----------------------------------

// SplitContractTxExecutor implements the TxExecutor interface
type SplitContractTxExecutor struct {
	state *st.LedgerState
}

// NewSplitContractTxExecutor creates a new instance of SplitContractTxExecutor
func NewSplitContractTxExecutor(state *st.LedgerState) *SplitContractTxExecutor {
	return &SplitContractTxExecutor{
		state: state,
	}
}

func (exec *SplitContractTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
	tx := transaction.(*types.SplitContractTx)

	res := tx.Initiator.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get inputs
	initiatorAccount, res := getInput(view, tx.Initiator)
	if res.IsError() {
		return res
	}

	// Validate inputs and outputs, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(initiatorAccount, signBytes, tx.Initiator)
	if res.IsError() {
		return res
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("invalid fee")
	}

	minimalBalance := types.Coins{tx.Fee}
	if !initiatorAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("the contract initiator did not have enough to cover the fee %X", tx.Initiator.Address))
		return result.Error("the contract initiator account balance is %v, but required minimal balance is %v", initiatorAccount.Balance, minimalBalance)
	}

	totalPercentage := uint(0)
	for _, split := range tx.Splits {
		percentage := split.Percentage
		if percentage < 0 {
			return result.Error("Percentage needs to be positive")
		}
		if percentage > 100 {
			return result.Error("Percentage needs to be less than 100")
		}
		totalPercentage += percentage
	}

	if totalPercentage > 100 {
		return result.Error("Sum of the percentages should be at most 100")
	}

	resourceId := tx.ResourceId
	if exec.state.SplitContractExists(resourceId) {
		splitContract := exec.state.GetSplitContract(resourceId)
		if splitContract.InitiatorAddress == tx.Initiator.Address {
			return result.Error("Cannot create multiple split contracts for the same resourceId")
		}
	}

	return result.OK
}

func (exec *SplitContractTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.SplitContractTx)

	initiatorAccount, res := getInput(view, tx.Initiator)
	if res.IsError() {
		return common.Hash{}, res
	}

	currentBlockHeight := exec.state.Height()
	exec.state.DeleteExpiredSplitContracts(currentBlockHeight)

	resourceId := tx.ResourceId
	success := false
	if exec.state.SplitContractExists(resourceId) {
		splitContract := exec.state.GetSplitContract(resourceId)
		if splitContract.InitiatorAddress == tx.Initiator.Address {
			return common.Hash{}, result.Error("split contract from a different initiator existed")
		}
		endBlockHeight := currentBlockHeight + tx.Duration
		splitContract.EndBlockHeight = endBlockHeight
		splitContract.Splits = tx.Splits
		success = exec.state.UpdateSplitContract(splitContract)
	} else {
		endBlockHeight := currentBlockHeight + tx.Duration
		splitContract := types.SplitContract{
			InitiatorAddress: tx.Initiator.Address,
			ResourceId:       tx.ResourceId,
			Splits:           tx.Splits,
			EndBlockHeight:   endBlockHeight,
		}
		success = exec.state.AddSplitContract(&splitContract)
	}

	if !success {
		return common.Hash{}, result.Error("failed to add or update split contract")
	}

	if !chargeFee(initiatorAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	initiatorAccount.Sequence++
	view.SetAccount(tx.Initiator.Address, initiatorAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}
