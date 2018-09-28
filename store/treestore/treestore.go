package treestore

import (
	"bytes"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/trie"
)

// NewTreeStore create a new instance of TreeStore.
func NewTreeStore(root common.Hash, db database.Database, nonpersistent bool) *TreeStore {
	var tr *trie.Trie
	var err error
	if nonpersistent {
		tr, err = trie.New(root, trie.NewNonpersistentDatabase(db))
	} else {
		tr, err = trie.New(root, trie.NewDatabase(db))
	}
	if err != nil {
		return nil
	}
	return &TreeStore{tr}
}

type TreeStore struct {
	*trie.Trie
}

// Copy returns a copy of the TreeStore
func (store *TreeStore) Copy() (*TreeStore, error) {
	copiedTrie, err := store.Trie.Copy()
	if err != nil {
		return nil, err
	}

	copiedStore := &TreeStore{copiedTrie}
	return copiedStore, nil
}

// Get retrieves value of given key.
func (store *TreeStore) Get(key common.Bytes) common.Bytes {
	return store.Trie.Get(key)
}

// Set sets value of given key.
func (store *TreeStore) Set(key, value common.Bytes) {
	store.Trie.Update(key, value)
}

// Traverse traverses the trie and calls cb callback func on every key/value pair
// with key having prefix
func (store *TreeStore) Traverse(prefix common.Bytes, cb func(k, v common.Bytes) bool) bool {
	// TODO: find alternative way without traversal
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

// Delete deletes the key/value pair.
func (store *TreeStore) Delete(key common.Bytes) (deleted bool) {
	store.Trie.Delete(key)
	return true
}

// Prune deletes all non-referenced nodes.
func (store *TreeStore) Prune() error {
	return store.Trie.Prune()
}
