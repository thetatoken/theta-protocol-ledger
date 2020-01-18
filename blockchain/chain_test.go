package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/core"
)

func TestBlockchain(t *testing.T) {
	assert := assert.New(t)
	core.ResetTestBlocks()

	expected := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"b2", "a1",
		"c1", "a0"})
	var err error

	chain := CreateTestChain()
	a1 := core.CreateTestBlock("a1", "a0")
	b, err := chain.AddBlock(a1)
	assert.Nil(err)
	b.Status = core.BlockStatusValid
	chain.saveBlock(b)

	a2 := core.CreateTestBlock("a2", "a1")
	b, err = chain.AddBlock(a2)
	assert.Nil(err)
	b.Status = core.BlockStatusValid
	chain.saveBlock(b)

	b2 := core.CreateTestBlock("b2", "a1")
	b, err = chain.AddBlock(b2)
	assert.Nil(err)
	b.Status = core.BlockStatusValid
	chain.saveBlock(b)

	c1 := core.CreateTestBlock("c1", "a0")
	b, err = chain.AddBlock(c1)
	assert.Nil(err)
	b.Status = core.BlockStatusValid
	chain.saveBlock(b)

	AssertChainsEqual(assert, expected, expected.Root().Hash(), chain, chain.Root().Hash())
}

func TestFinalizePreviousBlocks(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	core.ResetTestBlocks()

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
	block, err := ch.FindBlock(core.GetTestBlock("a3").Hash())
	require.Nil(err)

	ch.FinalizePreviousBlocks(block.Hash())

	for _, name := range []string{"a0", "a1", "a2", "a3"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.Nil(err)
		assert.True(block.Status.IsFinalized())
	}

	for _, name := range []string{"b2", "b3", "c1", "a4", "a5"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.False(block.Status.IsFinalized())
	}

	block, err = ch.FindBlock(core.GetTestBlock("a5").Hash())
	require.Nil(err)
	ch.FinalizePreviousBlocks(block.Hash())

	for _, name := range []string{"a0", "a1", "a2", "a3", "a4", "a5"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.True(block.Status.IsFinalized())
	}

	for _, name := range []string{"b2", "b3", "c1"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.False(block.Status.IsFinalized())
	}

}
func TestFinalizePreviousBlocks2(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	core.ResetTestBlocks()

	ch := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"a3", "a2",
		"a4", "a3",
		"a5", "a4",
		"b2", "a1",
		"b3", "b2",
		"b4", "b3",
		"b5", "b4",
		"b6", "b5",
		"b7", "b6",
		"c1", "a0",
	})
	block, err := ch.FindBlock(core.GetTestBlock("b3").Hash())
	require.Nil(err)

	ch.FinalizePreviousBlocks(block.Hash())

	for _, name := range []string{"a0", "a1", "b2", "b3"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.Nil(err)
		assert.True(block.Status.IsFinalized())
	}

	for _, name := range []string{"b7", "b6", "b5", "b4", "c1", "a2", "a3"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.False(block.Status.IsFinalized())
	}

	block, err = ch.FindBlock(core.GetTestBlock("a5").Hash())
	require.Nil(err)
	ch.FinalizePreviousBlocks(block.Hash())

	for _, name := range []string{"a0", "a1", "a2", "a3", "a4", "a5", "b2", "b3"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.True(block.Status.IsFinalized())
	}

	for _, name := range []string{"b7", "b6", "b5", "b4", "c1"} {
		block, err = ch.FindBlock(core.GetTestBlock(name).Hash())
		assert.False(block.Status.IsFinalized())
	}

}

func TestBlockIndex(t *testing.T) {
	assert := assert.New(t)
	core.ResetTestBlocks()

	chain := CreateTestChainByBlocks([]string{
		"a1", "a0",
		"a2", "a1",
		"b2", "a1",
		"c1", "a0"})

	block, _ := chain.FindBlock(core.GetTestBlock("a0").Hash())
	assert.NotNil(block)
	assert.Equal(core.GetTestBlock("a0").Hash(), block.Hash())

	blocks := chain.FindBlocksByHeight(0)
	assert.Equal(1, len(blocks))
	assert.Equal(core.GetTestBlock("a0").Hash(), blocks[0].Hash())

	blocks = chain.FindBlocksByHeight(2)
	assert.Equal(2, len(blocks))
	assert.Equal(core.GetTestBlock("a2").Hash(), blocks[0].Hash())
	assert.Equal(core.GetTestBlock("b2").Hash(), blocks[1].Hash())
}
