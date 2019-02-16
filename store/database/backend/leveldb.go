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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"github.com/thetatoken/theta/common/metrics"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "store"})

const (
	writePauseWarningThrottler = 1 * time.Minute
)

var OpenFileLimit = 64

type LDBDatabase struct {
	fn    string      // filename for reporting
	db    *leveldb.DB // LevelDB instance
	refdb *leveldb.DB // LevelDB instance for references

	compTimeMeter    metrics.Meter // Meter for measuring the total time spent in database compaction
	compReadMeter    metrics.Meter // Meter for measuring the data read during compaction
	compWriteMeter   metrics.Meter // Meter for measuring the data written during compaction
	writeDelayNMeter metrics.Meter // Meter for measuring the write delay number due to database compaction
	writeDelayMeter  metrics.Meter // Meter for measuring the write delay duration due to database compaction
	diskReadMeter    metrics.Meter // Meter for measuring the effective amount of data read
	diskWriteMeter   metrics.Meter // Meter for measuring the effective amount of data written

	quitLock sync.Mutex      // Mutex protecting the quit channel access
	quitChan chan chan error // Quit channel to stop the metrics collection before closing the database
}

// NewLDBDatabase returns a LevelDB wrapped object.
func NewLDBDatabase(file string, reffile string, cache int, handles int) (*LDBDatabase, error) {
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

	// Open the reference db and recover any potential corruptions
	refdb, err := leveldb.OpenFile(reffile, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache / 2 * opt.MiB,
		WriteBuffer:            cache / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		refdb, err = leveldb.RecoverFile(reffile, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		return nil, err
	}

	return &LDBDatabase{
		fn:    file,
		db:    db,
		refdb: refdb,
	}, nil
}

// Path returns the path to the database directory.
func (db *LDBDatabase) Path() string {
	return db.fn
}

// Put puts the given key / value to the queue
func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Has(key []byte) (bool, error) {
	return db.db.Has(key, nil)
}

// Get returns the given key if it's present.
func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
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
func (db *LDBDatabase) Delete(key []byte) error {
	db.refdb.Delete(key, nil)
	err := db.db.Delete(key, nil)
	if err != nil && err == leveldb.ErrNotFound {
		return store.ErrKeyNotFound
	}
	return err
}

func (db *LDBDatabase) Reference(key []byte) error {
	// check if k/v exists
	value, err := db.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return store.ErrKeyNotFound
		}
		return err
	}
	if value == nil {
		return store.ErrKeyNotFound
	}

	var ref int
	dat, err := db.refdb.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
		ref = 1
	} else {
		ref, err = strconv.Atoi(string(dat))
		if err != nil {
			return err
		}
		ref++
	}
	return db.refdb.Put(key, []byte(strconv.Itoa(ref)), nil)
}

func (db *LDBDatabase) Dereference(key []byte) error {
	// check if k/v exists
	value, err := db.Get(key)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return store.ErrKeyNotFound
		}
		return err
	}
	if value == nil {
		return store.ErrKeyNotFound
	}

	var ref int
	dat, err := db.refdb.Get(key, nil)
	if err != nil {
		if err != leveldb.ErrNotFound {
			return err
		}
	} else {
		ref, err = strconv.Atoi(string(dat))
		if err != nil {
			return err
		}
		if ref > 0 {
			return db.refdb.Put(key, []byte(strconv.Itoa(ref-1)), nil)
		}
	}
	return nil
}

func (db *LDBDatabase) CountReference(key []byte) (int, error) {
	// check if k/v exists
	// value, err := db.Get(key)
	// if err != nil {
	// 	if err == leveldb.ErrNotFound {
	// 		return 0, store.ErrKeyNotFound
	// 	}
	// 	return 0, err
	// }
	// if value == nil {
	// 	return 0, store.ErrKeyNotFound
	// }

	dat, err := db.refdb.Get(key, nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			return 0, store.ErrKeyNotFound
		}
		return 0, err
	}
	if dat == nil {
		return 0, nil
	}
	ref, err := strconv.Atoi(string(dat))
	if err != nil {
		return 0, err
	}
	return ref, nil
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}

