package blockstore

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/serialization/rlp"
	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/database"
)

// NewBlockStore create a new instance of BlockStore.
func NewBlockStore(db database.Database) store.Store {
	return &BlockStore{db}
}

// BlockStore a Database wrapped object.
type BlockStore struct {
	db database.Database
}

// Put upserts key/value into DB
func (store *BlockStore) Put(key common.Bytes, value interface{}) error {
	encodedValue, err := rlp.EncodeToBytes(value)
	if err != nil {
		return err
	}
	return store.db.Put(key, encodedValue)
}

// Delete deletes key entry from DB
func (store *BlockStore) Delete(key common.Bytes) error {
	return store.db.Delete(key)
}

// Get looks up DB with key and returns result into value (passed by reference)
func (store *BlockStore) Get(key common.Bytes, value interface{}) error {
	encodedValue, err := store.db.Get(key)
	if err != nil {
		return err
	}
	return rlp.DecodeBytes(encodedValue, value)
}
