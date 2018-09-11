package statestore

import (
	"bytes"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/trie"
)

// NewStateStore create a new instance of StateStore.
func NewStateStore(root common.Hash, db database.Database, noWrite bool) *StateStore {
	var tr *trie.Trie
	var err error
	if noWrite {
		tr, err = trie.New(root, trie.NewDatabaseWithoutFlush(db))
	} else {
		tr, err = trie.New(root, trie.NewDatabase(db))
	}
	if err != nil {
		return nil
	}
	return &StateStore{tr}
}

type StateStore struct {
	*trie.Trie
}

// Get retrieves value of given key.
func (store *StateStore) Get(key []byte) []byte {
	return store.Trie.Get(key)
}

// Set sets value of given key.
func (store *StateStore) Set(key, value []byte) {
	store.Trie.Update(key, value)
}

// Traverse traverses the trie and calls cb callback func on every key/value pair
// with key having prefix
func (store *StateStore) Traverse(prefix []byte, cb func([]byte, []byte) bool) bool {
	it := trie.NewIterator(store.Trie.NodeIterator(prefix))
	for it.Next() {
		if bytes.HasPrefix(it.Key, prefix) {
			cb(it.Key, it.Value)
		} else {
			break
		}
	}
	return true
}
