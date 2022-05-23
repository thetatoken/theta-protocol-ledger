package execution

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/state"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
)

var _ TxExecutor = (*WithdrawStakeExecutor)(nil)

// ------------------------------- WithdrawStake Transaction -----------------------------------

// WithdrawStakeExecutor implements the TxExecutor interface
type WithdrawStakeExecutor struct {
	state *st.LedgerState
}

// NewWithdrawStakeExecutor creates a new instance of WithdrawStakeExecutor
func NewWithdrawStakeExecutor(state *st.LedgerState) *WithdrawStakeExecutor {
	return &WithdrawStakeExecutor{
		state: state,
	}
}

func (exec *WithdrawStakeExecutor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result {
	blockHeight := view.Height() + 1 // the view points to the parent of the current block
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
	res = validateInputAdvanced(sourceAccount, signBytes, tx.Source, blockHeight)
	if res.IsError() {
		logger.Debugf(fmt.Sprintf("validateSourceAdvanced failed on %v: %v", tx.Source.Address.Hex(), res))
		return res
	}

	if minTxFee, success := sanityCheckForFee(tx.Fee, blockHeight); !success {
		return result.Error("Insufficient fee. Transaction fee needs to be at least %v TFuelWei",
			minTxFee).WithErrorCode(result.CodeInvalidFee)
	}

	if !(tx.Purpose == core.StakeForValidator || tx.Purpose == core.StakeForGuardian || tx.Purpose == core.StakeForEliteEdgeNode) {
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

// NOTE: WithdrawStakeExecutor.process() does NOT return the stake to the source. Instead, it updates
//       the ReturnHeight of the withdrawn stake. The stake will be returned to the source when
//       the block height reaches the ReturnHeigth
func (exec *WithdrawStakeExecutor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result) {
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
		currentHeight := exec.state.Height()
		err := vcp.WithdrawStake(sourceAddress, holderAddress, currentHeight)
		if err != nil {
			return common.Hash{}, result.Error("Failed to withdraw stake, err: %v", err)
		}
		view.UpdateValidatorCandidatePool(vcp)
	} else if tx.Purpose == core.StakeForGuardian {
		gcp := view.GetGuardianCandidatePool()
		currentHeight := exec.state.Height()
		err := gcp.WithdrawStake(sourceAddress, holderAddress, currentHeight)
		if err != nil {
			return common.Hash{}, result.Error("Failed to withdraw stake, err: %v", err)
		}
		view.UpdateGuardianCandidatePool(gcp)
	} else if tx.Purpose == core.StakeForEliteEdgeNode {
		eenp := state.NewEliteEdgeNodePool(view, false)
		currentHeight := exec.state.Height()
		withdrawnStake, err := eenp.WithdrawStake(sourceAddress, holderAddress, currentHeight)
		if err != nil || withdrawnStake == nil {
			return common.Hash{}, result.Error("Failed to withdraw stake, err: %v", err)
		}
		updateEliteEdgeNodeStakeReturns(view, holderAddress, *withdrawnStake)
	} else {
		return common.Hash{}, result.Error("Invalid staking purpose").WithErrorCode(result.CodeInvalidStakePurpose)
	}

	// Only update stake transaction height list for validator stake tx.
	if tx.Purpose == core.StakeForValidator {
		hl := view.GetStakeTransactionHeightList()
		if hl == nil {
			hl = &types.HeightList{}
		}
		blockHeight := view.Height() + 1 // the view points to the parent of the current block
		hl.Append(blockHeight)
		view.UpdateStakeTransactionHeightList(hl)
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
	gas := new(big.Int).SetUint64(getRegularTxGas(exec.state))
	effectiveGasPrice := new(big.Int).Div(fee.TFuelWei, gas)
	return effectiveGasPrice
}

func updateEliteEdgeNodeStakeReturns(view *st.StoreView, eenAddress common.Address, withdrawnStake core.Stake) {
	returnHeight := withdrawnStake.ReturnHeight
	stakesToBeReturned := view.GetEliteEdgeNodeStakeReturns(returnHeight)
	stakesToBeReturned = append(stakesToBeReturned, state.StakeWithHolder{
		Holder: eenAddress,
		Stake:  withdrawnStake,
	})
	view.SetEliteEdgeNodeStakeReturns(returnHeight, stakesToBeReturned)
}
