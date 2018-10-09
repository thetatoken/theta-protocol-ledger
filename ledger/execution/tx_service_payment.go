package execution

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
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

func (exec *ServicePaymentTxExecutor) sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result {
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
	if !sourceAccount.PubKey.VerifySignature(sourceSignBytes, tx.Source.Signature) {
		errMsg := fmt.Sprintf("sanityCheckForServicePaymentTx failed on source signature, addr: %v", sourceAddress.Hex())
		log.Infof(errMsg)
		return result.Error(errMsg)
	}

	// Verify target
	if targetAccount.Sequence+1 != tx.Target.Sequence {
		return result.Error("Got %v, expected %v. (acc.seq=%v)",
			tx.Target.Sequence, targetAccount.Sequence+1, targetAccount.Sequence)
	}

	targetSignBytes := tx.TargetSignBytes(chainID)
	if !targetAccount.PubKey.VerifySignature(targetSignBytes, tx.Target.Signature) {
		errMsg := fmt.Sprintf("sanityCheckForServicePaymentTx failed on target signature, addr: %v", targetAddress.Hex())
		log.Infof(errMsg)
		return result.Error(errMsg)
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("invalid fee")
	}

	transferAmount := tx.Source.Coins
	currentBlockHeight := exec.state.Height()
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

func (exec *ServicePaymentTxExecutor) process(chainID string, view types.ViewDataAccessor, transaction types.Tx) (common.Hash, result.Result) {
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
	splitContract := exec.state.GetSplitContract(resourceID)

	fullTransferAmount := tx.Source.Coins
	splitSuccess, coinsMap, accountAddressMap := exec.splitPayment(view, splitContract, resourceID, targetAddress, targetAccount, fullTransferAmount)
	if !splitSuccess {
		return common.Hash{}, result.Error("Failed to split payment")
	}

	currentBlockHeight := exec.state.Height()
	reserveSequence := tx.ReserveSequence
	shouldSlash, slashIntent := sourceAccount.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, tx)
	if shouldSlash {
		exec.state.AddSlashIntent(slashIntent)
	}
	if !chargeFee(targetAccount, tx.Fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}
	targetAccount.Sequence++ // targetAccount broadcasted the transaction

	view.SetAccount(sourceAddress, sourceAccount)
	for account := range coinsMap {
		address, exists := accountAddressMap[account]
		if !exists {
			panic(fmt.Sprintf("Cannot find address for account: %v", account))
		}
		view.SetAccount(address, account)
	}

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *ServicePaymentTxExecutor) splitPayment(view types.ViewDataAccessor, splitContract *types.SplitContract, resourceID common.Bytes,
	targetAddress common.Address, targetAccount *types.Account, fullAmount types.Coins) (bool, map[*types.Account]types.Coins, map[*types.Account](common.Address)) {
	coinsMap := map[*types.Account]types.Coins{}
	accountAddressMap := map[*types.Account](common.Address){}

	// no splitContract associated with the resourceID, full payment goes to the target account
	if splitContract == nil {
		coinsMap[targetAccount] = fullAmount
		accountAddressMap[targetAccount] = targetAddress
		return true, coinsMap, accountAddressMap
	}

	// the splitContract has expired, full payment goes to the target account. also delete the splitContract
	if exec.state.Height() > splitContract.EndBlockHeight {
		coinsMap[targetAccount] = fullAmount
		exec.state.DeleteSplitContract(resourceID)
		accountAddressMap[targetAccount] = targetAddress
		return true, coinsMap, accountAddressMap
	}

	// the splitContract is valid, split the payment among the participated addresses
	remainingAmount := fullAmount
	for _, split := range splitContract.Splits {
		splitAddress := split.Address
		splitAccount := getOrMakeAccount(view, splitAddress)
		percentage := split.Percentage
		if percentage > 100 || percentage < 0 {
			continue
		}

		splitAmount := fullAmount.CalculatePercentage(percentage)
		coinsMap[splitAccount] = splitAmount
		accountAddressMap[splitAccount] = splitAddress
		remainingAmount = remainingAmount.Minus(splitAmount)
	}

	if !remainingAmount.IsNonnegative() {
		return false, coinsMap, accountAddressMap
	}
	coinsMap[targetAccount] = remainingAmount
	accountAddressMap[targetAccount] = targetAddress

	return true, coinsMap, accountAddressMap
}
