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

var _ TxExecutor = (*DepositStakeExecutor)(nil)

// ------------------------------- DepositStake Transaction -----------------------------------

// DepositStakeExecutor implements the TxExecutor interface
type DepositStakeExecutor struct {
}

// NewDepositStakeExecutor creates a new instance of DepositStakeExecutor
func NewDepositStakeExecutor() *DepositStakeExecutor {
	return &DepositStakeExecutor{}
}

func (exec *DepositStakeExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	tx := transaction.(*types.DepositStakeTx)

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
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			types.MinimumTransactionFeeTFuelWei).WithErrorCode(result.CodeInvalidFee)
	}

	if !(tx.Purpose == core.StakeForValidator || tx.Purpose == core.StakeForGuardian) {
		return result.Error("Invalid stake purpose!").
			WithErrorCode(result.CodeInvalidStakePurpose)
	}

	stake := tx.Source.Coins.NoNil()
	if !stake.IsValid() || !stake.IsNonnegative() {
		return result.Error("Invalid stake for stake deposit!").
			WithErrorCode(result.CodeInvalidStake)
	}

	if stake.TFuelWei.Cmp(types.Zero) != 0 {
		return result.Error("TFuel has to be zero for stake deposit!").
			WithErrorCode(result.CodeInvalidStake)
	}

	// Minimum stake deposit requirement to avoid spamming
	if stake.ThetaWei.Cmp(core.MinValidatorStakeDeposit) < 0 {
		return result.Error("Insufficient amount of stake, at least %v ThetaWei is required for each deposit", core.MinValidatorStakeDeposit).
			WithErrorCode(result.CodeInsufficientStake)
	}

	minimalBalance := stake.Plus(tx.Fee)
	if !sourceAccount.Balance.IsGTE(minimalBalance) {
		logger.Infof(fmt.Sprintf("DepositStake: Source did not have enough balance %v", tx.Source.Address.Hex()))
		return result.Error("DepositStake: Source balance is %v, but required minimal balance is %v",
			sourceAccount.Balance, minimalBalance).WithErrorCode(result.CodeInsufficientStake)
	}

	return result.OK
}

func (exec *DepositStakeExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	tx := transaction.(*types.DepositStakeTx)

	sourceAccount, success := getInput(view, tx.Source)
	if success.IsError() {
		return common.Hash{}, result.Error("Failed to get the source account")
	}

	if !chargeFee(sourceAccount, tx.Fee) {
		return common.Hash{}, result.Error("Failed to charge transaction fee")
	}

	stake := tx.Source.Coins.NoNil()
	if !sourceAccount.Balance.IsGTE(stake) {
		return common.Hash{}, result.Error("Not enough balance to stake").WithErrorCode(result.CodeNotEnoughBalanceToStake)
	}

	sourceAddress := tx.Source.Address
	holderAddress := tx.Holder.Address

	if tx.Purpose == core.StakeForValidator {
		sourceAccount.Balance = sourceAccount.Balance.Minus(stake)
		stakeAmount := stake.ThetaWei
		vcp := view.GetValidatorCandidatePool()
		err := vcp.DepositStake(sourceAddress, holderAddress, stakeAmount)
		if err != nil {
			return common.Hash{}, result.Error("Failed to deposit stake, err: %v", err)
		}
		view.UpdateValidatorCandidatePool(vcp)
	} else if tx.Purpose == core.StakeForGuardian {
		return common.Hash{}, result.Error("Staking for guardian not supported yet")
	} else {
		return common.Hash{}, result.Error("Invalid staking purpose").WithErrorCode(result.CodeInvalidStakePurpose)
	}

	hl := view.GetStakeTransactionHeightList()
	if hl == nil {
		hl = &types.HeightList{}
	}
	hl.Append(view.Height())
	view.UpdateStakeTransactionHeightList(hl)

	sourceAccount.Sequence++
	view.SetAccount(sourceAddress, sourceAccount)

	txHash := types.TxID(chainID, tx)
	return txHash, result.OK
}

func (exec *DepositStakeExecutor) getTxInfo(transaction types.Tx) *core.TxInfo {
	tx := transaction.(*types.DepositStakeTx)
	return &core.TxInfo{
		Address:           tx.Source.Address,
		Sequence:          tx.Source.Sequence,
		EffectiveGasPrice: exec.calculateEffectiveGasPrice(transaction),
	}
}

func (exec *DepositStakeExecutor) calculateEffectiveGasPrice(transaction types.Tx) *big.Int {
	tx := transaction.(*types.DepositStakeTx)
	fee := tx.Fee
	gas := new(big.Int).SetUint64(types.GasDepositStakeTx)
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}
