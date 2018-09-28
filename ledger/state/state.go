package state

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/types"
)

//
// ------------------------- State -------------------------
//

type LedgerState struct {
	chainID string

	coinbaseTransactinProcessed bool
	slashIntents                []types.SlashIntent
	validatorsDiff              []*core.Validator

	delivered *StoreView
	checked   *StoreView
}

// NewLedgerState creates a new Leger State with givn store.
func NewLedgerState(sv *StoreView) *LedgerState {
	copiedView, err := sv.Copy()
	if err != nil {
		panic(fmt.Sprintf("Failed to create ledger state: %v", err))
	}
	return &LedgerState{
		checked:   copiedView,
		delivered: sv,
	}
}

// SetChainID sets chain ID.
func (s *LedgerState) SetChainID(chainID string) {
	s.chainID = chainID
	s.delivered.Set(common.Bytes("chain_id"), common.Bytes(chainID))
}

// GetChainID gets chain ID.
func (s *LedgerState) GetChainID() string {
	if s.chainID != "" {
		return s.chainID
	}
	s.chainID = string(s.delivered.Get(common.Bytes("chain_id")))
	return s.chainID
}

// AddSlashIntent adds slashIntent
func (s *LedgerState) AddSlashIntent(slashIntent types.SlashIntent) {
	s.slashIntents = append(s.slashIntents, slashIntent)
}

// GetSlashIntents retrieves all the slashIntents
func (s *LedgerState) GetSlashIntents() []types.SlashIntent {
	return s.slashIntents
}

// ClearSlashIntents clears all the slashIntents
func (s *LedgerState) ClearSlashIntents() {
	s.slashIntents = []types.SlashIntent{}
}

// CoinbaseTransactinProcessed returns whether the coinbase transaction for the current block has been processed
func (s *LedgerState) CoinbaseTransactinProcessed() bool {
	return s.coinbaseTransactinProcessed
}

// SetCoinbaseTransactionProcessed sets whether the coinbase transaction for the current block has been processed
func (s *LedgerState) SetCoinbaseTransactionProcessed(processed bool) {
	s.coinbaseTransactinProcessed = processed
}

// GetValidatorDiff retrives validator diff
func (s *LedgerState) GetAndClearValidatorDiff() []*core.Validator {
	res := s.validatorsDiff
	s.validatorsDiff = nil
	return res
}

// SetValidatorDiff set validator diff
func (s *LedgerState) SetValidatorDiff(diff []*core.Validator) {
	s.validatorsDiff = diff
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

// Commit stores the current delivered view as committed, starts new delivered/checked state and
// returns the hash for the commit.
func (s *LedgerState) Commit() common.Hash {
	hash := s.delivered.Save()
	copiedView, err := s.delivered.Copy()
	if err != nil {
		panic(fmt.Errorf("Failed to copy the delivered store view: %v", err))
	}
	s.checked = copiedView
	return hash
}

// GetAccount implements the ViewDataAccessor interface
func (s *LedgerState) GetAccount(addr common.Address) *types.Account {
	// return types.GetAccount(s.Delivered(), addr)
	return s.Delivered().GetAccount(addr)
}

// SetAccount implements the ViewDataAccessor interface
func (s *LedgerState) SetAccount(addr common.Address, acc *types.Account) {
	s.Delivered().SetAccount(addr, acc)
}

// SplitContractExists checks if a split contract associated with the given resourceId already exists
func (s *LedgerState) SplitContractExists(resourceId common.Bytes) bool {
	exists := (s.Delivered().GetSplitContract(resourceId) != nil)
	return exists
}

// GetSplitContract implements the ViewDataAccessor interface
func (s *LedgerState) GetSplitContract(resourceId common.Bytes) *types.SplitContract {
	return s.Delivered().GetSplitContract(resourceId)
}

// SetSplitContract implements the ViewDataAccessor interface
func (s *LedgerState) SetSplitContract(resourceId common.Bytes, splitContract *types.SplitContract) {
	s.Delivered().SetSplitContract(resourceId, splitContract)
}

// AddSplitContract adds a split contract
func (s *LedgerState) AddSplitContract(splitContract *types.SplitContract) bool {
	if s.SplitContractExists(splitContract.ResourceId) {
		return false // Each resourceId can have at most one corresponding split contract
	}

	s.SetSplitContract(splitContract.ResourceId, splitContract)
	return true
}

// UpdateSplitContract updates a split contract
func (s *LedgerState) UpdateSplitContract(splitContract *types.SplitContract) bool {
	if !s.SplitContractExists(splitContract.ResourceId) {
		return false
	}

	s.SetSplitContract(splitContract.ResourceId, splitContract)
	return true
}

// DeleteSplitContract implements the ViewDataAccessor interface
func (s *LedgerState) DeleteSplitContract(resourceId common.Bytes) bool {
	return s.Delivered().DeleteSplitContract(resourceId)
}

// DeleteExpiredSplitContracts implements the ViewDataAccessor interface
func (s *LedgerState) DeleteExpiredSplitContracts(currentBlockHeight uint64) bool {
	return s.Delivered().DeleteExpiredSplitContracts(currentBlockHeight)
}
