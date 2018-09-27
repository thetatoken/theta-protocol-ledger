package ledger

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	mp "github.com/thetatoken/ukulele/mempool"
)

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	consensus core.ConsensusEngine
	mempool   *mp.Mempool

	state    *st.LedgerState
	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(consensus core.ConsensusEngine, mempool *mp.Mempool) *Ledger {
	return nil // TODO: proper implementation..
}

// CheckTx implements the core.Ledger interface
func (ledger *Ledger) CheckTx(rawTx common.Bytes) result.Result {
	var tx types.Tx
	tx, err := types.TxFromBytes(rawTx)
	if err != nil {
		return result.Error("Error decoding tx: %v", err)
	}

	if shouldSkipCheckTx(tx) {
		return result.Error("Unauthorized transaction, should skip")
	}

	_, res := ledger.executor.ExecuteTx(tx, true) // Sanity check only
	return res
}

func (ledger *Ledger) DeliverTxs(rawTxs *[]common.Bytes) result.Result {
	return result.OK
}

func (ledger *Ledger) Query() {

}

// CheckTx() should skip all the transactions that can only be initiated by the validators
// i.e., if a regular user submits a coinbaseTx or slashTx, it should be skipped so it will not
// get into the mempool
func shouldSkipCheckTx(tx types.Tx) bool {
	switch tx.(type) {
	case *types.CoinbaseTx:
		return true
	case *types.SlashTx:
		return true
	default:
		return false
	}
}
