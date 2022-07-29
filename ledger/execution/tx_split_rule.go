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

var _ TxExecutor = (*SplitRuleTxExecutor)(nil)

// ------------------------------- SplitRule Transaction -----------------------------------

// SplitRuleTxExecutor implements the TxExecutor interface
type SplitRuleTxExecutor struct {
	state *st.LedgerState
}

// NewSplitRuleTxExecutor creates a new instance of SplitRuleTxExecutor
func NewSplitRuleTxExecutor(state *st.LedgerState) *SplitRuleTxExecutor {
	return &SplitRuleTxExecutor{
		state: state,
	}
}

func (exec *SplitRuleTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
	tx := transaction.(*types.SplitRuleTx)

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
	res = validateInputAdvanced(initiatorAccount, signBytes, tx.Initiator, blockHeight)
	if res.IsError() {
		return res
	}

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	minimalBalance := tx.Fee
	if !initiatorAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("the contract initiator did not have enough to cover the fee %X", tx.Initiator.Address))
		return result.Error("the contract initiator account balance is %v, but required minimal balance is %v", initiatorAccount.Balance, minimalBalance)
	}

	numAccountsAffected := len(tx.Splits) + 1
	if numAccountsAffected > types.MaxAccountsAffectedPerTx {
		return result.Error("This allows one trasaction to modify many accounts. At most %v accounts are allowed per transaction.",
			types.MaxAccountsAffectedPerTx)
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

	resourceID := tx.ResourceID
	if view.SplitRuleExists(resourceID) {
		splitRule := view.GetSplitRule(resourceID)
		if splitRule.InitiatorAddress != tx.Initiator.Address {
			return result.Error("Cannot create multiple split rules for the same resourceID").
				WithErrorCode(result.CodeUnauthorizedToUpdateSplitRule)
		}
	}

	return result.OK
}

func (exec *SplitRuleTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.SplitRuleTx)

	initiatorAccount, res := getInput(view, tx.Initiator)
	if res.IsError() {
		return common.Hash{}, res
	}

	if !chargeFee(initiatorAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	currentBlockHeight := view.Height()
	view.DeleteExpiredSplitRules(currentBlockHeight)

	resourceID := tx.ResourceID
	success := false
	if view.SplitRuleExists(resourceID) {
		splitRule := view.GetSplitRule(resourceID)
		if splitRule.InitiatorAddress != tx.Initiator.Address {
			return common.Hash{}, result.Error("split rule from a different initiator existed").
				WithErrorCode(result.CodeUnauthorizedToUpdateSplitRule)
		}
		endBlockHeight := currentBlockHeight + tx.Duration
		splitRule.EndBlockHeight = endBlockHeight
		splitRule.Splits = tx.Splits
		success = view.UpdateSplitRule(splitRule)
	} else {
		endBlockHeight := currentBlockHeight + tx.Duration
		splitRule := types.SplitRule{
			InitiatorAddress: tx.Initiator.Address,
			ResourceID:       tx.ResourceID,
			Splits:           tx.Splits,
			EndBlockHeight:   endBlockHeight,
		}
		success = view.AddSplitRule(&splitRule)
	}

	if !success {
		return common.Hash{}, result.Error("failed to add or update split rule")
	}

	initiatorAccount.Sequence++
	view.SetAccount(tx.Initiator.Address, initiatorAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *SplitRuleTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.SplitRuleTx)
	return &core.TxInfo{
		Address:           tx.Initiator.Address,
		Sequence:          tx.Initiator.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *SplitRuleTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.SplitRuleTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
