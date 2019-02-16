// Adapted for Theta
// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package backend

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
)

func newTestLDB() (*LDBDatabase, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "ethdb_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	refname, err := ioutil.TempDir(os.TempDir(), "ethdb_ref_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	db, err := NewLDBDatabase(dirname, refname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
		os.RemoveAll(refname)
	}
}

var testValues = []string{"a", "1251", "\x00123\x00"}

func TestLDB_PutGet(t *testing.T) {
	db, remove := newTestLDB()
	batch := db.NewBatch()
	defer remove()
	testPutGet(db, batch, t)
}

func TestMemoryDB_PutGet(t *testing.T) {
	memDB := NewMemDatabase()
	testPutGet(memDB, memDB.NewBatch(), t)
}

func testPutGet(db database.Database, batch database.Batch, t *testing.T) {
	t.Parallel()

	for _, k := range testValues {
		err := db.Put([]byte(k), nil)
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	for _, k := range testValues {
		data, err := db.Get([]byte(k))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if len(data) != 0 {
			t.Fatalf("get returned wrong result, got %q expected nil", string(data))
		}
	}

	_, err := db.Get([]byte("non-exist-key"))
	if err == nil {
		t.Fatalf("expect to return a not found error")
	}

	exists, err := db.Has([]byte("non-exist-key"))
	if err != nil {
		t.Fatalf("has failed: %v", err)
	}
	if exists {
		t.Fatalf("expect to return not found")
	}

	for _, v := range testValues {
		err := db.Put([]byte(v), []byte(v))
		if err != nil {
			t.Fatalf("put failed: %v", err)
		}
	}

	for _, v := range testValues {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v)) {
			t.Fatalf("get returned wrong result, got %q expected %q", string(data), v)
		}
	}

	err = db.Reference([]byte("non-exist-key"))
	if err == nil || err != store.ErrKeyNotFound {
		t.Fatalf("reference non existent key didn't fail")
	}

	err = db.Dereference([]byte("non-exist-key"))
	if err == nil || err != store.ErrKeyNotFound {
		t.Fatalf("dereference non existent key didn't fail")
	}

	_, err = db.CountReference([]byte("non-exist-key"))
	if err == nil || err != store.ErrKeyNotFound {
		t.Fatalf("count reference non existent key didn't fail")
	}

	// test dereference on nil refs first
	for _, k := range testValues {
		err := db.Dereference([]byte(k))
		if err != nil {
			t.Fatalf("dereference failed: %v", err)
		}
	}

	for _, k := range testValues {
		ref, _ := db.CountReference([]byte(k))
		if ref != 0 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 0)
		}
	}

	for i, k := range testValues {
		for j := 0; j <= i; j++ {
			err := db.Reference([]byte(k))
			if err != nil {
				t.Fatalf("reference failed: %v", err)
			}
		}
	}

	for i, k := range testValues {
		ref, err := db.CountReference([]byte(k))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != i+1 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, i+1)
		}
	}

	for i, k := range testValues {
		for j := 0; j <= i; j++ {
			err := db.Dereference([]byte(k))
			if err != nil {
				t.Fatalf("dereference failed: %v", err)
			}
		}

		ref, err := db.CountReference([]byte(k))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 0 {
			t.Fatalf("count reference returned wrong result, got %d expected 0", ref)
		}
	}

	for _, k := range testValues {
		err := db.Dereference([]byte(k))
		if err != nil {
			t.Fatalf("dereference failed: %v", err)
		}
	}

	for _, k := range testValues {
		ref, err := db.CountReference([]byte(k))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 0 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 0)
		}
	}

	for _, k := range testValues {
		err := db.Reference([]byte(k))
		if err != nil {
			t.Fatalf("reference failed: %v", err)
		}
	}

	for _, v := range testValues {
		err := db.Put([]byte(v), []byte("?"))
		if err != nil {
			t.Fatalf("put override failed: %v", err)
		}
	}

	for _, v := range testValues {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range testValues {
		orig, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		orig[0] = byte(0xff)
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte("?")) {
			t.Fatalf("get returned wrong result, got %q expected ?", string(data))
		}
	}

	for _, v := range testValues {
		has, err := db.Has([]byte(v))
		if err != nil {
			t.Fatalf("has failed: %v", err)
		}
		if !has {
			t.Fatalf("can't find %v", v)
		}
	}

	for _, v := range testValues {
		err := db.Delete([]byte(v))
		if err != nil {
			t.Fatalf("delete %q failed: %v", v, err)
		}
	}

	for _, v := range testValues {
		has, err := db.Has([]byte(v))
		if err != nil {
			t.Fatalf("has failed: %v", err)
		}
		if has {
			t.Fatalf("find deleted %v", v)
		}
	}

	for _, v := range testValues {
		_, err := db.Get([]byte(v))
		if err == nil || err != store.ErrKeyNotFound {
			t.Fatalf("got deleted value %q", v)
		}
	}

	// test batch
	for _, v := range testValues {
		err := batch.Put([]byte(v), nil)
		if err != nil {
			t.Fatalf("batch put %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		has, err := db.Has([]byte(v))
		if err != nil {
			t.Fatalf("has failed: %v", err)
		}
		if !has {
			t.Fatalf("can't find %v", v)
		}
	}

	for _, k := range testValues {
		data, err := db.Get([]byte(k))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if len(data) != 0 {
			t.Fatalf("get returned wrong result, got %q expected nil", string(data))
		}
	}

	for _, v := range testValues {
		err := batch.Put([]byte(v), []byte(v))
		if err != nil {
			t.Fatalf("batch put %q failed: %v", v, err)
		}
		err = batch.Reference([]byte(v))
		if err != nil {
			t.Fatalf("batch reference %q failed: %v", v, err)
		}
		err = batch.Dereference([]byte(v))
		if err != nil {
			t.Fatalf("batch dereference %q failed: %v", v, err)
		}
		err = batch.Reference([]byte(v))
		if err != nil {
			t.Fatalf("batch reference %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		data, err := db.Get([]byte(v))
		if err != nil {
			t.Fatalf("get failed: %v", err)
		}
		if !bytes.Equal(data, []byte(v)) {
			t.Fatalf("get returned wrong result, got %q expected %q", string(data), v)
		}

		ref, err := db.CountReference([]byte(v))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 1 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 1)
		}
	}

	for _, v := range testValues {
		err := batch.Dereference([]byte(v))
		if err != nil {
			t.Fatalf("batch dereference %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		ref, err := db.CountReference([]byte(v))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 0 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 0)
		}
	}

	for _, v := range testValues {
		err := batch.Delete([]byte(v))
		if err != nil {
			t.Fatalf("batch delete %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		has, err := db.Has([]byte(v))
		if err != nil {
			t.Fatalf("has failed: %v", err)
		}
		if has {
			t.Fatalf("find deleted %v", v)
		}
	}

	for _, v := range testValues {
		err = batch.Dereference([]byte(v))
		if err != nil {
			t.Fatalf("batch dereference %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		ref, err := db.CountReference([]byte(v))
		if err == nil || err != store.ErrKeyNotFound {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 0 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 0)
		}
	}

	for _, v := range testValues {
		err = batch.Reference([]byte(v))
		if err != nil {
			t.Fatalf("batch reference %q failed: %v", v, err)
		}
	}
	err = batch.Write()
	if err != nil {
		t.Fatalf("batch write failed: %v", err)
	}

	for _, v := range testValues {
		ref, err := db.CountReference([]byte(v))
		if err != nil {
			t.Fatalf("count reference failed: %v", err)
		}
		if ref != 1 {
			t.Fatalf("count reference returned wrong result, got %d expected %d", ref, 1)
		}
	}
}

func TestLDB_ParallelPutGet(t *testing.T) {
	db, remove := newTestLDB()
	defer remove()
	testParallelPutGet(db, t)
}

func TestMemoryDB_ParallelPutGet(t *testing.T) {
	testParallelPutGet(NewMemDatabase(), t)
}

func testParallelPutGet(db database.Database, t *testing.T) {
	const n = 8
	var pending sync.WaitGroup

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			err := db.Put([]byte(key), []byte("v"+key))
			if err != nil {
				panic("put failed: " + err.Error())
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			data, err := db.Get([]byte(key))
			if err != nil {
				panic("get failed: " + err.Error())
			}
			if !bytes.Equal(data, []byte("v"+key)) {
				panic(fmt.Sprintf("get failed, got %q expected %q", []byte(data), []byte("v"+key)))
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			err := db.Delete([]byte(key))
			if err != nil {
				panic("delete failed: " + err.Error())
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()

	pending.Add(n)
	for i := 0; i < n; i++ {
		go func(key string) {
			defer pending.Done()
			_, err := db.Get([]byte(key))
			if err == nil {
				panic("get succeeded")
			}
		}(strconv.Itoa(i))
	}
	pending.Wait()
}