// NewIteratorWithPrefix returns a iterator to iterate over subset of database content with a particular prefix.
func (db *LDBDatabase) NewIteratorWithPrefix(prefix []byte) iterator.Iterator {
	return db.db.NewIterator(util.BytesPrefix(prefix), nil)
}

func (db *LDBDatabase) Close() {
	// Stop the metrics collection to avoid internal database races
	db.quitLock.Lock()
	defer db.quitLock.Unlock()

	if db.quitChan != nil {
		errc := make(chan error)
		db.quitChan <- errc
		if err := <-errc; err != nil {
			logger.Errorf("Metrics collection failed, err: %v", err)
		}
		db.quitChan = nil
	}
	err := db.db.Close()
	err = db.refdb.Close()
	if err == nil {
		logger.Infof("Database closed")
	} else {
		logger.Errorf("Failed to close database, err: %v", err)
	}
}

func (db *LDBDatabase) LDB() *leveldb.DB {
	return db.db
}

// Meter configures the database metrics collectors and
func (db *LDBDatabase) Meter(prefix string) {
	if metrics.Enabled {
		// Initialize all the metrics collector at the requested prefix
		db.compTimeMeter = metrics.NewRegisteredMeter(prefix+"compact/time", nil)
		db.compReadMeter = metrics.NewRegisteredMeter(prefix+"compact/input", nil)
		db.compWriteMeter = metrics.NewRegisteredMeter(prefix+"compact/output", nil)
		db.diskReadMeter = metrics.NewRegisteredMeter(prefix+"disk/read", nil)
		db.diskWriteMeter = metrics.NewRegisteredMeter(prefix+"disk/write", nil)
	}
	// Initialize write delay metrics no matter we are in metric mode or not.
	db.writeDelayMeter = metrics.NewRegisteredMeter(prefix+"compact/writedelay/duration", nil)
	db.writeDelayNMeter = metrics.NewRegisteredMeter(prefix+"compact/writedelay/counter", nil)

	// Create a quit channel for the periodic collector and run it
	db.quitLock.Lock()
	db.quitChan = make(chan chan error)
	db.quitLock.Unlock()

	go db.meter(3 * time.Second)
}

