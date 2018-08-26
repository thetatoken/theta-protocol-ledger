package backend

import (
	"context"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/findopt"
	"github.com/mongodb/mongo-go-driver/mongo/updateopt"
	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/store/database"
)

const (
	Id         string = "_id"
	Value      string = "value"
	Database   string = "peer_service"
	Collection string = "peer"
)

type Document struct {
	Key   []byte `bson:"_id" json:"key"`
	Value []byte `bson:"value" json:"value"`
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
	return res.Next(nil), err
}

// Get returns the given key if it's present.
func (db *MongoDatabase) Get(key []byte) ([]byte, error) {
	result := new(Document)
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	err := db.collection.FindOne(nil, filter).Decode(result)
	return []byte(result.Value), err
}

// Delete deletes the key from the database
func (db *MongoDatabase) Delete(key []byte) error {
	filter := bson.NewDocument(bson.EC.Binary(Id, key))
	_, err := db.collection.DeleteMany(nil, filter)
	return err
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
	return nil
}
