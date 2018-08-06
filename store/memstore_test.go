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

	val, err := memstore.Get(key)
	assert.Nil(err)
	assert.Equal("hello!", val)

	memstore.Delete(key)
	val, err = memstore.Get(key)
	assert.NotNil(err)
}