// meter periodically retrieves internal leveldb counters and reports them to
// the metrics subsystem.
//
// This is how a stats table look like (currently):
//   Compactions
//    Level |   Tables   |    Size(MB)   |    Time(sec)  |    Read(MB)   |   Write(MB)
//   -------+------------+---------------+---------------+---------------+---------------
//      0   |          0 |       0.00000 |       1.27969 |       0.00000 |      12.31098
//      1   |         85 |     109.27913 |      28.09293 |     213.92493 |     214.26294
//      2   |        523 |    1000.37159 |       7.26059 |      66.86342 |      66.77884
//      3   |        570 |    1113.18458 |       0.00000 |       0.00000 |       0.00000
//
// This is how the write delay look like (currently):
// DelayN:5 Delay:406.604657ms Paused: false
//
// This is how the iostats look like (currently):
// Read(MB):3895.04860 Write(MB):3654.64712
func (db *LDBDatabase) meter(refresh time.Duration) {
	// Create the counters to store current and previous compaction values
	compactions := make([][]float64, 2)
	for i := 0; i < 2; i++ {
		compactions[i] = make([]float64, 3)
	}
	// Create storage for iostats.
	var iostats [2]float64

	// Create storage and warning log tracer for write delay.
	var (
		delaystats      [2]int64
		lastWritePaused time.Time
	)

	var (
		errc chan error
		merr error
	)

	// Iterate ad infinitum and collect the stats
	for i := 1; errc == nil && merr == nil; i++ {
		// Retrieve the database stats
		stats, err := db.db.GetProperty("leveldb.stats")
		if err != nil {
			logger.Errorf("Failed to read database stats, err: %v", err)
			merr = err
			continue
		}
		// Find the compaction table, skip the header
		lines := strings.Split(stats, "\n")
		for len(lines) > 0 && strings.TrimSpace(lines[0]) != "Compactions" {
			lines = lines[1:]
		}
		if len(lines) <= 3 {
			logger.Errorf("Compaction table not found")
			merr = errors.New("compaction table not found")
			continue
		}
		lines = lines[3:]

		// Iterate over all the table rows, and accumulate the entries
		for j := 0; j < len(compactions[i%2]); j++ {
			compactions[i%2][j] = 0
		}
		for _, line := range lines {
			parts := strings.Split(line, "|")
			if len(parts) != 6 {
				break
			}
			for idx, counter := range parts[3:] {
				value, err := strconv.ParseFloat(strings.TrimSpace(counter), 64)
				if err != nil {
					logger.Errorf("Compaction entry parsing failed, err: %v", err)
					merr = err
					continue
				}
				compactions[i%2][idx] += value
			}
		}
		// Update all the requested meters
		if db.compTimeMeter != nil {
			db.compTimeMeter.Mark(int64((compactions[i%2][0] - compactions[(i-1)%2][0]) * 1000 * 1000 * 1000))
		}
		if db.compReadMeter != nil {
			db.compReadMeter.Mark(int64((compactions[i%2][1] - compactions[(i-1)%2][1]) * 1024 * 1024))
		}
		if db.compWriteMeter != nil {
			db.compWriteMeter.Mark(int64((compactions[i%2][2] - compactions[(i-1)%2][2]) * 1024 * 1024))
		}

		// Retrieve the write delay statistic
		writedelay, err := db.db.GetProperty("leveldb.writedelay")
		if err != nil {
			logger.Errorf("Failed to read database write delay statistic, err: %v", err)
			merr = err
			continue
		}
		var (
			delayN        int64
			delayDuration string
			duration      time.Duration
			paused        bool
		)
		if n, err := fmt.Sscanf(writedelay, "DelayN:%d Delay:%s Paused:%t", &delayN, &delayDuration, &paused); n != 3 || err != nil {
			logger.Errorf("Write delay statistic not found")
			merr = err
			continue
		}
		duration, err = time.ParseDuration(delayDuration)
		if err != nil {
			logger.Errorf("Failed to parse delay duration, err: %v", err)
			merr = err
			continue
		}
		if db.writeDelayNMeter != nil {
			db.writeDelayNMeter.Mark(delayN - delaystats[0])
		}
		if db.writeDelayMeter != nil {
			db.writeDelayMeter.Mark(duration.Nanoseconds() - delaystats[1])
		}
		// If a warning that db is performing compaction has been displayed, any subsequent
		// warnings will be withheld for one minute not to overwhelm the user.
		if paused && delayN-delaystats[0] == 0 && duration.Nanoseconds()-delaystats[1] == 0 &&
			time.Now().After(lastWritePaused.Add(writePauseWarningThrottler)) {
			logger.Warnf("Database compacting, degraded performance")
			lastWritePaused = time.Now()
		}
		delaystats[0], delaystats[1] = delayN, duration.Nanoseconds()

		// Retrieve the database iostats.
		ioStats, err := db.db.GetProperty("leveldb.iostats")
		if err != nil {
			logger.Errorf("Failed to read database iostats, err: %v", err)
			merr = err
			continue
		}
		var nRead, nWrite float64
		parts := strings.Split(ioStats, " ")
		if len(parts) < 2 {
			logger.Errorf("Bad syntax of ioStats, ioStats: %v", ioStats)
			merr = fmt.Errorf("bad syntax of ioStats %s", ioStats)
			continue
		}
		if n, err := fmt.Sscanf(parts[0], "Read(MB):%f", &nRead); n != 1 || err != nil {
			logger.Errorf("Bad syntax of read entry, entry: %v", parts[0])
			merr = err
			continue
		}
		if n, err := fmt.Sscanf(parts[1], "Write(MB):%f", &nWrite); n != 1 || err != nil {
			logger.Errorf("Bad syntax of write entry, entry: %v", parts[1])
			merr = err
			continue
		}
		if db.diskReadMeter != nil {
			db.diskReadMeter.Mark(int64((nRead - iostats[0]) * 1024 * 1024))
		}
		if db.diskWriteMeter != nil {
			db.diskWriteMeter.Mark(int64((nWrite - iostats[1]) * 1024 * 1024))
		}
		iostats[0], iostats[1] = nRead, nWrite

		// Sleep a bit, then repeat the stats collection
		select {
		case errc = <-db.quitChan:
			// Quit requesting, stop hammering the database
		case <-time.After(refresh):
			// Timeout, gather a new set of stats
		}
	}

	if errc == nil {
		errc = <-db.quitChan
	}
	errc <- merr
}

