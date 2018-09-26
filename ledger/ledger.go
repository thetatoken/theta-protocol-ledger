package ledger

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	st "github.com/thetatoken/ukulele/ledger/state"
)

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	state     *st.LedgerState
	consensus core.ConsensusEngine

	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(consensus core.ConsensusEngine) *Ledger {
	return nil // TODO: proper implementation..
}

func (ledger *Ledger) BeginBlock(blockHash common.Bytes, header core.BlockHeader) bool {
	return false
}

func (ledger *Ledger) CheckTx(txBytes common.Bytes) bool {
	return false
}

func (ledger *Ledger) DeliverTx(txBytes common.Bytes) bool {
	return false
}

func (ledger *Ledger) EndBlock(height uint64) bool {
	return false
}

func (ledger *Ledger) Commit() bool {
	return false
}

func (ledger *Ledger) Query() {

}
