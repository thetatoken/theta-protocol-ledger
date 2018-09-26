package backend

import (
	"time"

	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/database"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// MgoDatabase a MongoDB (using mgo driver) wrapped object.
type MgoDatabase struct {
	session    *mgo.Session
	collection *mgo.Collection
}

// NewMgoDatabase returns a MongoDB (using mgo driver) wrapped object.
func NewMgoDatabase() (*MgoDatabase, error) {
	const ConnectionUri string = "localhost:27017"

	Host := []string{
		ConnectionUri,
	}

	session, err := mgo.DialWithInfo(&mgo.DialInfo{
		Addrs:    Host,
		Timeout:  100 * time.Millisecond,
		Database: Database,
	})
	if err != nil {
		panic(err)
	}

	collection := session.DB(Database).C(Collection)

	return &MgoDatabase{
		session:    session,
		collection: collection,
	}, nil
}

// Put puts the given key / value to the database
func (db *MgoDatabase) Put(key []byte, value []byte) error {
	selector := bson.M{Id: key}
	update := bson.M{"$set": bson.M{Value: value}}
	_, err := db.collection.Upsert(selector, update)
	return err
}

// Has checks if the given key is present in the database
func (db *MgoDatabase) Has(key []byte) (bool, error) {
	res := db.collection.Find(bson.M{Id: key}).Limit(1)
	cnt, err := res.Count()
	return cnt > 0, err
}

// Get returns the given key if it's present.
func (db *MgoDatabase) Get(key []byte) ([]byte, error) {
	result := new(Document)
	err := db.collection.Find(bson.M{Id: key}).One(&result)
	if err == mgo.ErrNotFound {
		return nil, store.ErrKeyNotFound
	}
	return []byte(result.Value), nil
}

// Delete deletes the key from the database
func (db *MgoDatabase) Delete(key []byte) error {
	err := db.collection.Remove(bson.M{Id: key})
	return err
}

func (db *MgoDatabase) Reference(key []byte) error {
	// TODO
	return nil
}

func (db *MgoDatabase) Dereference(key []byte) error {
	// TODO
	return nil
}

func (db *MgoDatabase) CountReference(key []byte) (int, error) {
	return 0, nil
}

func (db *MgoDatabase) Close() {
	db.session.Close()
}

func (db *MgoDatabase) NewBatch() database.Batch {
	batch := &mgodbBatch{collection: db.collection, b: db.collection.Bulk()}
	batch.b.Unordered()
	return batch
}

type mgodbBatch struct {
	collection *mgo.Collection
	b          *mgo.Bulk
	size       int
}

func (b *mgodbBatch) Put(key, value []byte) error {
	selector := bson.M{Id: key}
	update := bson.M{"$set": bson.M{Value: value}}
	b.b.Upsert(selector, update)
	b.size += len(value)
	return nil
}

func (b *mgodbBatch) Delete(key []byte) error {
	selector := bson.M{Id: key}
	b.b.Remove(selector)
	b.size++
	return nil
}

func (b *mgodbBatch) Reference(key []byte) error {
	// TODO
	return nil
}

func (b *mgodbBatch) Dereference(key []byte) error {
	// TODO
	return nil
}

func (b *mgodbBatch) Write() error {
	_, err := b.b.Run()
	b.Reset()
	return err
}

func (b *mgodbBatch) ValueSize() int {
	return b.size
}

func (b *mgodbBatch) Reset() {
	b.b = b.collection.Bulk()
	b.b.Unordered()
	b.size = 0
}
