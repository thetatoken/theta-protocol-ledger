package execution

import (
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	st "github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "ledger"})

//
// TxExecutor defines the interface of the transaction executors
//
type TxExecutor interface {
	sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) result.Result
	process(chainID string, view *st.StoreView, viewSel core.ViewSelector, transaction types.Tx) (common.Hash, result.Result)
	getTxInfo(transaction types.Tx) *core.TxInfo
}

//
// Executor executes the transactions
//
type Executor struct {
	db        database.Database
	chain     *blockchain.Chain
	state     *st.LedgerState
	consensus core.ConsensusEngine
	valMgr    core.ValidatorManager
	ledger    core.Ledger

	coinbaseTxExec *CoinbaseTxExecutor
	// slashTxExec          *SlashTxExecutor
	sendTxExec                    *SendTxExecutor
	reserveFundTxExec             *ReserveFundTxExecutor
	releaseFundTxExec             *ReleaseFundTxExecutor
	servicePaymentTxExec          *ServicePaymentTxExecutor
	splitRuleTxExec               *SplitRuleTxExecutor
	smartContractTxExec           *SmartContractTxExecutor
	depositStakeTxExec            *DepositStakeExecutor
	withdrawStakeTxExec           *WithdrawStakeExecutor
	stakeRewardDistributionTxExec *StakeRewardDistributionTxExecutor

	skipSanityCheck bool
}

// NewExecutor creates a new instance of Executor
func NewExecutor(db database.Database, chain *blockchain.Chain, state *st.LedgerState, consensus core.ConsensusEngine, valMgr core.ValidatorManager, ledger core.Ledger) *Executor {
	executor := &Executor{
		db:             db,
		chain:          chain,
		state:          state,
		consensus:      consensus,
		valMgr:         valMgr,
		coinbaseTxExec: NewCoinbaseTxExecutor(db, chain, state, consensus, valMgr),
		// slashTxExec:          NewSlashTxExecutor(consensus, valMgr),
		sendTxExec:                    NewSendTxExecutor(state),
		reserveFundTxExec:             NewReserveFundTxExecutor(state),
		releaseFundTxExec:             NewReleaseFundTxExecutor(state),
		servicePaymentTxExec:          NewServicePaymentTxExecutor(state),
		splitRuleTxExec:               NewSplitRuleTxExecutor(state),
		smartContractTxExec:           NewSmartContractTxExecutor(chain, state, ledger),
		depositStakeTxExec:            NewDepositStakeExecutor(state),
		withdrawStakeTxExec:           NewWithdrawStakeExecutor(state),
		stakeRewardDistributionTxExec: NewStakeRewardDistributionTxExecutor(state),
		skipSanityCheck:               false,
	}

	return executor
}

// SetSkipSanityCheck sets the flag for sanity check.
// Skip checks while replaying commmitted blocks.
func (exec *Executor) SetSkipSanityCheck(skip bool) {
	exec.skipSanityCheck = skip
}

// ExecuteTx executes the given transaction
func (exec *Executor) ExecuteTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.DeliveredView)
}

// CheckTx checks the validity of the given transaction
func (exec *Executor) CheckTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.CheckedView)
}

// ScreenTx checks the validity of the given transaction
func (exec *Executor) ScreenTx(tx types.Tx) (common.Hash, result.Result) {
	return exec.processTx(tx, core.ScreenedView)
}

// GetTxInfo extracts tx information used by mempool to sort Txs.
func (exec *Executor) GetTxInfo(tx types.Tx) (*core.TxInfo, result.Result) {
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor == nil {
		return nil, result.Error("Unknown tx type")
	}

	txInfo := txExecutor.getTxInfo(tx)
	return txInfo, result.OK
}

// processTx contains the main logic to process the transaction. If the tx is invalid, a TMSP error will be returned.
func (exec *Executor) processTx(tx types.Tx, viewSel core.ViewSelector) (common.Hash, result.Result) {
	chainID := exec.state.GetChainID()
	var view *st.StoreView
	switch viewSel {
	case core.DeliveredView:
		view = exec.state.Delivered()
	case core.CheckedView:
		view = exec.state.Checked()
	default:
		view = exec.state.Screened()
	}

	sanityCheckResult := exec.sanityCheck(chainID, view, viewSel, tx)
	if sanityCheckResult.IsError() {
		return common.Hash{}, sanityCheckResult
	}

	txHash, processResult := exec.process(chainID, view, viewSel, tx)
	return txHash, processResult
}

func (exec *Executor) sanityCheck(chainID string, view *st.StoreView, viewSel core.ViewSelector, tx types.Tx) result.Result {
	if exec.skipSanityCheck { // Skip checks, e.g. while replaying commmitted blocks.
		return result.OK
	}

	if !exec.isTxTypeSupported(view, tx) {
		return result.Error("tx type not supported yet")
	}

	var sanityCheckResult result.Result
	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		sanityCheckResult = txExecutor.sanityCheck(chainID, view, viewSel, tx)
	} else {
		sanityCheckResult = result.Error("Unknown tx type")
	}

	return sanityCheckResult
}

func (exec *Executor) process(chainID string, view *st.StoreView, viewSel core.ViewSelector, tx types.Tx) (common.Hash, result.Result) {
	var processResult result.Result
	var txHash common.Hash

	if !exec.isTxTypeSupported(view, tx) {
		return txHash, result.Error("tx type not supported yet")
	}

	txExecutor := exec.getTxExecutor(tx)
	if txExecutor != nil {
		txHash, processResult = txExecutor.process(chainID, view, viewSel, tx)
		if processResult.IsError() {
			logger.Warnf("Tx processing error: %v", processResult.Message)
		}
	} else {
		processResult = result.Error("Unknown tx type")
	}

	return txHash, processResult
}

func (exec *Executor) isTxTypeSupported(view *st.StoreView, tx types.Tx) bool {
	blockHeight := view.Height() + 1

	switch tx.(type) {
	case *types.SmartContractTx:
		if blockHeight < common.HeightEnableSmartContract {
			return false
		}
	case *types.StakeRewardDistributionTx:
		if blockHeight < common.HeightEnableTheta3 {
			return false
		}
	default:
		return true
	}

	return true
}

func (exec *Executor) getTxExecutor(tx types.Tx) TxExecutor {
	var txExecutor TxExecutor
	switch tx.(type) {
	case *types.CoinbaseTx:
		txExecutor = exec.coinbaseTxExec
	// case *types.SlashTx:
	// 	txExecutor = exec.slashTxExec
	case *types.SendTx:
		txExecutor = exec.sendTxExec
	case *types.ReserveFundTx:
		txExecutor = exec.reserveFundTxExec
	case *types.ReleaseFundTx:
		txExecutor = exec.releaseFundTxExec
	case *types.ServicePaymentTx:
		txExecutor = exec.servicePaymentTxExec
	case *types.SplitRuleTx:
		txExecutor = exec.splitRuleTxExec
	case *types.SmartContractTx:
		txExecutor = exec.smartContractTxExec
	case *types.DepositStakeTx:
		txExecutor = exec.depositStakeTxExec
	case *types.WithdrawStakeTx:
		txExecutor = exec.withdrawStakeTxExec
	case *types.DepositStakeTxV2:
		txExecutor = exec.depositStakeTxExec
	case *types.StakeRewardDistributionTx:
		txExecutor = exec.stakeRewardDistributionTxExec
	default:
		txExecutor = nil
	}
	return txExecutor
}
