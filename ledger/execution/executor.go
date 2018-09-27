package execution

import (
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
	nd "github.com/thetatoken/ukulele/node"
)

//
// TxExecutor defines the interface of the transaction executors
//
type TxExecutor interface {
	sanityCheck(chainID string, view types.ViewDataGetter, transaction types.Tx) result.Result
	process(chainID string, view types.ViewDataAccessor, transaction types.Tx) result.Result
}

//
// Executor executes the transactions
//
type Executor struct {
	state *st.LedgerState
	node  *nd.Node

	coinbaseTxExec        *CoinbaseTxExecutor
	slashTxExec           *SlashTxExecutor
	updateValidatorTxExec *UpdateValidatorsTxExecutor
	sendTxExec            *SendTxExecutor
	reserveFundTxExec     *ReserveFundTxExecutor
	releaseFundTxExec     *ReleaseFundTxExecutor
	servicePaymentTxExec  *ServicePaymentTxExecutor
	splitContractTxExec   *SplitContractTxExecutor
}

// NewExecutor creates a new instance of Executor
func NewExecutor(state *st.LedgerState, node *nd.Node) *Executor {
	executor := &Executor{
		state:                 state,
		node:                  node,
		coinbaseTxExec:        NewCoinbaseTxExecutor(state, node),
		slashTxExec:           NewSlashTxExecutor(node),
		updateValidatorTxExec: NewUpdateValidatorsTxExecutor(state),
		sendTxExec:            NewSendTxExecutor(),
		reserveFundTxExec:     NewReserveFundTxExecutor(),
		releaseFundTxExec:     NewReleaseFundTxExecutor(),
		servicePaymentTxExec:  NewServicePaymentTxExecutor(state),
		splitContractTxExec:   NewSplitContractTxExecutor(state),
	}

	return executor
}

// SetNode sets the node pointer of the executors
func (exec *Executor) SetNode(node *nd.Node) {
	exec.node = node
	exec.coinbaseTxExec.node = node
	exec.slashTxExec.node = node
}

// ExecuteTx contains the main logic for CheckTx and DeliverTx. If the tx is invalid, a TMSP error will be returned.
func (exec *Executor) ExecuteTx(tx types.Tx, isCheckTx bool) result.Result {
	chainID := exec.state.GetChainID()
	var view *st.StoreView
	if isCheckTx {
		view = exec.state.Checked()
	} else {
		view = exec.state.Delivered()
	}

	sanityCheckResult := exec.sanityCheck(chainID, view, tx)
	if sanityCheckResult.IsErr() || isCheckTx {
		return sanityCheckResult
	}

	processResult := exec.process(chainID, view, tx)
	return processResult
}

func (exec *Executor) sanityCheck(chainID string, view types.ViewDataGetter, tx types.Tx) result.Result {
	// Skip checks while replaying commmitted blocks.
	if exec.node == nil {
		return result.OK
	}

	var sanityCheckResult result.Result
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		sanityCheckResult = txExecutor.sanityCheck(chainID, view, tx)
	} else {
		sanityCheckResult = result.ErrBaseEncodingError.SetLog("Unknown tx type")
	}

	return sanityCheckResult
}

func (exec *Executor) process(chainID string, view types.ViewDataAccessor, tx types.Tx) result.Result {
	var processResult result.Result
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		processResult = txExecutor.process(chainID, view, tx)
	} else {
		processResult = result.ErrBaseEncodingError.SetLog("Unknown tx type")
	}

	return processResult
}

func (exec *Executor) getTxExecutor(tx types.Tx) TxExecutor {
	var txExecutor TxExecutor
	switch tx.(type) {
	case *types.CoinbaseTx:
		txExecutor = exec.coinbaseTxExec
	case *types.SlashTx:
		txExecutor = exec.slashTxExec
	case *types.SendTx:
		txExecutor = exec.sendTxExec
	case *types.ReserveFundTx:
		txExecutor = exec.reserveFundTxExec
	case *types.ReleaseFundTx:
		txExecutor = exec.releaseFundTxExec
	case *types.ServicePaymentTx:
		txExecutor = exec.servicePaymentTxExec
	case *types.SplitContractTx:
		txExecutor = exec.splitContractTxExec
	case *types.UpdateValidatorsTx:
		txExecutor = exec.updateValidatorTxExec
	default:
		txExecutor = nil
	}
	return txExecutor
}
