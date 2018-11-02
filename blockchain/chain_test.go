package blockchain

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockchain(t *testing.T) {
	assert := assert.New(t)

	expected := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"b2", "a1",
		"c1", "a0"})
	var err error

	chain := CreateTestChain()
	a1 := CreateTestBlock("a1", "a0")
	_, err = chain.AddBlock(a1)
	assert.Nil(err)

	a2 := CreateTestBlock("a2", "a1")
	_, err = chain.AddBlock(a2)
	assert.Nil(err)

	b2 := CreateTestBlock("b2", "a1")
	_, err = chain.AddBlock(b2)
	assert.Nil(err)

	c1 := CreateTestBlock("c1", "a0")
	_, err = chain.AddBlock(c1)
	assert.Nil(err)

	AssertChainsEqual(assert, expected, expected.Root.Hash, chain, chain.Root.Hash)
}

func TestBlockchainDeepestDescendant(t *testing.T) {
	assert := assert.New(t)
	ch := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"b2", "a1",
		"b3", "b2",
		"c1", "a0"})

	ret, depth := ch.FindDeepestDescendant(ch.Root.Hash)
	assert.True(bytes.Equal(ParseHex("b3"), ret.Hash), "Expected deepest block: %v, actual: %v", ParseHex("b3"), ret.Hash)
	assert.Equal(3, depth)
}

func TestFinalizePreviousBlocks(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	ch := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"a3", "a2",
		"a4", "a3",
		"a5", "a4",
		"b2", "a1",
		"b3", "b2",
		"c1", "a0",
	})
	block, err := ch.FindBlock(ParseHex("a3"))
	require.Nil(err)

	ch.FinalizePreviousBlocks(block)

	for _, hash := range []string{"a0", "a1", "a2", "a3"} {
		block, err = ch.FindBlock(ParseHex(hash))
		assert.Nil(err)
		assert.True(block.Finalized)
	}

	for _, hash := range []string{"b2", "b3", "c1", "a4", "a5"} {
		block, err = ch.FindBlock(ParseHex(hash))
		assert.False(block.Finalized)
	}

	block, err = ch.FindBlock(ParseHex("a5"))
	require.Nil(err)
	ch.FinalizePreviousBlocks(block)

	for _, hash := range []string{"a0", "a1", "a2", "a3", "a4", "a5"} {
		block, err = ch.FindBlock(ParseHex(hash))
		assert.True(block.Finalized)
	}

	for _, hash := range []string{"b2", "b3", "c1"} {
		block, err = ch.FindBlock(ParseHex(hash))
		assert.False(block.Finalized)
	}

}
