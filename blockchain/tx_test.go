package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
)

func TestTxIndex(t *testing.T) {
	assert := assert.New(t)

	tx1 := common.Bytes("tx1")
	tx2 := common.Bytes("tx2")
	tx3 := common.Bytes("tx3")
	tx4 := common.Bytes("tx4")
	block1 := core.CreateTestBlock("b1", "")
	block1.Height = 10
	block1.Txs = []common.Bytes{tx1, tx2, tx3}
	block1.UpdateHash()

	chain := CreateTestChain()
	chain.AddBlock(block1)

	for _, t := range block1.Txs {
		tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(t))
		assert.True(found)
		assert.NotNil(tx)
		assert.Equal(t, tx)
		assert.NotNil(block)
		assert.Equal(block.Hash(), block1.Hash())
	}

	tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(tx4))
	assert.False(found)
	assert.Nil(tx)
	assert.Nil(block)
}

func TestTxIndexDuplicateTx(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	tx1 := common.Bytes("tx1")
	tx2 := common.Bytes("tx2")
	tx3 := common.Bytes("tx3")

	core.ResetTestBlocks()
	chain := CreateTestChain()

	block1 := core.CreateTestBlock("b1", "")
	block1.Height = 10
	block1.Txs = []common.Bytes{tx1, tx2}

	block2 := core.CreateTestBlock("b2", "")
	block2.Height = 20
	block2.Txs = []common.Bytes{tx2, tx3}

	_, err := chain.AddBlock(block1)
	require.Nil(err)

	_, err = chain.AddBlock(block2)
	require.Nil(err)

	tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(tx1))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx1, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block1.Hash())

	// Tx2 should be linked with block1 instead of block2.
	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx2))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx2, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block1.Hash())

	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx3))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx3, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block2.Hash())

	// Tx2 should be linked with block2 after force insert.
	eb := &core.ExtendedBlock{Block: block2}
	chain.AddTxsToIndex(eb, true)
	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx2))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx2, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block2.Hash())
}
func TestTxIndexDuplicateTx2(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	tx1 := common.Bytes("tx1")
	tx2 := common.Bytes("tx2")
	tx3 := common.Bytes("tx3")
	tx4 := common.Bytes("tx4")

	core.ResetTestBlocks()
	chain := CreateTestChain()

	block1 := core.CreateTestBlock("b1", "")
	block1.Height = 10
	block1.Txs = []common.Bytes{tx1, tx2}

	block2 := core.CreateTestBlock("b2", "")
	block2.Height = 20
	block2.Txs = []common.Bytes{tx2, tx3}

	block3 := core.CreateTestBlock("b3", "")
	block3.Height = 30
	block3.Txs = []common.Bytes{tx3, tx4}

	block4 := core.CreateTestBlock("b4", "")
	block4.Height = 15
	block4.Txs = []common.Bytes{tx4, tx1}

	_, err := chain.AddBlock(block1)
	require.Nil(err)

	_, err = chain.AddBlock(block2)
	require.Nil(err)

	_, err = chain.AddBlock(block3)
	require.Nil(err)

	_, err = chain.AddBlock(block4)
	require.Nil(err)

	tx, block, found := chain.FindTxByHash(crypto.Keccak256Hash(tx1))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx1, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block1.Hash())

	// Tx2 should be linked with block1 instead of block2.
	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx2))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx2, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block1.Hash())

	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx3))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx3, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block2.Hash())

	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx4))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx4, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block3.Hash())

	// Tx2 should be linked with block2 after force insert.
	eb := &core.ExtendedBlock{Block: block2}
	chain.AddTxsToIndex(eb, true)
	tx, block, found = chain.FindTxByHash(crypto.Keccak256Hash(tx2))
	assert.True(found)
	assert.NotNil(tx)
	assert.Equal(tx2, tx)
	assert.NotNil(block)
	assert.Equal(block.Hash(), block2.Hash())
}
