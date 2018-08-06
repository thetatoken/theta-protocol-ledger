package store

import (
	"encoding/hex"

	"github.com/pkg/errors"

	"github.com/thetatoken/ukulele/types"
)

var _ Store = MemKVStore{}

var (
	// ErrKeyNotFound for key is not found in store.
	ErrKeyNotFound = errors.New("KeyNotFound")
)

// MemKVStore is a in-memory implementation of Store to be used in testing.
type MemKVStore map[string]interface{}

// NewMemKVStore create a new instance of MemKVStore.
func NewMemKVStore() MemKVStore {
	return make(MemKVStore)
}

func getKey(key types.Bytes) string {
	return hex.EncodeToString(key)
}

// Put implements Store.Put().
func (mkv MemKVStore) Put(key types.Bytes, value interface{}) error {
	keystr := getKey(key)
	mkv[keystr] = value
	return nil
}

// Delete implements Store.Delete().
func (mkv MemKVStore) Delete(key types.Bytes) error {
	delete(mkv, getKey(key))
	return nil
}

// Get implements Store.Get().
func (mkv MemKVStore) Get(key types.Bytes) (value interface{}, err error) {
	value, ok := mkv[getKey(key)]
	if !ok {
		err = ErrKeyNotFound
	}
	return
}
