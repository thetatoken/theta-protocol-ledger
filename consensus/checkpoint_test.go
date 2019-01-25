package consensus

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/store/database/backend"
)

func TestGenerateGenesis(t *testing.T) {
	assert := assert.New(t)

	genesis, err := generateGenesisCheckpoint()
	assert.Nil(err)

	db := backend.NewMemDatabase()
	LoadCheckpointLedgerState(genesis, db)

	// Should be able to load tree rooting at state hash from db.
	s := state.NewStoreView(0, genesis.FirstBlock.StateHash, db)
	assert.NotNil(s)

	filePath := "tmp.bin"
	err = WriteGenesisCheckpoint(filePath)
	assert.Nil(err)

	genesis2, err := LoadCheckpoint(filePath)
	assert.Nil(err)

	assert.Equal(genesis.FirstBlock.Hash(), genesis2.FirstBlock.Hash())
	assert.Equal(len(genesis.LedgerState), len(genesis2.LedgerState))

	os.Remove(filePath)
	os.Remove(filePath + ".bak")
}
