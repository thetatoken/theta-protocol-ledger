package backend

func newTestMDB() (*MongoDatabase, func()) {
	db, err := NewMongoDatabase()
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
	}
}

/*
func TestMDB_PutGet(t *testing.T) {
	db, close := newTestMDB()
	defer close()
	testPutGet(db, t)
}
*/