func (db *LDBDatabase) NewBatch() database.Batch {
	return &ldbBatch{db: db.db, refdb: db.refdb, b: new(leveldb.Batch), references: make(map[string]int)}
}

type ldbBatch struct {
	db         *leveldb.DB
	refdb      *leveldb.DB
	b          *leveldb.Batch
	references map[string]int
	size       int
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	b.size += len(value)
	return nil
}

func (b *ldbBatch) Delete(key []byte) error {
	b.refdb.Delete(key, nil)
	b.b.Delete(key)
	b.size += 1
	return nil
}

func (b *ldbBatch) Reference(key []byte) error {
	b.references[string(key)]++
	b.size++
	return nil
}

func (b *ldbBatch) Dereference(key []byte) error {
	b.references[string(key)]--
	b.size++
	return nil
}

func (b *ldbBatch) Write() error {
	err := b.db.Write(b.b, nil)
	if err != nil {
		return err
	}

	for k, v := range b.references {
		if v == 0 {
			// refs and derefs canceled out
			delete(b.references, k)
		}
	}

	for k, v := range b.references {
		var ref int
		dat, err := b.refdb.Get([]byte(k), nil)
		if err != nil {
			if err != leveldb.ErrNotFound {
				return err
			}
			if v < 0 {
				continue
			}
			ref = v
		} else {
			ref, err = strconv.Atoi(string(dat))
			if err != nil {
				return err
			}
			if ref <= 0 && v < 0 {
				continue
			}
			ref = ref + v
			if ref < 0 {
				ref = 0
			}
		}
		err = b.refdb.Put([]byte(k), []byte(strconv.Itoa(ref)), nil)
		if err != nil {
			return err
		}
	}

	b.Reset()

	return nil
}

func (b *ldbBatch) ValueSize() int {
	return b.size
}

func (b *ldbBatch) Reset() {
	b.b.Reset()
	b.references = make(map[string]int)
	b.size = 0
}

type table struct {
	db     database.Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db database.Database, prefix string) database.Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Has(key []byte) (bool, error) {
	return dt.db.Has(append([]byte(dt.prefix), key...))
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Reference(key []byte) error {
	return dt.db.Reference(append([]byte(dt.prefix), key...))
}

func (dt *table) Dereference(key []byte) error {
	return dt.db.Dereference(append([]byte(dt.prefix), key...))
}

func (dt *table) CountReference(key []byte) (int, error) {
	return dt.db.CountReference(key)
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

type tableBatch struct {
	batch  database.Batch
	prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db database.Database, prefix string) database.Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() database.Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Delete(key []byte) error {
	return tb.batch.Delete(append([]byte(tb.prefix), key...))
}

func (tb *tableBatch) Reference(key []byte) error {
	return tb.batch.Reference(append([]byte(tb.prefix), key...))
}

func (tb *tableBatch) Dereference(key []byte) error {
	return tb.batch.Dereference(append([]byte(tb.prefix), key...))
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}

func (tb *tableBatch) ValueSize() int {
	return tb.batch.ValueSize()
}

func (tb *tableBatch) Reset() {
	tb.batch.Reset()
}
