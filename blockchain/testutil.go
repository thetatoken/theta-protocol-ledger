package blockchain

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store/blockstore"
	"github.com/thetatoken/ukulele/store/database/backend"
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
	store := blockstore.NewBlockStore(backend.NewMemDatabase())
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
func AreChainsEqual(ch1 *Chain, head1 common.Bytes, ch2 *Chain, head2 common.Bytes) (bool, string) {
	if 0 != bytes.Compare(head1, head2) {
		return false, fmt.Sprintf("%v != %v", head1, head2)
	}
	c1, err := ch1.FindBlock(head1)
	if err != nil {
		return false, err.Error()
	}
	c2, err := ch2.FindBlock(head2)
	if err != nil {
		return false, err.Error()
	}
	if len(c1.Children) != len(c2.Children) {
		return false, fmt.Sprintf("len(%v.Children) != len(%v.Children)", c1.Hash, c2.Hash)
	}
	for i := 0; i < len(c1.Children); i++ {
		eq, msg := AreChainsEqual(ch1, c1.Children[i], ch2, c2.Children[i])
		if !eq {
			return false, msg
		}
	}
	return true, ""
}

// AssertChainsEqual asserts that two chains are the same.
func AssertChainsEqual(assert *assert.Assertions, ch1 *Chain, head1 common.Bytes, ch2 *Chain, head2 common.Bytes) {
	eq, msg := AreChainsEqual(ch1, head1, ch2, head2)
	assert.True(eq, msg)
}

// AssertChainsNotEqual asserts that two chains are not the same.
func AssertChainsNotEqual(assert *assert.Assertions, ch1 *Chain, head1 common.Bytes, ch2 *Chain, head2 common.Bytes) {
	eq, _ := AreChainsEqual(ch1, head1, ch2, head2)
	assert.False(eq)
}
