package backend

import (
	"encoding/json"

	"github.com/dgraph-io/badger"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
)

// BadgerDatabase a MongoDB (using badger driver) wrapped object.
type BadgerDatabase struct {
	db *badger.DB
}

// NewBadgerDatabase returns a BadgerDB wrapped object.
func NewBadgerDatabase(dirname string) (*BadgerDatabase, error) {
	opts := badger.DefaultOptions(dirname)
	opts.Dir = dirname
	opts.ValueDir = dirname
	db, err := badger.Open(opts)
	if err != nil {
		panic(err)
	}

	return &BadgerDatabase{
		db: db,
	}, nil
}

// Put puts the given key / value to the database
func (db *BadgerDatabase) Put(key []byte, value []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		document := Document{Value: value}
		marshal, err := json.Marshal(document)
		if err != nil {
			return err
		}
		return txn.Set(key, marshal)
	})
}

// Has checks if the given key is present in the database
func (db *BadgerDatabase) Has(key []byte) (bool, error) {
	err := db.db.View(func(txn *badger.Txn) error {
		_, err := txn.Get(key)
		return err
	})
	if err != nil {
		if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Get returns the given key if it's present.
func (db *BadgerDatabase) Get(key []byte) ([]byte, error) {
	var document Document
	err := db.db.View(func(txn *badger.Txn) error {
		unmarshal, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
				return store.ErrKeyNotFound
			}
			return err
		}

		return unmarshal.Value(func(val []byte) error {
			return json.Unmarshal(val, &document)
		})
	})
	return document.Value, err
}

// Delete deletes the key from the database
func (db *BadgerDatabase) Delete(key []byte) error {
	err := db.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(key)
	})
	if err != nil {
		if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
			return store.ErrKeyNotFound
		}
	}
	return err
}

func (db *BadgerDatabase) Reference(key []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		unmarshal, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
				return store.ErrKeyNotFound
			}
			return err
		}

		var document Document
		err = unmarshal.Value(func(val []byte) error {
			return json.Unmarshal(val, &document)
		})
		if err != nil {
			return err
		}

		document.Reference++
		marshal, err := json.Marshal(document)
		if err != nil {
			return err
		}
		return txn.Set(key, marshal)
	})
}

func (db *BadgerDatabase) Dereference(key []byte) error {
	return db.db.Update(func(txn *badger.Txn) error {
		unmarshal, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
				return store.ErrKeyNotFound
			}
			return err
		}

		var document Document
		err = unmarshal.Value(func(val []byte) error {
			return json.Unmarshal(val, &document)
		})
		if err != nil {
			return err
		}

		if document.Reference > 0 {
			document.Reference--
			marshal, err := json.Marshal(document)
			if err != nil {
				return err
			}
			return txn.Set(key, marshal)
		}
		return nil
	})
}

func (db *BadgerDatabase) CountReference(key []byte) (int, error) {
	var document Document
	err := db.db.View(func(txn *badger.Txn) error {
		unmarshal, err := txn.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound || err == badger.ErrEmptyKey {
				return store.ErrKeyNotFound
			}
			return err
		}
		return unmarshal.Value(func(val []byte) error {
			return json.Unmarshal(val, &document)
		})
	})
	if err != nil {
		return 0, err
	}
	return document.Reference, nil
}

func (db *BadgerDatabase) Close() {
	db.db.Close()
}

func (db *BadgerDatabase) NewBatch() database.Batch {
	batch := &badgerdbBatch{db: db.db, references: make(map[string]int)}

	return batch
}

type badgerdbBatch struct {
	db         *badger.DB
	puts       []Document
	deletes    []Document
	references map[string]int
	size       int
}

func (b *badgerdbBatch) Put(key, value []byte) error {
	b.puts = append(b.puts, Document{Key: key, Value: value})
	b.size += len(value)
	return nil
}

func (b *badgerdbBatch) Delete(key []byte) error {
	b.deletes = append(b.deletes, Document{Key: key})
	b.size++
	return nil
}

func (b *badgerdbBatch) Reference(key []byte) error {
	b.references[string(key)]++
	b.size++
	return nil
}

func (b *badgerdbBatch) Dereference(key []byte) error {
	b.references[string(key)]--
	b.size++
	return nil
}

func (b *badgerdbBatch) Write() error {
	txn := b.db.NewTransaction(true)
	for i := range b.puts {
		doc := b.puts[i]
		marshal, err := json.Marshal(Document{Value: doc.Value})
		if err != nil {
			return err
		}
		err = txn.Set(doc.Key, marshal)
		if err != nil {
			if err == badger.ErrTxnTooBig {
				if err := txn.Commit(); err != nil {
					return err
				}
				txn = b.db.NewTransaction(true)
				if err = txn.Set(doc.Key, marshal); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	for i := range b.deletes {
		doc := b.deletes[i]
		err := txn.Delete(doc.Key)
		if err != nil {
			if err == badger.ErrTxnTooBig {
				if err := txn.Commit(); err != nil {
					return err
				}
				txn = b.db.NewTransaction(true)
				if err = txn.Delete(doc.Key); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	for k, v := range b.references {
		if v == 0 {
			// refs and derefs canceled out
			delete(b.references, k)
		}
	}

	for k, v := range b.references {
		var document Document
		unmarshal, err := txn.Get([]byte(k))
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
		} else {
			err = unmarshal.Value(func(val []byte) error {
				return json.Unmarshal(val, &document)
			})
			if err != nil {
				return err
			}
		}

		if document.Reference <= 0 && v < 0 {
			continue
		}
		document.Reference += v
		if document.Reference < 0 {
			document.Reference = 0
		}
		marshal, err := json.Marshal(document)
		if err != nil {
			return err
		}

		err = txn.Set([]byte(k), marshal)
		if err != nil {
			if err == badger.ErrTxnTooBig {
				if err := txn.Commit(); err != nil {
					return err
				}
				txn = b.db.NewTransaction(true)
				if err = txn.Set([]byte(k), marshal); err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	if err := txn.Commit(); err != nil {
		return err
	}

	b.Reset()
	return nil
}

func (b *badgerdbBatch) ValueSize() int {
	return b.size
}

func (b *badgerdbBatch) Reset() {
	b.puts = nil
	b.deletes = nil
	b.references = make(map[string]int)
	b.size = 0
}
