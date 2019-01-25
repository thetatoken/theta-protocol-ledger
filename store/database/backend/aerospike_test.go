// +build cluster_deployment

package backend

import (
	"testing"

	"github.com/thetatoken/theta/store/database"
)

func newTestAerospikeDB() (*AerospikeDatabase, database.Batch, func()) {
	db, err := NewAerospikeDatabase()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	batch := db.NewBatch()

	return db, batch, func() {
		db.Close()
	}
}

func TestAerospikeDB_PutGet(t *testing.T) {
	db, batch, close := newTestAerospikeDB()
	defer close()
	testPutGet(db, batch, t)
}
