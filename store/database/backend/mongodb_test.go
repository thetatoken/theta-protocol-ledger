package backend

// import (
// 	"testing"

// 	"github.com/thetatoken/ukulele/store/database"
// )

// func newTestMDB() (*MongoDatabase, database.Batch, func()) {
// 	db, err := NewMongoDatabase()
// 	if err != nil {
// 		panic("failed to create test database: " + err.Error())
// 	}

// 	batch := db.NewBatch()

// 	return db, batch, func() {
// 		db.Close()
// 	}
// }

// func TestMDB_PutGet(t *testing.T) {
// 	db, batch, close := newTestMDB()
// 	defer close()
// 	testPutGet(db, batch, t)
// }
