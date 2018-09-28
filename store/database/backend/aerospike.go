package backend

import (
	"time"

	"github.com/aerospike/aerospike-client-go"
	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/database"
)

const (
	Host      string = "127.0.0.1"
	Port      int    = 3100
	Namespace string = "test"
	Set       string = "store"
	ValueBin  string = "value"
	RefBin    string = "ref"
)

// AerospikeDatabase a MongoDB wrapped object.
type AerospikeDatabase struct {
	client *aerospike.Client
}

func getDBKey(key []byte) *aerospike.Key {
	askey, _ := aerospike.NewKey(Namespace, Set, key)
	return askey
}

// NewAerospikeDatabase returns a AerospikeDatabase wrapped object.
func NewAerospikeDatabase() (*AerospikeDatabase, error) {
	hosts := []*aerospike.Host{
		aerospike.NewHost(Host, Port),
	}

	client, err := aerospike.NewClientWithPolicyAndHost(nil, hosts...)
	if err != nil {
		return nil, err
	}

	return &AerospikeDatabase{
		client: client,
	}, nil
}

// Put puts the given key / value to the database
func (db *AerospikeDatabase) Put(key []byte, value []byte) error {
	bin := aerospike.NewBin(ValueBin, value)
	writePolicy := aerospike.NewWritePolicy(0, 0)
	writePolicy.Timeout = 300 * time.Millisecond
	err := db.client.PutBins(writePolicy, getDBKey(key), bin)
	return err
}

// Has checks if the given key is present in the database
func (db *AerospikeDatabase) Has(key []byte) (bool, error) {
	return db.client.Exists(nil, getDBKey(key))
}

// Get returns the given key if it's present.
func (db *AerospikeDatabase) Get(key []byte) ([]byte, error) {
	rec, err := db.client.Get(nil, getDBKey(key), ValueBin)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, store.ErrKeyNotFound
	}

	value := rec.Bins[ValueBin].([]byte)
	return value, nil
}

// Delete deletes the key from the database
func (db *AerospikeDatabase) Delete(key []byte) error {
	_, err := db.client.Delete(nil, getDBKey(key))
	return err
}

func (db *AerospikeDatabase) Reference(key []byte) error {
	rec, err := db.client.Get(nil, getDBKey(key), RefBin)
	if err != nil {
		return err
	}
	if rec == nil {
		return store.ErrKeyNotFound
	}

	var ref int
	if rec.Bins[RefBin] == nil {
		ref = 1
	} else {
		ref = rec.Bins[RefBin].(int) + 1
	}

	bin := aerospike.NewBin(RefBin, ref)
	writePolicy := aerospike.NewWritePolicy(0, 0)
	writePolicy.Timeout = 300 * time.Millisecond
	err = db.client.PutBins(writePolicy, getDBKey(key), bin)
	return err
}

func (db *AerospikeDatabase) Dereference(key []byte) error {
	rec, err := db.client.Get(nil, getDBKey(key), RefBin)
	if err != nil {
		return err
	}
	if rec == nil {
		return store.ErrKeyNotFound
	}

	if rec.Bins[RefBin] != nil {
		ref := rec.Bins[RefBin].(int)
		if ref > 0 {
			bin := aerospike.NewBin(RefBin, ref-1)
			writePolicy := aerospike.NewWritePolicy(0, 0)
			writePolicy.Timeout = 300 * time.Millisecond
			err = db.client.PutBins(writePolicy, getDBKey(key), bin)
			return err
		}
	}
	return nil
}

func (db *AerospikeDatabase) CountReference(key []byte) (int, error) {
	rec, err := db.client.Get(nil, getDBKey(key), RefBin)
	if err != nil {
		return 0, err
	}
	if rec == nil {
		return 0, store.ErrKeyNotFound
	}

	var ref int
	if rec.Bins[RefBin] == nil {
		ref = 0
	} else {
		ref = rec.Bins[RefBin].(int)
	}
	return ref, nil
}

func (db *AerospikeDatabase) Close() {
	db.client.Close()
}

func (db *AerospikeDatabase) NewBatch() database.Batch {
	return &adbBatch{db: db}
}

type adbBatch struct {
	db           *AerospikeDatabase
	puts         []Document
	deletes      []Document
	references   []Document
	dereferences []Document
	size         int
}

func (b *adbBatch) Put(key, value []byte) error {
	b.puts = append(b.puts, Document{Key: key, Value: value})
	b.size += len(value)
	return nil
}

func (b *adbBatch) Delete(key []byte) error {
	b.deletes = append(b.deletes, Document{Key: key})
	b.size++
	return nil
}

func (b *adbBatch) Reference(key []byte) error {
	b.references = append(b.references, Document{Key: key})
	b.size++
	return nil
}

func (b *adbBatch) Dereference(key []byte) error {
	b.dereferences = append(b.dereferences, Document{Key: key})
	b.size++
	return nil
}

func (b *adbBatch) Write() error {
	for i := range b.puts {
		doc := b.puts[i]
		b.db.Put(doc.Key, doc.Value)
	}

	for i := range b.deletes {
		b.db.Delete(b.deletes[i].Key)
	}

	for i := range b.references {
		b.db.Reference(b.references[i].Key)
	}

	for i := range b.dereferences {
		b.db.Dereference(b.dereferences[i].Key)
	}

	b.Reset()

	return nil
}

func (b *adbBatch) ValueSize() int {
	return b.size
}

func (b *adbBatch) Reset() {
	b.puts = nil
	b.deletes = nil
	b.size = 0
}
