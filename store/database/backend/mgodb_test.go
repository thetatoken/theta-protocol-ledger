package backend

import (
	"testing"
)

func newTestMgoDB() (*MgoDatabase, func()) {
	db, err := NewMgoDatabase()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
	}
}

func TestMgoDB_PutGet(t *testing.T) {
	db, close := newTestMgoDB()
	defer close()
	testPutGet(db, t)
}
