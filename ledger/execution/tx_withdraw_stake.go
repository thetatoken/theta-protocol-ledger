package execution

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
)

var _ TxExecutor = (*WithdrawStakeExecutor)(nil)

// ------------------------------- WithdrawStake Transaction -----------------------------------

// WithdrawStakeExecutor implements the TxExecutor interface
type WithdrawStakeExecutor struct {
	valMgr core.ValidatorManager
}

// NewWithdrawStakeExecutor creates a new instance of WithdrawStakeExecutor
func NewWithdrawStakeExecutor(valMgr core.ValidatorManager) *WithdrawStakeExecutor {
	return &WithdrawStakeExecutor{
		valMgr: valMgr,
	}
}

func (exec *WithdrawStakeExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	tx := transaction.(*types.WithdrawStakeTx)

	res := tx.Source.ValidateBasic()
	if res.IsError() {
		return res
	}

	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return result.Error("Failed to get the source account: %v", tx.Source.Address)
	}

	signBytes := tx.SignBytes(chainID)
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source)
	if res.IsError() {
		logger.Infof(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
		return res
	}

	if !sanityCheckForFee(tx.Fee) {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v GammaWei",
			types.MinimumTransactionFeeGammaWei).WithErrorCode(result.CodeInvalidFee)
	}

	if !(tx.Purpose == core.StakeForValidator || tx.Purpose == core.StakeForGuardian) {
		return result.Error("Invalid stake purpose!").
			WithErrorCode(result.CodeInvalidStakePurpose)
	}

	minimalBalance := tx.Fee
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("WithdrawStake: Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("WithdrawStake: Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance)
	}

	return result.OK
}

func (exec *WithdrawStakeExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.WithdrawStakeTx)

	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return common.Hash{}, result.Error("Failed to get the source account")
	}

	if !chargeFee(sourceAccount, tx.Fee) {
		return common.Hash{}, result.Error("Failed to charge transaction fee")
	}

	sourceAddress := tx.Source.Address
	holderAddress := tx.Holder.Address

	if tx.Purpose == core.StakeForValidator {
		vcp := view.GetValidatorCandidatePool()
		withdrawnAmount, err := vcp.WithdrawStake(sourceAddress, holderAddress)
		if err != nil {
			return common.Hash{}, result.Error("Failed to withdraw stake, err: %v", err)
		}
		withdrawnStake := types.NewCoins(0, 0)
		withdrawnStake.ThetaWei = withdrawnAmount
		sourceAccount.Balance = sourceAccount.Balance.Plus(withdrawnStake)
		view.UpdateValidatorCandidatePool(vcp)

		// TODO: acknowledge the consensus engine about the potential validator set change

	} else if tx.Purpose == core.StakeForGuardian {
		return common.Hash{}, result.Error("Withdraw stake for guardian not supported yet")
	} else {
		return common.Hash{}, result.Error("Invalid staking purpose").WithErrorCode(result.CodeInvalidStakePurpose)
	}

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *WithdrawStakeExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.WithdrawStakeTx)
	return &core.TxInfo{
		Address:           tx.Source.Address,
		Sequence:          tx.Source.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *WithdrawStakeExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.WithdrawStakeTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(types.GasWidthdrawStakeTx)
	effectiveGasPrice := new(big.Int).Div(fee.GammaWei, gas)
	return effectiveGasPrice
}
