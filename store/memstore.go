package store

import (
	"encoding/hex"
	"sync"

	"github.com/pkg/errors"
	"github.com/thetatoken/ukulele/common"
)

var _ Store = MemKVStore{}

var (
	// ErrKeyNotFound for key is not found in store.
	ErrKeyNotFound = errors.New("KeyNotFound")
)

// MemKVStore is a in-memory implementation of Store to be used in testing.
type MemKVStore struct {
	*sync.RWMutex
	data map[string]interface{}
}

// NewMemKVStore create a new instance of MemKVStore.
func NewMemKVStore() MemKVStore {
	return MemKVStore{
		RWMutex: &sync.RWMutex{},
		data:    make(map[string]interface{}),
	}
}

func getKey(key common.Bytes) string {
	return hex.EncodeToString(key)
}

// Put implements Store.Put().
func (mkv MemKVStore) Put(key common.Bytes, value interface{}) error {
	mkv.Lock()
	defer mkv.Unlock()

	keystr := getKey(key)
	mkv.data[keystr] = value
	return nil
}

// Delete implements Store.Delete().
func (mkv MemKVStore) Delete(key common.Bytes) error {
	mkv.Lock()
	defer mkv.Unlock()

	delete(mkv.data, getKey(key))
	return nil
}

// Get implements Store.Get().
func (mkv MemKVStore) Get(key common.Bytes) (value interface{}, err error) {
	mkv.RLock()
	defer mkv.RUnlock()

	value, ok := mkv.data[getKey(key)]
	if !ok {
		err = ErrKeyNotFound
	}
	return
}
