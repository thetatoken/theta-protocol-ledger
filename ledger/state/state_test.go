package state

/*
import (
	"bytes"
	"testing"

	"github.com/thetatoken/ukulele/ledger/types"

	"github.com/stretchr/testify/assert"
)

func TestLedgerState(t *testing.T) {
	assert := assert.New(t)

	//States and Stores for tests
	store := NewMemTreeKVStore()
	state := NewState(store)

	//Account and address for tests
	dumAddr := []byte("dummyAddress")

	acc := new(types.Account)
	acc.Sequence = 1

	//reset the store/state/cache
	reset := func() {
		store := NewMemTreeKVStore()
		state = NewState(store)
	}

	//key value pairs to be tested within the system
	keyvalue := []struct {
		key   string
		value string
	}{
		{"foo", "snake"},
		{"bar", "mouse"},
	}

	//set the kvc to have all the key value pairs
	setRecords := func(kv types.KVStore) {
		for _, n := range keyvalue {
			kv.Set([]byte(n.key), []byte(n.value))
		}
	}

	//store has all the key value pairs
	storeHasAll := func(kv types.KVStore) bool {
		for _, n := range keyvalue {
			if !bytes.Equal(kv.Get([]byte(n.key)), []byte(n.value)) {
				return false
			}
		}
		return true
	}

	//test chainID
	state.SetChainID("testchain")
	assert.Equal(state.GetChainID(), "testchain", "ChainID is improperly stored")

	//test basic retrieve
	setRecords(store)
	assert.True(storeHasAll(store), "store doesn't retrieve after Set")

	// Test account retrieve
	state.SetAccount(dumAddr, acc)
	assert.Equal(state.GetAccount(dumAddr).Sequence, 1, "GetAccount not retrieving")

	// Test Checked()
	reset()
	checked := state.Checked()
	setRecords(checked)
	assert.True(storeHasAll(checked), "state.Checked() is not updated after Set")
	assert.False(storeHasAll(state.Delivered()), "state.Deliverred() is updated after Set")

	// Test Commit()
	reset()
	setRecords(state.Delivered())
	assert.True(storeHasAll(state.Delivered()), "state.Deliverred() is not updated after Set")
	assert.False(storeHasAll(state.Checked()), "state.Checked() is not updated after Set")
	assert.False(storeHasAll(state.Committed()), "state.Committed is updated after Set")
	state.Commit()
	assert.True(storeHasAll(state.Delivered()), "state.Deliverred() is not updated after Set")
	assert.True(storeHasAll(state.Checked()), "state.Deliverred() is not updated after Set")
	assert.True(storeHasAll(state.Committed()), "state.Committed is not updated after Set")
}
*/
