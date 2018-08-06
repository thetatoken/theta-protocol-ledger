// +build unit

package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainUtil(t *testing.T) {
	assert := assert.New(t)

	c1 := CreateTestChainByBlocks([]string{})
	c2 := CreateTestChainByBlocks([]string{})
	c3 := CreateTestChainByBlocks([]string{"a1", "a0"})
	c4 := CreateTestChainByBlocks([]string{"a1", "a0"})
	c5 := CreateTestChainByBlocks([]string{"a1", "a0", "a2", "a1", "b2", "a1", "b3", "b2"})
	c6 := CreateTestChainByBlocks([]string{"a1", "a0", "a2", "a1", "b2", "a1", "b3", "b2"})

	AssertChainsEqual(assert, c1.Root, c2.Root)

	AssertChainsNotEqual(assert, c1.Root, c3.Root)
	AssertChainsEqual(assert, c3.Root, c4.Root)

	AssertChainsNotEqual(assert, c3.Root, c6.Root)
	AssertChainsEqual(assert, c5.Root, c6.Root)
}
