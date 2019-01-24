// +build unit

package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/core"
)

func TestChainUtil(t *testing.T) {
	assert := assert.New(t)

	core.ResetTestBlocks()

	c1 := CreateTestChainByBlocks([]string{})
	c2 := CreateTestChainByBlocks([]string{})
	c3 := CreateTestChainByBlocks([]string{"a1", "a0"})
	c4 := CreateTestChainByBlocks([]string{"a1", "a0"})
	c5 := CreateTestChainByBlocks([]string{"a1", "a0", "a2", "a1", "b2", "a1", "b3", "b2"})
	c6 := CreateTestChainByBlocks([]string{"a1", "a0", "a2", "a1", "b2", "a1", "b3", "b2"})

	AssertChainsEqual(assert, c1, c1.Root().Hash(), c2, c2.Root().Hash())

	AssertChainsNotEqual(assert, c1, c1.Root().Hash(), c3, c3.Root().Hash())
	AssertChainsEqual(assert, c3, c3.Root().Hash(), c4, c4.Root().Hash())

	AssertChainsNotEqual(assert, c3, c3.Root().Hash(), c6, c6.Root().Hash())
	AssertChainsEqual(assert, c5, c5.Root().Hash(), c6, c6.Root().Hash())
}
