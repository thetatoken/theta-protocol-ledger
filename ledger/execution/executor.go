package execution

import (
	"github.com/thetatoken/ukulele/core"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/ledger/types/result"
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
	state     *st.LedgerState
	consensus core.ConsensusEngine

	coinbaseTxExec        *CoinbaseTxExecutor
	slashTxExec           *SlashTxExecutor
	updateValidatorTxExec *UpdateValidatorsTxExecutor
	sendTxExec            *SendTxExecutor
	reserveFundTxExec     *ReserveFundTxExecutor
	releaseFundTxExec     *ReleaseFundTxExecutor
	servicePaymentTxExec  *ServicePaymentTxExecutor
	splitContractTxExec   *SplitContractTxExecutor

	skipSanityCheck bool
}

// NewExecutor creates a new instance of Executor
func NewExecutor(state *st.LedgerState, consensus core.ConsensusEngine) *Executor {
	executor := &Executor{
		state:                 state,
		consensus:             consensus,
		coinbaseTxExec:        NewCoinbaseTxExecutor(state, consensus),
		slashTxExec:           NewSlashTxExecutor(consensus),
		updateValidatorTxExec: NewUpdateValidatorsTxExecutor(state),
		sendTxExec:            NewSendTxExecutor(),
		reserveFundTxExec:     NewReserveFundTxExecutor(),
		releaseFundTxExec:     NewReleaseFundTxExecutor(),
		servicePaymentTxExec:  NewServicePaymentTxExecutor(state),
		splitContractTxExec:   NewSplitContractTxExecutor(state),
		skipSanityCheck:       false,
	}

	return executor
}

// SetSkipSanityCheck sets the flag for sanity check.
// Skip checks while replaying commmitted blocks.
func (exec *Executor) SetSkipSanityCheck(skip bool) {
	exec.skipSanityCheck = skip
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
	if exec.skipSanityCheck {
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
