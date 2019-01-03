package execution

import (
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
	valMgr core.ValidatorManager
}

// NewDepositStakeExecutor creates a new instance of DepositStakeExecutor
func NewDepositStakeExecutor(valMgr core.ValidatorManager) *DepositStakeExecutor {
	return &DepositStakeExecutor{
		valMgr: valMgr,
	}
}

// TODO: implementation
func (exec *DepositStakeExecutor) sanityCheck(chainID string, view *st.StoreView, transaction types.Tx) result.Result {
	//tx := transaction.(*types.DepositStakeTx)

	// 1. Minimum stake deposit requirement to avoid spamming

	return result.OK
}

// TODO: implementation
func (exec *DepositStakeExecutor) process(chainID string, view *st.StoreView, transaction types.Tx) (common.Hash, result.Result) {
	//tx := transaction.(*types.DepositStakeTx)
	return common.Hash{}, result.OK
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
	effectiveGasPrice := new(big.Int).Div(fee.GammaWei, gas)
	return effectiveGasPrice
}
