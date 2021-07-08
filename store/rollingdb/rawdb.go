package rollingdb

import (
	"time"

	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
)

type RawDB struct {
	fn string      // filename for reporting
	db *leveldb.DB // LevelDB instance
}

const (
	writePauseWarningThrottler = 1 * time.Minute
)

var OpenFileLimit = 64

func NewRawDB(file string) (*RawDB, error) {
	cache := viper.GetInt(common.CfgStorageLevelDBCacheSize)
	handles := viper.GetInt(common.CfgStorageLevelDBHandles)
	// Ensure we have some minimal caching and file guarantees
	if cache < 16 {
		cache = 16
	}
	if handles < 16 {
		handles = 16
	}
	logger.Infof("Allocated cache and file handles, cache: %v, handles: %v", cache, handles)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	return &RawDB{
		fn: file,
		db: db,
	}, nil
}

// Path returns the path to the database directory.
func (db *RawDB) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *RawDB) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *RawDB) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *RawDB) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return nil, store.ErrKeyNotFound
		}
		return nil, err
	}
	return dat, nil
}

// Delete deletes the key from the queue and database
func (db *RawDB) Delete(key []byte) error {
	err := db.db.Delete(key, nil)
	if err != nil && err == leveldb.ErrNotFound {
		return store.ErrKeyNotFound
	}
	return err
}

func (db *RawDB) Reference(key []byte) error {
	// NOOP
	return nil
}

func (db *RawDB) Dereference(key []byte) error {
	// NOOP
	return nil
}

func (db *RawDB) CountReference(key []byte) (int, error) {
	// NOOP
	return 0, nil
}

func (db *RawDB) NewIterator() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (db *RawDB) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (db *RawDB) Close() {
	db.db.Close()
}

func (db *RawDB) LDB() *leveldb.DB {
	return db.db
}

func (db *RawDB) NewBatch() database.Batch {
	return &rawdbBatch{db: db.db, b: new(leveldb.Batch)}
}

type rawdbBatch struct {
	db   *leveldb.DB
	b    *leveldb.Batch
	size int
}

func (b *rawdbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *rawdbBatch) Delete(key []byte) error {
	b.b.Delete(key)
	b.size += 1
	return nil
}

func (b *rawdbBatch) Reference(key []byte) error {
	// NOOP
	return nil
}

func (b *rawdbBatch) Dereference(key []byte) error {
	// NOOP
	return nil
}

func (b *rawdbBatch) Write() error {
	err := b.db.Write(b.b, nil)
	if err != nil {
		return err
	}

	b.Reset()

	return nil
}

func (b *rawdbBatch) ValueSize() int {
	return b.size
}

func (b *rawdbBatch) Reset() {
	b.b.Reset()
	b.size = 0
}
