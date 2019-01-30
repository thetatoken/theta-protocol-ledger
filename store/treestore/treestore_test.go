// +build cluster_deployment

package treestore

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/database/backend"
)

func newTestTreeStore() (database.Database, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "db_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	db, err := backend.NewBadgerDatabase(dirname)
	// db, err := backend.NewMongoDatabase()
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

	key333 := common.Bytes("test/333")
	value333 := common.Bytes("ddd")
	treestore.Set(key333, value333)

	key3331 := common.Bytes("test/3331")
	value3331 := common.Bytes("eee")
	treestore.Set(key3331, value3331)

	key334 := common.Bytes("test/334")
	value334 := common.Bytes("fff")
	treestore.Set(key334, value334)

	key3332 := common.Bytes("test/3332")
	value3332 := common.Bytes("ggg")
	treestore.Set(key3332, value3332)

	key33321 := common.Bytes("test/33321")
	value33321 := common.Bytes("hhh")
	treestore.Set(key33321, value33321)

	key33322 := common.Bytes("test/33322")
	value33322 := common.Bytes("iii")
	treestore.Set(key33322, value33322)

	key6 := common.Bytes("test/6")
	value6 := common.Bytes("6")
	treestore.Set(key6, value6)

	key66 := common.Bytes("test/66")
	value66 := common.Bytes("66")
	treestore.Set(key66, value66)

	key666 := common.Bytes("test/666")
	value666 := common.Bytes("666")
	treestore.Set(key666, value666)

	key6666 := common.Bytes("test/6666")
	value6666 := common.Bytes("6666")
	treestore.Set(key6666, value6666)

	key6667 := common.Bytes("test/6667")
	value6667 := common.Bytes("6667")
	treestore.Set(key6667, value6667)

	key667 := common.Bytes("test/667")
	value667 := common.Bytes("667")
	treestore.Set(key667, value667)

	key67 := common.Bytes("test/67")
	value67 := common.Bytes("67")
	treestore.Set(key67, value67)

	key676 := common.Bytes("test/676")
	value676 := common.Bytes("676")
	treestore.Set(key676, value676)

	key677 := common.Bytes("test/677")
	value677 := common.Bytes("677")
	treestore.Set(key677, value677)

	key6676 := common.Bytes("test/6676")
	value6676 := common.Bytes("6676")
	treestore.Set(key6676, value6676)

	key6677 := common.Bytes("test/6677")
	value6677 := common.Bytes("6677")
	treestore.Set(key6677, value6677)
	// for further use
	key66776 := common.Bytes("test/66776")
	value66776 := common.Bytes("66776")
	key33 := common.Bytes("test/33")
	value33 := common.Bytes("33")

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

	prefix3 := common.Bytes("test/33")
	treestore.Traverse(prefix3, cb(prefix3))
	assert.Equal(6, cnt)

	prefix4 := common.Bytes("test/333")
	treestore.Traverse(prefix4, cb(prefix4))
	assert.Equal(5, cnt)

	prefix5 := common.Bytes("test/3332")
	treestore.Traverse(prefix5, cb(prefix5))
	assert.Equal(3, cnt)

	prefix6 := common.Bytes("test/33322")
	treestore.Traverse(prefix6, cb(prefix6))
	assert.Equal(1, cnt)

	assert.Equal(value334, treestore.Get(key334))
	treestore.Set(key334, nil)
	assert.Nil(treestore.Get(key334))
	treestore.Set(key334, value334)
	assert.Equal(value334, treestore.Get(key334))
	treestore.Delete(key334)
	assert.Nil(treestore.Get(key334))

	root, _ := treestore.Commit()
	// treestore.Trie.GetDB().Commit(root, true)

	assert.True(db.Has(root[:]))
	assert.Equal(value1, treestore.Get(key1))
	assert.Equal(value2, treestore.Get(key2))
	assert.Equal(value3, treestore.Get(key3))
	assert.Equal(value3331, treestore.Get(key3331))
	assert.Equal(value66, treestore.Get(key66))

	//////////////////////////////

	treestore1 := NewTreeStore(treestore.Hash(), db)
	assert.Equal(value2, treestore1.Get(key2))

	treestore1.Set(key2, value3)
	assert.Equal(value3, treestore1.Get(key2))
	assert.Equal(value2, treestore.Get(key2))

	treestore1.Set(key333, nil)
	treestore1.Set(key66, common.Bytes("zzz"))
	treestore1.Set(key667, nil)
	treestore1.Set(key66776, value3)

	treestore1.Commit()
	// root1, _ := treestore1.Commit()
	// treestore1.GetDB().Commit(root1, true)

	//////////////////////////////

	treestore2 := NewTreeStore(treestore.Hash(), db)
	assert.Equal(value3332, treestore2.Get(key3332))

	treestore2.Set(key3332, common.Bytes("zzz"))
	assert.NotEqual(value3332, treestore2.Get(key3332))
	assert.Equal(value3332, treestore.Get(key3332))

	treestore2.Set(key677, nil)

	treestore2.Commit()
	// root2, _ := treestore2.Commit()
	// treestore2.GetDB().Commit(root2, true)
	//////////////////////////////

	treestore3 := NewTreeStore(treestore.Hash(), db)
	treestore3.Set(key66776, value66776)
	treestore3.Set(key33, value33)
	treestore3.Commit()
	assert.Equal(value66776, treestore3.Get(key66776))
	assert.Equal(value33, treestore3.Get(key33))

	//////////////////////////////

	treestore4 := NewTreeStore(treestore.Hash(), db)
	treestore4.Commit()

	//////////////////////////////

	hashMap := make(map[common.Hash]bool)
	hashMap1 := make(map[common.Hash]bool)
	hashMap2 := make(map[common.Hash]bool)
	hashMap3 := make(map[common.Hash]bool)

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

	for it := treestore2.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashMap2[it.Hash()] = true
		}
	}

	for it := treestore3.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hashMap3[it.Hash()] = true
		}
	}

	pruneStore := NewTreeStore(treestore.Hash(), db)
	pruneStore.Prune(nil)
	pruneStore = NewTreeStore(treestore1.Hash(), db)
	pruneStore.Prune(nil)
	pruneStore = NewTreeStore(treestore2.Hash(), db)
	pruneStore.Prune(nil)

	for hash := range hashMap {
		has, _ := db.Has(hash[:])
		if _, ok := hashMap3[hash]; ok {
			assert.True(has)
		} else {
			assert.False(has)
		}
	}

	for hash := range hashMap1 {
		has, _ := db.Has(hash[:])
		if _, ok := hashMap3[hash]; ok {
			assert.True(has)
		} else {
			assert.False(has)
		}
	}

	for hash := range hashMap2 {
		has, _ := db.Has(hash[:])
		if _, ok := hashMap3[hash]; ok {
			assert.True(has)
		} else {
			assert.False(has)
		}
	}

	for it := treestore3.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			ref, _ := db.CountReference(hash[:])
			assert.Equal(1, ref)
		}
	}

	//////////////////////////////

	pruneStore = NewTreeStore(treestore3.Hash(), db)
	pruneStore.Prune(nil)
}
