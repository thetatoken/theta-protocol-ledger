package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store"
)

// ParseHex parse hex string into bytes.
func ParseHex(s string) common.Bytes {
	bytes, err := hex.DecodeString(s)
	if err != nil {
		panic(fmt.Sprintf("Error parsing: \"%s\": %v\n", s, err))
	}
	return bytes
}

// CreateTestBlock creates a block for testing.
func CreateTestBlock(hash string, parent string) *Block {
	block := &Block{}
	block.ChainID = "testchain"
	block.Hash = ParseHex(hash)
	block.ParentHash = ParseHex(parent)
	return block
}

// CreateTestChain creates a chain for testing.
func CreateTestChain() *Chain {
	store := store.NewMemKVStore()
	root := &Block{}
	root.ChainID = "testchain"
	root.Epoch = 0
	root.Hash = ParseHex("a0")

	chain := NewChain("testchain", store, root)
	return chain
}

// CreateTestChainByBlocks creates a chain with given string slice in format:
//   [block1_hash, block1_parent_hash, block2_hash, block1_parent_hash, ...]
func CreateTestChainByBlocks(pairs []string) *Chain {
	chain := CreateTestChain()
	for i := 0; i < len(pairs); i += 2 {
		block := CreateTestBlock(pairs[i], pairs[i+1])
		_, err := chain.AddBlock(block)
		if err != nil {
			panic(err)
		}
	}
	return chain
}

// AreChainsEqual returns whehter two chains are the same.
func AreChainsEqual(c1 *ExtendedBlock, c2 *ExtendedBlock) (bool, string) {
	if 0 != bytes.Compare(c1.Hash, c2.Hash) {
		return false, fmt.Sprintf("%v != %v", c1.Hash, c2.Hash)
	}
	if len(c1.Children) != len(c2.Children) {
		return false, fmt.Sprintf("len(%v.Children) != len(%v.Children)", c1.Hash, c2.Hash)
	}
	for i := 0; i < len(c1.Children); i++ {
		eq, msg := AreChainsEqual(c1.Children[i], c2.Children[i])
		if !eq {
			return false, msg
		}
	}
	return true, ""
}

// AssertChainsEqual asserts that two chains are the same.
func AssertChainsEqual(assert *assert.Assertions, c1 *ExtendedBlock, c2 *ExtendedBlock) {
	eq, msg := AreChainsEqual(c1, c2)
	assert.True(eq, msg)
}

// AssertChainsNotEqual asserts that two chains are not the same.
func AssertChainsNotEqual(assert *assert.Assertions, c1 *ExtendedBlock, c2 *ExtendedBlock) {
	eq, _ := AreChainsEqual(c1, c2)
	assert.False(eq)
}
