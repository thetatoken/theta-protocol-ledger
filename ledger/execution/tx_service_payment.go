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

var _ TxExecutor = (*ServicePaymentTxExecutor)(nil)

// ------------------------------- ServicePayment Transaction -----------------------------------

// ServicePaymentTxExecutor implements the TxExecutor interface
type ServicePaymentTxExecutor struct {
	state *st.LedgerState
}

// NewServicePaymentTxExecutor creates a new instance of ServicePaymentTxExecutor
func NewServicePaymentTxExecutor(state *st.LedgerState) *ServicePaymentTxExecutor {
	return &ServicePaymentTxExecutor{
		state: state,
	}
}

func (exec *ServicePaymentTxExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	tx := transaction.(*types.ServicePaymentTx)

	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	res = tx.Target.ValidateBasic()
	if res.IsError() {
		return res
	}

	sourceAddress := tx.Source.Address
	targetAddress := tx.Target.Address

	if sourceAddress == targetAddress {
		return result.Error("Source and target address for the service payment cannot be identical: %v", sourceAddress)
	}

	sourceAccount, res := getInput(view, tx.Source)
	if res.IsError() {
		return res
	}

	// Get the target account (that signed and broadcasted this transaction)
	targetAccount, res := getOrMakeInput(view, tx.Target)
	if res.IsError() {
		return res
	}

	if tx.Source.Coins.ThetaWei.Cmp(types.Zero) != 0 {
		return result.Error("Cannot send ThetaWei as service payment!")
	}

	// Verify source
	sourceSignBytes := tx.SourceSignBytes(chainID)
	if !tx.Source.Signature.Verify(sourceSignBytes, sourceAccount.Address) {
		errMsg := fmt.Sprintf("sanityCheckForServicePaymentTx failed on source signature, addr: %v", sourceAddress.Hex())
		logger.Infof(errMsg)
		return result.Error(errMsg)
	}

	targetSignBytes := tx.TargetSignBytes(chainID)
	if !tx.Target.Signature.Verify(targetSignBytes, targetAccount.Address) {
		errMsg := fmt.Sprintf("sanityCheckForServicePaymentTx failed on target signature, addr: %v", targetAddress.Hex())
		logger.Infof(errMsg)
		return result.Error(errMsg)
	}

	blockHeight := view.Height() + 1 // the view points to the parent of the current block
	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	transferAmount := tx.Source.Coins
	currentBlockHeight := view.Height()
	reserveSequence := tx.ReserveSequence
	paymentSequence := tx.PaymentSequence

	// Note: No need to check whether the source account has enough reserved fund to cover the
	//       transaction. If the source account does not have sufficient reserved fund,
	//       the source account will be slashed by the process() function
	err := sourceAccount.CheckTransferReservedFund(targetAccount, transferAmount, paymentSequence, currentBlockHeight, reserveSequence)
	if err != nil {
		return result.Error(err.Error()).WithErrorCode(result.CodeCheckTransferReservedFundFailed)
	}

	return result.OK
}

func (exec *ServicePaymentTxExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.ServicePaymentTx)

	sourceAddress := tx.Source.Address
	targetAddress := tx.Target.Address

	sourceAccount, res := getInput(view, tx.Source)
	if res.IsError() {
		return common.Hash{}, res
	}

	targetAccount, res := getOrMakeInput(view, tx.Target)
	if res.IsError() {
		return common.Hash{}, res
	}

	resourceID := tx.ResourceID
	splitRule := view.GetSplitRule(resourceID)

	fullTransferAmount := tx.Source.Coins
	splitSuccess, addrCoinsMap := exec.splitPayment(view, splitRule, resourceID, targetAddress, fullTransferAmount)
	if !splitSuccess {
		return common.Hash{}, result.Error("Failed to split payment")
	}

	accCoinsMap := map[*types.Account]types.Coins{}
	for addr, coins := range addrCoinsMap {
		var account *types.Account
		if addr == targetAddress {
			account = targetAccount
		} else if addr == sourceAddress {
			account = sourceAccount
		} else {
			account = getOrMakeAccount(view, addr)
		}
		accCoinsMap[account] = coins
	}

	currentBlockHeight := view.Height()
	reserveSequence := tx.ReserveSequence
	shouldSlash, _ := sourceAccount.TransferReservedFund(accCoinsMap, currentBlockHeight, reserveSequence, tx)
	if shouldSlash {
		//view.AddSlashIntent(slashIntent)
	}
	if !chargeFee(targetAccount, tx.Fee) {
		// should charge after transfer the fund, so an empty address has some fund to pay the tx fee
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	view.SetAccount(sourceAddress, sourceAccount)
	view.SetAccount(targetAddress, targetAccount)
	for account := range accCoinsMap {
		view.SetAccount(account.Address, account)
	}

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *ServicePaymentTxExecutor) splitPayment(view *st.StoreView, splitRule *types.SplitRule, resourceID string,
	targetAddress common.Address, fullAmount types.Coins) (bool, map[common.Address]types.Coins) {
	addressCoinsMap := map[common.Address]types.Coins{}

	// no splitRule associated with the resourceID, full payment goes to the target account
	if splitRule == nil {
		addressCoinsMap[targetAddress] = fullAmount
		return true, addressCoinsMap
	}

	// the splitRule has expired, full payment goes to the target account. also delete the splitRule
	if exec.state.Height() > splitRule.EndBlockHeight {
		addressCoinsMap[targetAddress] = fullAmount
		view.DeleteSplitRule(resourceID)
		return true, addressCoinsMap
	}

	// the splitRule is valid, split the payment among the participated addresses
	remainingAmount := fullAmount
	for _, split := range splitRule.Splits {
		splitAddress := split.Address
		percentage := split.Percentage
		if percentage > 100 || percentage < 0 {
			continue
		}

		splitAmount := fullAmount.CalculatePercentage(percentage)
		if _, exists := addressCoinsMap[splitAddress]; exists {
			addressCoinsMap[splitAddress] = splitAmount.Plus(addressCoinsMap[splitAddress])
		} else {
			addressCoinsMap[splitAddress] = splitAmount
		}
		remainingAmount = remainingAmount.Minus(splitAmount)
	}

	if !remainingAmount.IsNonnegative() { // so that the sum of percentage cannot be > 100
		return false, addressCoinsMap
	}

	if _, exists := addressCoinsMap[targetAddress]; exists { // the targetAddress could be included in the splitRule.Splits list
		addressCoinsMap[targetAddress] = remainingAmount.Plus(addressCoinsMap[targetAddress])
	} else {
		addressCoinsMap[targetAddress] = remainingAmount
	}

	return true, addressCoinsMap
}

func (exec *ServicePaymentTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.ServicePaymentTx)
	return &core.TxInfo{
		Address:           tx.Target.Address,
		Sequence:          tx.Target.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *ServicePaymentTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.ServicePaymentTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
