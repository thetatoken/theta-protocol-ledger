// +build cluster_deployment

package backend

import (
	"testing"

	"github.com/thetatoken/theta/store/database"
)

func newTestMgoDB() (*MgoDatabase, database.Batch, func()) {
	db, err := NewMgoDatabase()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	batch := db.NewBatch()

	return db, batch, func() {
		db.Close()
	}
}

func TestMgoDB_PutGet(t *testing.T) {
	db, batch, close := newTestMgoDB()
	defer close()
	testPutGet(db, batch, t)
}
