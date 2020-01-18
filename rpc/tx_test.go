package rpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
)

func TestTxCallbackManager(t *testing.T) {
	assert := assert.New(t)

	m := NewTxCallbackManager()

	cbExcuted := []bool{false, false}

	h1 := common.HexToHash("a0")
	cb1 := func(*core.Block) {
		cbExcuted[0] = true
	}

	h2 := common.HexToHash("b1")
	cb2 := func(*core.Block) {
		cbExcuted[1] = true
	}

	assert.Equal(0, len(m.txHashToCallback))

	m.AddCallback(h1, cb1)
	assert.Equal(1, len(m.txHashToCallback))

	cb, exists := m.RemoveCallback(h1)
	assert.Equal(0, len(m.txHashToCallback))
	assert.True(exists)
	cb.Callback(nil)
	assert.True(cbExcuted[0])

	cb, exists = m.RemoveCallback(h2)
	assert.False(exists)

	assert.Equal(0, len(m.txHashToCallback))

	m = NewTxCallbackManager()
	m.AddCallback(h1, cb1)
	m.AddCallback(h2, cb2)

	assert.Equal(2, len(m.txHashToCallback))
	m.Trim()
	assert.Equal(2, len(m.txHashToCallback))

	m.callbacks[0].created = time.Now().Add(-txTimeout).Add(-1 * time.Second)
	m.Trim()
	assert.Equal(1, len(m.txHashToCallback))
	assert.Equal(1, len(m.callbacks))
}
