package state

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/treestore"
)

//
// ------------------------- StoreView -------------------------
//

type StoreView struct {
	store *treestore.TreeStore
}

// Copy returns a copy of the StoreView
func (sv *StoreView) Copy() StoreView {
	// TODO: need proper implementation
	return StoreView{}
}

// Save saves the StoreView to the persistent storage, and return the root hash
func (sv *StoreView) Save() common.Hash {
	// TODO: need proper implementation
	return common.Hash{}
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
	if len(data) == 0 {
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
func (sv *StoreView) GetSplitContract(resourceId common.Bytes) *types.SplitContract {
	data := sv.Get(SplitContractKey(resourceId))
	if len(data) == 0 {
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
func (sv *StoreView) SetSplitContract(resourceId common.Bytes, splitContract *types.SplitContract) {
	splitContractBytes, err := types.ToBytes(splitContract)
	if err != nil {
		panic(fmt.Sprintf("Error writing splitContract %v error: %v",
			splitContract, err.Error()))
	}
	sv.Set(SplitContractKey(resourceId), splitContractBytes)
}

// DeleteSplitContract implements the ViewDataAccessor DeleteSplitContract() method
func (sv *StoreView) DeleteSplitContract(resourceId common.Bytes) (SplitContractBytes common.Bytes, deleted bool) {
	key := SplitContractKey(resourceId)
	splitContractBytes, deleted := sv.store.Delete(key)
	return splitContractBytes, deleted
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
		return expired
	})

	for _, key := range expiredKeys {
		sv.store.Delete(key)
	}

	return true
}
