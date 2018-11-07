package execution

import (
	"fmt"
	"math/big"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/vm"
)

var _ TxExecutor = (*SmartContractTxExecutor)(nil)

// ------------------------------- SmartContractTx Transaction -----------------------------------

// SmartContractTxExecutor implements the TxExecutor interface
type SmartContractTxExecutor struct {
}

// NewSmartContractTxExecutor creates a new instance of SmartContractTxExecutor
func NewSmartContractTxExecutor() *SmartContractTxExecutor {
	return &SmartContractTxExecutor{}
}

func (exec *SmartContractTxExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	tx := transaction.(*types.SmartContractTx)

	// Validate from, basic
	res := tx.From.ValidateBasic()
	if res.IsError() {
		return res
	}

	// Get input account
	fromAccount, success := getInput(view, tx.From)
	if success.IsError() {
		return result.Error("Failed to get the from account")
	}

	// Validate input, advanced
	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(fromAccount, signBytes, tx.From)
	if res.IsError() {
		log.Infof(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.From.Address.Hex(), res))
		return res
	}

	coins := tx.From.Coins.NoNil()
	if !coins.IsNonnegative() {
		return result.Error("Invalid value to transfer").
			WithErrorCode(result.CodeInvalidValueToTransfer)
	}

	if !sanityCheckForGasPrice(tx.GasPrice) {
		return result.Error("Invalid gas price").
			WithErrorCode(result.CodeInvalidGasPrice)
	}

	zero := big.NewInt(0)
	feeLimit := new(big.Int).Mul(tx.GasPrice, new(big.Int).SetUint64(tx.GasLimit))
	if feeLimit.BitLen() > 255 || feeLimit.Cmp(zero) < 0 {
		// There is no explicit upper limit for big.Int. Just be conservative
		// here to prevent potential overflow attack
		return result.Error("Fee limit too high").
			WithErrorCode(result.CodeFeeLimitTooHigh)
	}

	value := coins.GammaWei // NoNil() already guarantees value is NOT nil
	minimalBalance := types.Coins{
		ThetaWei: zero,
		GammaWei: feeLimit.Add(feeLimit, value),
	}
	if !fromAccount.Balance.IsGTE(minimalBalance) {
		log.Infof(fmt.Sprintf("Source did not have enough balance %v", tx.From.Address.Hex()))
		return result.Error("Source balance is %v, but required minimal balance is %v",
			fromAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientFund)
	}

	return result.OK
}

func (exec *SmartContractTxExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.SmartContractTx)

	// Note: for contract deployment, vm.Execute() might transfer coins from the fromAccount to the
	//       deployed smart contract. Thus, we should call vm.Execute() before calling getInput().
	//       Otherwise, the fromAccount returned by getInput() will have incorrect balance.
	_, _, gasUsed, _ := vm.Execute(tx, view)

	fromAddress := tx.From.Address
	fromAccount, success := getInput(view, tx.From)
	if success.IsError() {
		return common.Hash{}, result.Error("Failed to get the from account")
	}

	feeAmount := new(big.Int).Mul(tx.GasPrice, new(big.Int).SetUint64(gasUsed))
	fee := types.Coins{
		ThetaWei: big.NewInt(int64(0)),
		GammaWei: feeAmount,
	}
	if !chargeFee(fromAccount, fee) {
		return common.Hash{}, result.Error("failed to charge transaction fee")
	}

	fromAccount.Sequence++
	view.SetAccount(fromAddress, fromAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}
