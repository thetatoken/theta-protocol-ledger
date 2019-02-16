package backend

import (
	"time"

	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
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
	return result.Value, nil
}

// Delete deletes the key from the database
func (db *MgoDatabase) Delete(key []byte) error {
	err := db.collection.Remove(bson.M{Id: key})
	if err != nil && err == mgo.ErrNotFound {
		return store.ErrKeyNotFound
	}
	return err
}

func (db *MgoDatabase) Reference(key []byte) error {
	selector := bson.M{Id: key}
	incr := bson.M{"$inc": bson.M{Reference: 1}}
	err := db.collection.Update(selector, incr)
	if err != nil && err == mgo.ErrNotFound {
		return store.ErrKeyNotFound
	}
	return err
}

func (db *MgoDatabase) Dereference(key []byte) error {
	selector := bson.M{Id: key}
	result := new(Document)
	err := db.collection.Find(selector).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return store.ErrKeyNotFound
		}
		return err
	}

	if result.Reference > 0 {
		decr := bson.M{"$inc": bson.M{Reference: -1}}
		err := db.collection.Update(selector, decr)
		return err
	}

	return nil
}

func (db *MgoDatabase) CountReference(key []byte) (int, error) {
	result := new(Document)
	err := db.collection.Find(bson.M{Id: key}).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return 0, store.ErrKeyNotFound
		}
		return 0, err
	}
	return result.Reference, nil
}

func (db *MgoDatabase) Close() {
	db.session.Close()
}

func (db *MgoDatabase) NewBatch() database.Batch {
	batch := &mgodbBatch{collection: db.collection, b: db.collection.Bulk(), references: make(map[string]int)}
	batch.b.Unordered()
	return batch
}

type mgodbBatch struct {
	collection *mgo.Collection
	b          *mgo.Bulk
	references map[string]int
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
	// selector := bson.M{Id: key}
	// incr := bson.M{"$inc": bson.M{Reference: 1}}
	// b.b.Upsert(selector, incr)
	b.references[string(key)]++
	b.size++
	return nil
}

func (b *mgodbBatch) Dereference(key []byte) error {
	// selector := bson.M{Id: key}
	// decr := bson.M{"$inc": bson.M{Reference: -1}}
	// b.b.Upsert(selector, decr)
	b.references[string(key)]--
	b.size++
	return nil
}

func (b *mgodbBatch) Write() error {
	_, err := b.b.Run()

	for k, v := range b.references {
		if v == 0 {
			// refs and derefs canceled out
			delete(b.references, k)
		}
	}

	if len(b.references) > 0 {
		b.b = b.collection.Bulk()
		b.b.Unordered()

		for k, v := range b.references {
			selector := bson.M{Id: []byte(k)}
			result := new(Document)
			err := b.collection.Find(selector).One(&result)
			if err != nil {
				if err != mgo.ErrNotFound {
					return err
				}
			}

			if result.Reference <= 0 && v < 0 {
				continue
			}

			ref := result.Reference + v
			if ref < 0 {
				ref = 0
			}

			update := bson.M{"$set": bson.M{Reference: ref}}
			b.b.Upsert(selector, update)
		}
		_, err = b.b.Run()
	}

	b.Reset()
	return err
}

func (b *mgodbBatch) ValueSize() int {
	return b.size
}

func (b *mgodbBatch) Reset() {
	b.b = b.collection.Bulk()
	b.b.Unordered()
	b.references = make(map[string]int)
	b.size = 0
}
