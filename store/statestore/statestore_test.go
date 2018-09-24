package statestore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestStateStore(t *testing.T) {
	assert := assert.New(t)

	db, err := backend.NewAerospikeDatabase()
	// db, err := backend.NewMgoDatabase()
	assert.Nil(err)
	statestore := NewStateStore(common.Hash{}, db, false)

	key1 := []byte("s1")
	value1 := []byte("aaa")

	statestore.Set(key1, value1)
	assert.Equal(value1, statestore.Get(key1))

	key2 := []byte("s2")
	value2 := []byte("bbb")
	statestore.Set(key2, value2)

	key3 := []byte("s3")
	value3 := []byte("ddd")
	statestore.Set(key3, value3)

	root, _ := statestore.Commit(nil)
	statestore.GetDB().Commit(root, true)

	statestore1 := NewStateStore(statestore.Hash(), db, false)
	statestore1.Set(key2, value3)
	assert.Equal(value3, statestore1.Get(key2))
	assert.Equal(value2, statestore.Get(key2))

	// key4 := []byte("test/333")
	// value4 := []byte("ddd")
	// statestore.Set(key4, value4)

	// key5 := []byte("test/3331")
	// value5 := []byte("eee")
	// statestore.Set(key5, value5)

	// key6 := []byte("test/334")
	// value6 := []byte("fff")
	// statestore.Set(key6, value6)

	// root, _ := statestore.Commit(nil)
	// statestore.GetDB().Commit(root, true)

	// assert.True(db.Has(root[:]))

	// assert.Equal(value2, statestore.Get(key2))
	// assert.Equal(value3, statestore.Get(key3))
	// assert.Equal(value4, statestore.Get(key4))
	// assert.Equal(value5, statestore.Get(key5))
	// assert.Equal(value6, statestore.Get(key6))

	// var cnt int

	// cb := func(prefix []byte) func(k, v []byte) bool {
	// 	cnt = 0
	// 	return func(k, v []byte) bool {
	// 		cnt++
	// 		success := bytes.HasPrefix(k, prefix)
	// 		success = success && (bytes.Compare(v, statestore.Get(k)) == 0)
	// 		return success
	// 	}
	// }

	// prefix1 := []byte("test/1")
	// statestore.Traverse(prefix1, cb(prefix1))
	// assert.Equal(2, cnt)

	// prefix2 := []byte("test/2")
	// statestore.Traverse(prefix2, cb(prefix2))
	// assert.Equal(1, cnt)

	// prefix3 := []byte("test/333")
	// statestore.Traverse(prefix3, cb(prefix3))
	// assert.Equal(2, cnt)

	// prefix4 := []byte("test/33")
	// statestore.Traverse(prefix4, cb(prefix4))
	// assert.Equal(3, cnt)

	// prefix5 := []byte("test")
	// statestore.Traverse(prefix5, cb(prefix5))
	// assert.Equal(6, cnt)

	// statestore.Set(key1, nil)
	// assert.Nil(statestore.Get(key1))
	// statestore.Traverse(prefix5, cb(prefix5))
	// assert.Equal(5, cnt)

	// root, _ = statestore.Commit(nil)
	// statestore.Trie.GetDB().Commit(root, true)

	// //////////////////////////////

	// statestore1 := NewStateStore(statestore.Hash(), db, false)
	// assert.Nil(statestore1.Get(key1))
	// assert.Equal(value2, statestore1.Get(key2))

	// statestore1.Set(key2, value3)
	// assert.Equal(value3, statestore1.Get(key2))
	// assert.Equal(value2, statestore.Get(key2))

	// //////////////////////////////

	// nonpersistentstore := NewStateStore(common.Hash{}, db, true)

	// key7 := []byte("test/000")
	// value7 := []byte("zzz")

	// nonpersistentstore.Set(key7, value7)
	// assert.Equal(value7, nonpersistentstore.Get(key7))

	// root, _ = nonpersistentstore.Commit(nil)
	// nonpersistentstore.GetDB().Commit(root, true)

	// assert.False(db.Has(root[:]))
}
