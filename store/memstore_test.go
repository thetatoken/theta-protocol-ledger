// +build unit

package store

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemStore(t *testing.T) {
	assert := assert.New(t)

	memstore := NewMemKVStore()

	key, _ := hex.DecodeString("a0")
	memstore.Put(key, "hello!")

	var val string
	err := memstore.Get(key, &val)
	assert.Nil(err)
	assert.Equal("hello!", val)

	memstore.Delete(key)
	var val2 string
	err = memstore.Get(key, &val2)
	assert.NotNil(err)
}
