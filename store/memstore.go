package store

import (
	"encoding/hex"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

var _ Store = MemKVStore{}

// MemKVStore is a in-memory implementation of Store to be used in testing.
type MemKVStore map[string]interface{}

// NewMemKVStore create a new instance of MemKVStore.
func NewMemKVStore() MemKVStore {
	return make(MemKVStore)
}

func getKey(key common.Bytes) string {
	return hex.EncodeToString(key)
}

// Put implements Store.Put().
func (mkv MemKVStore) Put(key common.Bytes, value interface{}) error {
	encodedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}

	keystr := getKey(key)
	mkv[keystr] = encodedValue
	// mkv[keystr] = value
	return nil
}

// Delete implements Store.Delete().
func (mkv MemKVStore) Delete(key common.Bytes) error {
	delete(mkv, getKey(key))
	return nil
}

// Get implements Store.Get().
func (mkv MemKVStore) Get(key common.Bytes, value interface{}) error {
	encodedValue, ok := mkv[getKey(key)]
	if !ok {
		return ErrKeyNotFound
	}
	return rlp.DecodeBytes(encodedValue.([]byte), value)
}
