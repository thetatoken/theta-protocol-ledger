package backend

import (
	"errors"
	"time"

	"github.com/aerospike/aerospike-client-go"
	"github.com/thetatoken/ukulele/store/database"
)

const (
	Host      string = "127.0.0.1"
	Port      int    = 3100
	Namespace string = "test"
	Set       string = "store"
	ValueBin  string = "value"
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
		return nil, errors.New("not found")
	}

	value := rec.Bins[ValueBin].([]byte)
	return value, nil
}

// Delete deletes the key from the database
func (db *AerospikeDatabase) Delete(key []byte) error {
	_, err := db.client.Delete(nil, getDBKey(key))
	return err
}

func (db *AerospikeDatabase) Close() {
	db.client.Close()
}

func (db *AerospikeDatabase) NewBatch() database.Batch {
	return &adbBatch{db: db, puts: []Document{}, deletes: []Document{}}
}

type adbBatch struct {
	db      *AerospikeDatabase
	puts    []Document
	deletes []Document
	size    int
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

func (b *adbBatch) Write() error {
	for _, doc := range b.puts {
		err := b.db.Put(doc.Key, doc.Value)
		if err != nil {
			return err
		}
	}
	for _, doc := range b.deletes {
		err := b.db.Delete(doc.Key)
		if err != nil {
			return err
		}
	}
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
