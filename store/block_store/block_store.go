package block_store

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/serialization/rlp"
	"github.com/thetatoken/ukulele/store/database/backend"
)

// BlockStore a Database wrapped object.
type BlockStore struct {
	db *backend.LDBDatabase
}

// Put upserts key/value into DB
func (store *BlockStore) Put(key common.Bytes, value interface{}) error {
	encodedKey, err := rlp.EncodeToBytes(key)
	if err != nil {
		return err
	}
	encodedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	return store.db.Put(encodedKey, encodedValue)
}

// Delete deletes key entry from DB
func (store *BlockStore) Delete(key common.Bytes) error {
	encodedKey, err := rlp.EncodeToBytes(key)
	if err != nil {
		return err
	}
	return store.db.Delete(encodedKey)
}

// Get looks up DB with key and returns result into value (passed by reference)
func (store *BlockStore) Get(key common.Bytes, value interface{}) error {
	encodedKey, err := rlp.EncodeToBytes(key)
	if err != nil {
		return err
	}
	encodedValue, err := store.db.Get(encodedKey)
	if err != nil {
		return err
	}
	return rlp.DecodeBytes(encodedValue, value)
}
