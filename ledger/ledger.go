package ledger

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types/result"
	nd "github.com/thetatoken/ukulele/node"
)

var _ core.Ledger = (*Ledger)(nil)

//
// Ledger implements the core.Ledger interface
//
type Ledger struct {
	state *st.LedgerState
	node  *nd.Node

	executor *exec.Executor
}

// NewLedger creates an instance of Ledger
func NewLedger(node *nd.Node) *Ledger {
	return nil // TODO: proper implementation..
}

// SetNode sets the node instance
func (ledger *Ledger) SetNode(node *nd.Node) {
	ledger.node = node
	ledger.executor.SetNode(node)
}

func (ledger *Ledger) BeginBlock(blockHash common.Bytes, header core.BlockHeader) result.Result {
	return result.OK
}

func (ledger *Ledger) CheckTx(txBytes common.Bytes) result.Result {
	return result.OK
}

func (ledger *Ledger) DeliverTx(txBytes common.Bytes) result.Result {
	return result.OK
}

func (ledger *Ledger) EndBlock(height uint64) result.Result {
	return result.OK
}

func (ledger *Ledger) Commit() result.Result {
	return result.OK
}

func (ledger *Ledger) Query() {

}
