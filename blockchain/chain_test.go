// +build unit

package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

	AssertChainsEqual(assert, expected.Root, chain.Root)
}
