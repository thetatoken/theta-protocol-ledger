package store

import (
	"encoding/hex"
	"sync"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

var _ Store = MemKVStore{}

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

	encodedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	mkv.data[keystr] = encodedValue

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
func (mkv MemKVStore) Get(key common.Bytes, value interface{}) error {
	mkv.RLock()
	defer mkv.RUnlock()

	encodedValue, ok := mkv.data[getKey(key)]
	if !ok {
		return ErrKeyNotFound
	}
	return rlp.DecodeBytes(encodedValue.([]byte), value)
}
