package treestore

import (
	"bytes"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/trie"
)

// NewTreeStore create a new instance of TreeStore.
func NewTreeStore(root common.Hash, db database.Database) *TreeStore {
	var tr *trie.Trie
	var err error
	tr, err = trie.New(root, trie.NewDatabase(db))
	if err != nil {
		log.Errorf("Failed to create tree store for: %v: %v", root.Hex(), err)
		return nil
	}
	return &TreeStore{tr, db}
}

type TreeStore struct {
	*trie.Trie
	db database.Database
}

// GetDB returns the underlying database.
func (store *TreeStore) GetDB() database.Database {
	return store.db
}

func (store *TreeStore) Commit() (common.Hash, error) {
	h, err := store.Trie.Commit(nil)
	if err != nil {
		return common.Hash{}, err
	}
	err = store.Trie.GetDB().Commit(h, true)
	if err != nil {
		return common.Hash{}, err
	}
	return h, nil
}

// Revert creates a copy of the Trie with the given root, using the
// in-memory trie DB (i.e. store.Trie.GetDB()) of the current Trie.
// Note: Each time we call Trie.Commit() a new root node will be created,
// however, the older roots are still stored in the in-memory trie DB. The root
// passed to the Revert() function needs to be one of the previous roots,
// otherwise the function will return an error.
func (store *TreeStore) Revert(root common.Hash) (*TreeStore, error) {
	trieDB := store.Trie.GetDB()
	revertedTrie, err := trie.New(root, trieDB)
	if err != nil {
		return nil, err
	}

	revertedStore := &TreeStore{
		Trie: revertedTrie,
		db:   store.db,
	}
	return revertedStore, nil
}

// Copy returns a copy of the TreeStore
func (store *TreeStore) Copy() (*TreeStore, error) {
	store.Trie.Commit(nil)
	copiedTrie, err := store.Trie.Copy()
	if err != nil {
		return nil, err
	}

	copiedStore := &TreeStore{copiedTrie, store.db}
	return copiedStore, nil
}

// Get retrieves value of given key.
func (store *TreeStore) Get(key common.Bytes) common.Bytes {
	return store.Trie.Get(key)
}

func (store *TreeStore) ProveVCP(vcpKey []byte, vp *core.VCPProof) error {
	return store.Trie.Prove(vcpKey, 0, vp)
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
func (store *TreeStore) Prune(cb func(n []byte) bool) error {
	return store.Trie.Prune(cb)
}
