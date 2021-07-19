package state

import (
	"fmt"
	"log"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/store/database"
)

//
// ------------------------- State -------------------------
//

type Tagger interface {
	Tag(height uint64, root common.Hash)
}

type LedgerState struct {
	chainID  string
	db       database.Database
	dbTagger Tagger

	parentBlock *core.Block

	finalized *StoreView // for checking the latest finalized state
	delivered *StoreView // for actually applying the transactions
	checked   *StoreView // for block proposal check
	screened  *StoreView // for mempool screening
}

// NewLedgerState creates a new Leger State with given store.
// NOTE: before using the LedgerState, we need to call LedgerState.ResetState() to set
//       the proper height and stateRootHash
func NewLedgerState(chainID string, db database.Database, tagger Tagger) *LedgerState {
	s := &LedgerState{
		chainID:  chainID,
		db:       db,
		dbTagger: tagger,
	}
	//s.ResetState(uint64(0), common.Hash{})
	s.ResetState(&core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    uint64(0),
			StateHash: common.Hash{},
		},
	})
	s.Finalize(uint64(0), common.Hash{})
	return s
}

// ResetState resets the height and state root of its storeviews, and clear the in-memory states
//func (s *LedgerState) ResetState(height uint64, stateRootHash common.Hash) result.Result
func (s *LedgerState) ResetState(block *core.Block) result.Result {
	s.parentBlock = block

	height := block.Height
	stateRootHash := block.StateHash
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

// ParentBlock returns the pointer to the parent block for the current view
func (s *LedgerState) ParentBlock() *core.Block {
	return s.parentBlock
}

// Height returns the block height corresponding to the ledger state
func (s *LedgerState) Height() uint64 {
	return s.delivered.Height()
}

// Delivered returns a view of current state that contains both committed and delivered
// transactions.
func (s *LedgerState) Delivered() *StoreView {
	return s.delivered
}

// Checked creates a fresh clone of delivered view to be used for checking transactions.
func (s *LedgerState) Checked() *StoreView {
	return s.checked
}

// Screened creates a fresh clone of delivered view to be used for checking transactions.
func (s *LedgerState) Screened() *StoreView {
	return s.screened
}

// Finalized creates a fresh clone of delivered view to be used for checking transactions.
func (s *LedgerState) Finalized() *StoreView {
	return s.finalized
}

// Commit stores the current delivered view as committed, starts new delivered/checked state and
// returns the hash for the commit.
func (s *LedgerState) Commit() common.Hash {
	hash := s.delivered.Save()
	s.delivered.IncrementHeight()
	s.dbTagger.Tag(s.delivered.height, hash)

	var err error
	s.checked, err = s.delivered.Copy()
	if err != nil {
		log.Panicf("Commit: failed to copy to the checked view: %v", err)
	}
	s.screened, err = s.delivered.Copy()
	if err != nil {
		log.Panicf("Commit: failed to copy to the screened view: %v", err)
	}
	return hash
}
