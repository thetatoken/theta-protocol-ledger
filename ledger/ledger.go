package ledger

import (
	exec "github.com/thetatoken/ukulele/ledger/execution"
	st "github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	nd "github.com/thetatoken/ukulele/node"
)

type Ledger struct {
	state *st.LedgerState
	node  *nd.Node

	executor *exec.Executor
}

// SetState sets the ledger state
func (ledger *Ledger) SetState(state *st.LedgerState) {
	ledger.state = state
}

// GetState returns the state of the ledger
func (ledger *Ledger) GetState() *st.LedgerState {
	return ledger.state
}

// SetNode sets the node instance
func (ledger *Ledger) SetNode(n *nd.Node) {
	ledger.node = n
}

// GetNode returns the node instance
func (ledger *Ledger) GetNode() *nd.Node {
	return ledger.node
}

// SplitContractExists checks if a split contract associated with the given resourceId already exists
func (ledger *Ledger) SplitContractExists(resourceId []byte) bool {
	exists := (ledger.state.GetSplitContract(resourceId) != nil)
	return exists
}

// GetSplitContract returns a split contract associated with the given resourceId if exists, and nil otherwise
func (ledger *Ledger) GetSplitContract(resourceId []byte) *types.SplitContract {
	splitContract := ledger.state.GetSplitContract(resourceId)
	return splitContract
}

// AddSplitContract adds a split contract
func (ledger *Ledger) AddSplitContract(splitContract *types.SplitContract) bool {
	if ledger.SplitContractExists(splitContract.ResourceId) {
		return false // Each resourceId can have at most one corresponding split contract
	}

	ledger.state.SetSplitContract(splitContract.ResourceId, splitContract)
	return true
}

// UpdateSplitContract updates a split contract
func (ledger *Ledger) UpdateSplitContract(splitContract *types.SplitContract) bool {
	if !ledger.SplitContractExists(splitContract.ResourceId) {
		return false
	}

	ledger.state.SetSplitContract(splitContract.ResourceId, splitContract)
	return true
}

// DeleteSplitContract deletes the split contract associated with the given resourceId
func (ledger *Ledger) DeleteSplitContract(resourceId []byte) bool {
	_, deleted := ledger.state.DeleteSplitContract(resourceId)
	return deleted
}

// DeleteExpiredSplitContracts deletes split contracts that already expired
func (ledger *Ledger) DeleteExpiredSplitContracts(currentBlockHeight uint64) bool {
	deleted := ledger.state.DeleteExpiredSplitContracts(currentBlockHeight)
	return deleted
}
