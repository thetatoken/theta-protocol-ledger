// +build cluster_deployment

package backend

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/thetatoken/theta/store/database"
)

func newTestBDB() (*BadgerDatabase, database.Batch, func()) {
	dirname, err := ioutil.TempDir(os.TempDir(), "db_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	db, err := NewBadgerDatabase(dirname)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	batch := db.NewBatch()

	return db, batch, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}

func TestBadgerDB_PutGet(t *testing.T) {
	db, batch, close := newTestBDB()
	defer close()
	testPutGet(db, batch, t)
}
