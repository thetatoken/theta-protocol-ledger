package treestore

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestTreeStore(t *testing.T) {
	assert := assert.New(t)

	db, err := backend.NewMgoDatabase()
	assert.Nil(err)
	treestore := NewTreeStore(common.Hash{}, db, false)

	key1 := []byte("test/111")
	value1 := []byte("aaa")

	treestore.Set(key1, value1)
	assert.Equal(value1, treestore.Get(key1))

	key2 := []byte("test/123")
	value2 := []byte("bbb")
	treestore.Set(key2, value2)

	key3 := []byte("test/222")
	value3 := []byte("ccc")
	treestore.Set(key3, value3)

	key4 := []byte("test/333")
	value4 := []byte("ddd")
	treestore.Set(key4, value4)

	key5 := []byte("test/3331")
	value5 := []byte("eee")
	treestore.Set(key5, value5)

	key6 := []byte("test/334")
	value6 := []byte("fff")
	treestore.Set(key6, value6)

	root, _ := treestore.Commit(nil)
	treestore.GetDB().Commit(root, true)

	assert.True(db.Has(root[:]))

	assert.Equal(value2, treestore.Get(key2))
	assert.Equal(value3, treestore.Get(key3))
	assert.Equal(value4, treestore.Get(key4))
	assert.Equal(value5, treestore.Get(key5))
	assert.Equal(value6, treestore.Get(key6))

	var cnt int

	cb := func(prefix []byte) func(k, v []byte) bool {
		cnt = 0
		return func(k, v []byte) bool {
			cnt++
			success := bytes.HasPrefix(k, prefix)
			success = success && (bytes.Compare(v, treestore.Get(k)) == 0)
			return success
		}
	}

	prefix1 := []byte("test/1")
	treestore.Traverse(prefix1, cb(prefix1))
	assert.Equal(2, cnt)

	prefix2 := []byte("test/2")
	treestore.Traverse(prefix2, cb(prefix2))
	assert.Equal(1, cnt)

	prefix3 := []byte("test/333")
	treestore.Traverse(prefix3, cb(prefix3))
	assert.Equal(2, cnt)

	prefix4 := []byte("test/33")
	treestore.Traverse(prefix4, cb(prefix4))
	assert.Equal(3, cnt)

	prefix5 := []byte("test")
	treestore.Traverse(prefix5, cb(prefix5))
	assert.Equal(6, cnt)

	treestore.Set(key1, nil)
	assert.Nil(treestore.Get(key1))
	treestore.Traverse(prefix5, cb(prefix5))
	assert.Equal(5, cnt)

	root, _ = treestore.Commit(nil)
	treestore.Trie.GetDB().Commit(root, true)

	//////////////////////////////

	treestore1 := NewTreeStore(treestore.Hash(), db, false)
	assert.Nil(treestore1.Get(key1))
	assert.Equal(value2, treestore1.Get(key2))

	treestore1.Set(key2, value3)
	assert.Equal(value3, treestore1.Get(key2))
	assert.Equal(value2, treestore.Get(key2))

	//////////////////////////////

	nonpersistentstore := NewTreeStore(common.Hash{}, db, true)

	key7 := []byte("test/000")
	value7 := []byte("zzz")

	nonpersistentstore.Set(key7, value7)
	assert.Equal(value7, nonpersistentstore.Get(key7))

	root, _ = nonpersistentstore.Commit(nil)
	nonpersistentstore.GetDB().Commit(root, true)

	assert.False(db.Has(root[:]))
}
