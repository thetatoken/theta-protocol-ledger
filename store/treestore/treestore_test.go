package treestore

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func newTestTreeStore() (database.Database, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "db_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	db, err := backend.NewBadgerDatabase(dirname)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}

func TestTreeStore(t *testing.T) {
	db, close := newTestTreeStore()
	defer close()
	testTreeStore(db, t)
}

func testTreeStore(db database.Database, t *testing.T) {
	assert := assert.New(t)

	treestore := NewTreeStore(common.Hash{}, db)

	key1 := common.Bytes("s1")
	value1 := common.Bytes("aaa")

	treestore.Set(key1, value1)
	assert.Equal(value1, treestore.Get(key1))

	key2 := common.Bytes("s2")
	value2 := common.Bytes("bbb")
	treestore.Set(key2, value2)

	key3 := common.Bytes("s3")
	value3 := common.Bytes("ccc")
	treestore.Set(key3, value3)

	key4 := common.Bytes("test/333")
	value4 := common.Bytes("ddd")
	treestore.Set(key4, value4)

	key5 := common.Bytes("test/3331")
	value5 := common.Bytes("eee")
	treestore.Set(key5, value5)

	key6 := common.Bytes("test/334")
	value6 := common.Bytes("fff")
	treestore.Set(key6, value6)

	key7 := common.Bytes("test/3332")
	value7 := common.Bytes("ggg")
	treestore.Set(key7, value7)

	key8 := common.Bytes("test/33321")
	value8 := common.Bytes("hhh")
	treestore.Set(key8, value8)

	key9 := common.Bytes("test/33322")
	value9 := common.Bytes("iii")
	treestore.Set(key9, value9)

	var cnt int

	cb := func(prefix common.Bytes) func(k, v common.Bytes) bool {
		cnt = 0
		return func(k, v common.Bytes) bool {
			cnt++
			success := bytes.HasPrefix(k, prefix)
			success = success && (bytes.Compare(v, treestore.Get(k)) == 0)
			return success
		}
	}

	prefix1 := common.Bytes("s1")
	treestore.Traverse(prefix1, cb(prefix1))
	assert.Equal(1, cnt)

	prefix2 := common.Bytes("s")
	treestore.Traverse(prefix2, cb(prefix2))
	assert.Equal(3, cnt)

	prefix3 := common.Bytes("test/333")
	treestore.Traverse(prefix3, cb(prefix3))
	assert.Equal(5, cnt)

	prefix4 := common.Bytes("test/33")
	treestore.Traverse(prefix4, cb(prefix4))
	assert.Equal(6, cnt)

	assert.Equal(value9, treestore.Get(key9))
	treestore.Set(key9, nil)
	assert.Nil(treestore.Get(key9))
	treestore.Set(key9, value9)
	assert.Equal(value9, treestore.Get(key9))
	treestore.Delete(key9)
	assert.Nil(treestore.Get(key9))

	treestore.Traverse(prefix3, cb(prefix3))
	assert.Equal(4, cnt)
	treestore.Traverse(prefix4, cb(prefix4))
	assert.Equal(5, cnt)

	root, _ := treestore.Commit(nil)
	treestore.Trie.GetDB().Commit(root, true)

	assert.True(db.Has(root[:]))
	assert.Equal(value1, treestore.Get(key1))
	assert.Equal(value2, treestore.Get(key2))
	assert.Equal(value3, treestore.Get(key3))
	assert.Equal(value4, treestore.Get(key4))
	assert.Equal(value5, treestore.Get(key5))
	assert.Equal(value6, treestore.Get(key6))
	assert.Equal(value7, treestore.Get(key7))
	assert.Equal(value8, treestore.Get(key8))

	//////////////////////////////

	treestore1 := NewTreeStore(treestore.Hash(), db)
	assert.Equal(value2, treestore1.Get(key2))

	treestore1.Set(key2, value3)
	assert.Equal(value3, treestore1.Get(key2))
	assert.Equal(value2, treestore.Get(key2))

	root1, _ := treestore1.Commit(nil)
	treestore1.GetDB().Commit(root1, true)

	//////////////////////////////

	hashMap := make(map[common.Hash]bool)
	hashMap1 := make(map[common.Hash]bool)

	for it := treestore.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashMap[it.Hash()] = true
		}
	}

	for it := treestore1.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashMap1[it.Hash()] = true
		}
	}

	pruneStore := NewTreeStore(treestore.Hash(), db)
	pruneStore.Prune()

	assert.False(db.Has(root[:]))
	assert.True(db.Has(root1[:]))

	for hash := range hashMap {
		has, _ := db.Has(hash[:])
		if _, ok := hashMap1[hash]; ok {
			assert.True(has)
		} else {
			assert.False(has)
		}
	}

	for it := treestore1.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			ref, _ := db.CountReference(hash[:])
			assert.Equal(1, ref)
		}
	}

	//////////////////////////////

	for it := treestore.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			db.Delete(hash[:])
		}
	}

	for it := treestore1.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			db.Delete(hash[:])
		}
	}
}
