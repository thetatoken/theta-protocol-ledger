package backend

import (
	"context"

	"github.com/thetatoken/theta/store"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/store/database"
)

const (
	Id         string = "_id"
	Value      string = "value"
	Reference  string = "ref"
	Database   string = "peer_service"
	Collection string = "peer"
)

type Document struct {
	Key       []byte `bson:"_id" json:"k,omitempty"`
	Value     []byte `bson:"value" json:"v"`
	Reference int    `bson:"ref" json:"ref,omitempty"`
}

// MongoDatabase a MongoDB wrapped object.
type MongoDatabase struct {
	client     *mongo.Client
	collection *mongo.Collection
}

// NewMongoDatabase returns a MongoDB wrapped object.
func NewMongoDatabase() (*MongoDatabase, error) {
	const ConnectionUri string = "mongodb://localhost:27017"

	client, err := mongo.NewClient(ConnectionUri)
	if err != nil {
		return nil, err
	}
	err = client.Connect(context.Background())
	if err != nil {
		return nil, err
	}

	db := client.Database(Database)
	collection := db.Collection(Collection)

	return &MongoDatabase{
		client:     client,
		collection: collection,
	}, nil
}

// Put puts the given key / value to the database
func (db *MongoDatabase) Put(key []byte, value []byte) error {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	document := Document{Key: key, Value: value}
	updator := map[string]Document{"$set": document}
	option := updateopt.Upsert(true)
	_, err := db.collection.UpdateOne(nil, filter, updator, option)
	if err != nil {
		return err
	}
	return nil
}

// Has checks if the given key is present in the database
func (db *MongoDatabase) Has(key []byte) (bool, error) {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	option := findopt.Limit(1)
	res, err := db.collection.Find(nil, filter, option)
	if res == nil {
		return false, err
	}
	return res.Next(nil), err
}

// Get returns the given key if it's present.
func (db *MongoDatabase) Get(key []byte) ([]byte, error) {
	result := new(Document)
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	err := db.collection.FindOne(nil, filter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, store.ErrKeyNotFound
		}
		return nil, err
	}
	return result.Value, err
}

// Delete deletes the key from the database
func (db *MongoDatabase) Delete(key []byte) error {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	_, err := db.collection.DeleteOne(nil, filter)
	if err != nil && err == mongo.ErrNoDocuments {
		return store.ErrKeyNotFound
	}
	return err
}

func (db *MongoDatabase) Reference(key []byte) error {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	option := updateopt.Upsert(false)
	updator := map[string]map[string]int{"$inc": {Reference: 1}}
	res, err := db.collection.UpdateOne(nil, filter, updator, option)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return store.ErrKeyNotFound
	}
	return nil
}

func (db *MongoDatabase) Dereference(key []byte) error {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	result := new(Document)
	err := db.collection.FindOne(nil, filter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return store.ErrKeyNotFound
		}
		return err
	}

	if result.Reference > 0 {
		option := updateopt.Upsert(false)
		updator := map[string]map[string]int{"$inc": {Reference: -1}}
		res, err := db.collection.UpdateOne(nil, filter, updator, option)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return store.ErrKeyNotFound
		}
	}
	return nil
}

func (db *MongoDatabase) CountReference(key []byte) (int, error) {
	result := new(Document)
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	err := db.collection.FindOne(nil, filter).Decode(result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 0, store.ErrKeyNotFound
		}
		return 0, nil
	}
	return result.Reference, err
}

func (db *MongoDatabase) Close() {
	err := db.client.Disconnect(context.Background())
	if err == nil {
		log.Infof("Database closed")
	} else {
		log.Errorf("Failed to close database, err: %v", err)
	}
}

func (db *MongoDatabase) NewBatch() database.Batch {
	return &mdbBatch{db: db, collection: db.collection, references: make(map[string]int)}
}

type mdbBatch struct {
	db         *MongoDatabase
	collection *mongo.Collection
	puts       []Document
	deletes    []*bson.Value
	references map[string]int
	size       int
}

func (b *mdbBatch) Put(key, value []byte) error {
	b.puts = append(b.puts, Document{Key: key, Value: value})
	b.size += len(value)
	return nil
}

func (b *mdbBatch) Delete(key []byte) error {
	b.deletes = append(b.deletes, bson.VC.Binary(key))
	b.size++
	return nil
}

func (b *mdbBatch) Reference(key []byte) error {
	b.references[string(key)]++
	b.size++
	return nil
}

func (b *mdbBatch) Dereference(key []byte) error {
	b.references[string(key)]--
	b.size++
	return nil
}

func (b *mdbBatch) Write() error {
	for i := range b.puts {
		doc := b.puts[i]
		b.db.Put(doc.Key, doc.Value)
	}

	filter := bson.NewDocument(bson.EC.SubDocumentFromElements(Id, bson.EC.ArrayFromElements("$in", b.deletes...)))
	_, err := b.collection.DeleteMany(nil, filter)

	for k, v := range b.references {
		if v == 0 {
			// refs and derefs canceled out
			delete(b.references, k)
		}
	}

	for k, v := range b.references {
		filter := bson.NewDocument(bson.EC.Binary(Id, []byte(k)))
		result := new(Document)
		err := b.collection.FindOne(nil, filter).Decode(result)
		if err != nil {
			if err != mongo.ErrNoDocuments {
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
		option := updateopt.Upsert(false)
		updator := map[string]map[string]int{"$set": {Reference: ref}}
		res, err := b.collection.UpdateOne(nil, filter, updator, option)
		if err != nil {
			return err
		}
		if res.MatchedCount == 0 {
			return store.ErrKeyNotFound
		}
	}

	b.Reset()

	return err
}

func (b *mdbBatch) ValueSize() int {
	return b.size
}

func (b *mdbBatch) Reset() {
	b.puts = nil
	b.deletes = nil
	b.references = make(map[string]int)
	b.size = 0
}
