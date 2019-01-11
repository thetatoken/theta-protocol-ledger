package state

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/store/database"
)

//
// ------------------------- State -------------------------
//

type LedgerState struct {
	chainID string
	db      database.Database

	finalized *StoreView // for checking the latest finalized state
	delivered *StoreView // for actually applying the transactions
	checked   *StoreView // for block proposal check
	screened  *StoreView // for mempool screening
}

// NewLedgerState creates a new Leger State with given store.
// NOTE: before using the LedgerState, we need to call LedgerState.ResetState() to set
//       the proper height and stateRootHash
func NewLedgerState(chainID string, db database.Database) *LedgerState {
	s := &LedgerState{
		chainID: chainID,
		db:      db,
	}
	s.ResetState(uint64(0), common.Hash{})
	s.Finalize(uint64(0), common.Hash{})
	return s
}

// ResetState resets the height and state root of its storeviews, and clear the in-memory states
func (s *LedgerState) ResetState(height uint64, stateRootHash common.Hash) result.Result {
	storeview := NewStoreView(height, stateRootHash, s.db)
	if storeview == nil {
		return result.Error(fmt.Sprintf("Failed to set ledger state with state root hash: %v", stateRootHash))
	}
	s.delivered = storeview

	var err error
	s.checked, err = s.delivered.Copy()
	if err != nil {
		return result.Error(fmt.Sprintf("Failed to copy to the checked view: %v", err))
	}
	s.screened, err = s.delivered.Copy()
	if err != nil {
		return result.Error(fmt.Sprintf("Failed to copy to the screened view: %v", err))
	}

	return result.OK
}

// Finalize updates the finalized view.
func (s *LedgerState) Finalize(height uint64, stateRootHash common.Hash) result.Result {
	storeview := NewStoreView(height, stateRootHash, s.db)
	if storeview == nil {
		return result.Error(fmt.Sprintf("Failed to finalize ledger state with state root hash: %v", stateRootHash))
	}
	s.finalized = storeview
	return result.OK
}

// GetChainID gets chain ID.
func (s *LedgerState) GetChainID() string {
	if s.chainID != "" {
		return s.chainID
	}
	s.chainID = string(s.delivered.Get(ChainIDKey()))
	return s.chainID
}

// DB returns the database instance of the ledger state
func (s *LedgerState) DB() database.Database {
	return s.db
}

// Height returns the block height corresponding to the ledger state
func (s *LedgerState) Height() uint64 {
	return s.delivered.Height()
}

// Delivered returns a view of current state that contains both committed and delivered
// transcations.
func (s *LedgerState) Delivered() *StoreView {
	return s.delivered
}

// Checked creates a fresh clone of delivered view to be used for checking transcations.
func (s *LedgerState) Checked() *StoreView {
	return s.checked
}

// Screened creates a fresh clone of delivered view to be used for checking transcations.
func (s *LedgerState) Screened() *StoreView {
	return s.screened
}

// Finalized creates a fresh clone of delivered view to be used for checking transcations.
func (s *LedgerState) Finalized() *StoreView {
	return s.finalized
}

// Commit stores the current delivered view as committed, starts new delivered/checked state and
// returns the hash for the commit.
func (s *LedgerState) Commit() common.Hash {
	hash := s.delivered.Save()
	s.delivered.IncrementHeight()

	var err error
	s.checked, err = s.delivered.Copy()
	if err != nil {
		panic(fmt.Errorf("Commit: failed to copy to the checked view: %v", err))
	}
	s.screened, err = s.delivered.Copy()
	if err != nil {
		panic(fmt.Errorf("Commit: failed to copy to the screened view: %v", err))
	}
	return hash
}
