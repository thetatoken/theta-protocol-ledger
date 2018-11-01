package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
)

func TestTxIndex(t *testing.T) {
	assert := assert.New(t)

	tx1 := common.Bytes("tx1")
	tx2 := common.Bytes("tx2")
	tx3 := common.Bytes("tx3")
	tx4 := common.Bytes("tx4")
	block1 := core.NewBlock()
	block1.ChainID = "testchain"
	hash := common.BytesToHash(common.Bytes("block1"))
	block1.Hash = hash[:]
	block1.Height = 10
	block1.Txs = []common.Bytes{tx1, tx2, tx3}

	chain := CreateTestChain()
	chain.AddBlock(block1)

	for _, t := range block1.Txs {
		tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(t))
		assert.True(found)
		assert.NotNil(tx)
		assert.Equal(t, tx)
		assert.NotNil(block)
		assert.Equal(block.Hash, block1.Hash)
	}

	tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(tx4))
	assert.False(found)
	assert.Nil(tx)
	assert.Nil(block)
}
