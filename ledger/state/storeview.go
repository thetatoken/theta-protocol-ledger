package state

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/treestore"
)

//
// ------------------------- StoreView -------------------------
//

type StoreView struct {
	height uint64 // block height
	store  *treestore.TreeStore
}

// NewStoreView creates an instance of the StoreView
func NewStoreView(height uint64, root common.Hash, db database.Database) *StoreView {
	store := treestore.NewTreeStore(root, db, false)
	if store == nil {
		return nil
	}
	sv := &StoreView{height, store}
	return sv
}

// Copy returns a copy of the StoreView
func (sv *StoreView) Copy() (*StoreView, error) {
	copiedStore, err := sv.store.Copy()
	if err != nil {
		return nil, err
	}
	copiedStoreView := &StoreView{
		sv.height,
		copiedStore,
	}
	return copiedStoreView, nil
}

// Hash returns the root hash of the tree store
func (sv *StoreView) Hash() common.Hash {
	return sv.store.Hash()
}

// Height returns the block height corresponding to the stored state
func (sv *StoreView) Height() uint64 {
	return sv.height
}

// IncrementHeight increments the block height by 1
func (sv *StoreView) IncrementHeight() {
	sv.height++
}

// Save saves the StoreView to the persistent storage, and return the root hash
func (sv *StoreView) Save() common.Hash {
	rootHash, err := sv.store.Commit(nil)
	if err != nil {
		panic(fmt.Sprintf("Failed to save the StoreView: %v", err))
	}
	sv.store.Trie.GetDB().Commit(rootHash, true)
	return rootHash
}

// Get returns the value corresponding the key
func (sv *StoreView) Get(key common.Bytes) common.Bytes {
	value := sv.store.Get(key)
	return value
}

// Set returns the value corresponding the key
func (sv *StoreView) Set(key common.Bytes, value common.Bytes) {
	sv.store.Set(key, value)
}

// GetAccount implements the ViewDataAccessor GetAccount() method.
func (sv *StoreView) GetAccount(addr common.Address) *types.Account {
	data := sv.Get(AccountKey(addr))
	if data == nil || len(data) == 0 {
		return nil
	}
	acc := &types.Account{}
	err := types.FromBytes(data, acc)
	if err != nil {
		panic(fmt.Sprintf("Error reading account %X error: %v",
			data, err.Error()))
	}
	return acc
}

// SetAccount implements the ViewDataAccessor SetAccount() method.
func (sv *StoreView) SetAccount(addr common.Address, acc *types.Account) {
	accBytes, err := types.ToBytes(acc)
	if err != nil {
		panic(fmt.Sprintf("Error writing account %v error: %v",
			acc, err.Error()))
	}
	sv.Set(AccountKey(addr), accBytes)
}

// GetSplitContract implements the ViewDataAccessor GetSplitContract() method
func (sv *StoreView) GetSplitContract(resourceID common.Bytes) *types.SplitContract {
	data := sv.Get(SplitContractKey(resourceID))
	if data == nil || len(data) == 0 {
		return nil
	}
	splitContract := &types.SplitContract{}
	err := types.FromBytes(data, splitContract)
	if err != nil {
		panic(fmt.Sprintf("Error reading splitContract %X error: %v",
			data, err.Error()))
	}
	return splitContract
}

// SetSplitContract implements the ViewDataAccessor SetSplitContract() method
func (sv *StoreView) SetSplitContract(resourceID common.Bytes, splitContract *types.SplitContract) {
	splitContractBytes, err := types.ToBytes(splitContract)
	if err != nil {
		panic(fmt.Sprintf("Error writing splitContract %v error: %v",
			splitContract, err.Error()))
	}
	sv.Set(SplitContractKey(resourceID), splitContractBytes)
}

// DeleteSplitContract implements the ViewDataAccessor DeleteSplitContract() method
func (sv *StoreView) DeleteSplitContract(resourceID common.Bytes) bool {
	key := SplitContractKey(resourceID)
	deleted := sv.store.Delete(key)
	return deleted
}

// DeleteExpiredSplitContracts implements the ViewDataAccessor DeleteExpiredSplitContracts() method
func (sv *StoreView) DeleteExpiredSplitContracts(currentBlockHeight uint64) bool {
	prefix := SplitContractKeyPrefix()

	expiredKeys := []common.Bytes{}
	sv.store.Traverse(prefix, func(key, value common.Bytes) bool {
		var splitContract types.SplitContract
		err := types.FromBytes(value, &splitContract)
		if err != nil {
			panic(fmt.Sprintf("Error reading splitContract %X error: %v", value, err.Error()))
		}

		expired := (splitContract.EndBlockHeight < currentBlockHeight)
		if expired {
			expiredKeys = append(expiredKeys, key)
		}
		return true
	})

	for _, key := range expiredKeys {
		deleted := sv.store.Delete(key)
		if !deleted {
			log.Errorf("Failed to delete expired split contracts")
			return false
		}
	}

	return true
}

func (sv *StoreView) GetStore() *treestore.TreeStore {
	return sv.store
}
