package backend

import (
	"testing"
)

func newTestAerospikeDB() (*AerospikeDatabase, func()) {
	db, err := NewAerospikeDatabase()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
	}
}

func TestAerospikeDB_PutGet(t *testing.T) {
	db, close := newTestAerospikeDB()
	defer close()
	testPutGet(db, t)
}
