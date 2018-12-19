package execution

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/vm"
)

var _ TxExecutor = (*SmartContractTxExecutor)(nil)

// ------------------------------- SmartContractTx Transaction -----------------------------------

// SmartContractTxExecutor implements the TxExecutor interface
type SmartContractTxExecutor struct {
	state *st.LedgerState
}

// NewSmartContractTxExecutor creates a new instance of SmartContractTxExecutor
func NewSmartContractTxExecutor(state *st.LedgerState) *SmartContractTxExecutor {
	return &SmartContractTxExecutor{
		state: state,
	}
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
		logger.Infof(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.From.Address.Hex(), res))
		return res
	}

	coins := tx.From.Coins.NoNil()
	if !coins.IsNonnegative() {
		return result.Error("Invalid value to transfer").
			WithErrorCode(result.CodeInvalidValueToTransfer)
	}

	if !sanityCheckForGasPrice(tx.GasPrice) {
		return result.Error("Insufficient gas price. Gas price needs to be at least %v GammaWei", types.MinimumGasPrice).
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
		logger.Infof(fmt.Sprintf("Source did not have enough balance %v", tx.From.Address.Hex()))
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

	createContract := (tx.To.Address == common.Address{})
	if !createContract { // vm.create() increments the sequence of the from account
		fromAccount.Sequence++
	}
	view.SetAccount(fromAddress, fromAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *SmartContractTxExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.SmartContractTx)
	return &core.TxInfo{
		Address:           tx.From.Address,
		Sequence:          tx.From.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *SmartContractTxExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.SmartContractTx)
	return tx.GasPrice
}
