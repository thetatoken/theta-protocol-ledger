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
	"sync"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
)

/*
 * This is a test memory database. Do not use for any production it does not get persisted
 */
type MemDatabase struct {
	db    map[string][]byte
	refdb map[string]int
	lock  sync.RWMutex
}

func NewMemDatabase() *MemDatabase {
	return &MemDatabase{
		db:    make(map[string][]byte),
		refdb: make(map[string]int),
	}
}

func NewMemDatabaseWithCap(size int) *MemDatabase {
	return &MemDatabase{
		db:    make(map[string][]byte, size),
		refdb: make(map[string]int, size),
	}
}

func (db *MemDatabase) Put(key []byte, value []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.db[string(key)] = common.CopyBytes(value)
	return nil
}

func (db *MemDatabase) Has(key []byte) (bool, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	_, ok := db.db[string(key)]
	return ok, nil
}

func (db *MemDatabase) Get(key []byte) ([]byte, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	if entry, ok := db.db[string(key)]; ok {
		return common.CopyBytes(entry), nil
	}
	return nil, store.ErrKeyNotFound
}

func (db *MemDatabase) Keys() [][]byte {
	db.lock.RLock()
	defer db.lock.RUnlock()

	keys := [][]byte{}
	for key := range db.db {
		keys = append(keys, []byte(key))
	}
	return keys
}

func (db *MemDatabase) Delete(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	delete(db.refdb, string(key))
	delete(db.db, string(key))
	return nil
}

func (db *MemDatabase) Reference(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// check if k/v exists
	if _, ok := db.db[string(key)]; ok {
		db.refdb[string(key)] = db.refdb[string(key)] + 1
		return nil
	}
	return store.ErrKeyNotFound
}

func (db *MemDatabase) Dereference(key []byte) error {
	db.lock.Lock()
	defer db.lock.Unlock()

	// check if k/v exists
	if _, ok := db.db[string(key)]; ok {
		if db.refdb[string(key)] > 0 {
			db.refdb[string(key)] = db.refdb[string(key)] - 1
		}
		return nil
	}
	return store.ErrKeyNotFound
}

func (db *MemDatabase) CountReference(key []byte) (int, error) {
	db.lock.RLock()
	defer db.lock.RUnlock()

	// check if k/v exists
	if ref, ok := db.refdb[string(key)]; ok {
		return ref, nil
	}
	return 0, store.ErrKeyNotFound
}

func (db *MemDatabase) Close() {}

func (db *MemDatabase) NewBatch() database.Batch {
	return &memBatch{db: db, references: make(map[string]int)}
}

func (db *MemDatabase) Len() int { return len(db.db) }

type kv struct {
	k, v []byte
	del  bool
}

type memBatch struct {
	db         *MemDatabase
	writes     []kv
	references map[string]int
	size       int
}

func (b *memBatch) Put(key, value []byte) error {
	b.writes = append(b.writes, kv{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

func (b *memBatch) Delete(key []byte) error {
	delete(b.db.refdb, string(key))
	b.writes = append(b.writes, kv{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

func (b *memBatch) Reference(key []byte) error {
	b.references[string(key)]++
	b.size++
	return nil
}

func (b *memBatch) Dereference(key []byte) error {
	b.references[string(key)]--
	b.size++
	return nil
}

func (b *memBatch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, kv := range b.writes {
		if kv.del {
			delete(b.db.db, string(kv.k))
			continue
		}
		b.db.db[string(kv.k)] = kv.v
	}

	for k, v := range b.references {
		if v == 0 {
			// refs and derefs canceled out
			delete(b.references, k)
		}
	}

	for k, v := range b.references {
		if _, ok := b.db.db[k]; !ok {
			if v < 0 {
				continue
			}
			b.db.refdb[string(k)] = 0
		}
		b.db.refdb[string(k)] = b.db.refdb[string(k)] + v
		if b.db.refdb[string(k)] < 0 {
			b.db.refdb[string(k)] = 0
		}
	}

	b.Reset()

	return nil
}

func (b *memBatch) ValueSize() int {
	return b.size
}

func (b *memBatch) Reset() {
	b.writes = b.writes[:0]
	b.references = make(map[string]int)
	b.size = 0
}
