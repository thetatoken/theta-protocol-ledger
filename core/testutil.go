package core

import (
	"fmt"
	"strings"

	"github.com/thetatoken/ukulele/common"
)

var TestBlocks map[string]*Block = make(map[string]*Block)

func ResetTestBlocks() {
	TestBlocks = make(map[string]*Block)
}

func GetTestBlock(name string) *Block {
	name = strings.ToLower(name)
	block, ok := TestBlocks[name]
	if !ok {
		panic(fmt.Sprintf("Failed to find test block %v", name))
	}
	return block
}

// CreateTestBlock creates a block for testing.
func CreateTestBlock(name string, parent string) *Block {
	name = strings.ToLower(name)
	parent = strings.ToLower(parent)

	block := NewBlock()
	block.ChainID = "testchain"
	// block.hash = common.HexToHash(hash)
	block.StateHash = common.HexToHash(name)
	if parent != "" {
		pBlock, ok := TestBlocks[parent]
		if !ok {
			panic(fmt.Sprintf("Failed to find test block %v", parent))
		}
		block.Parent = pBlock.Hash()
		block.Height = pBlock.Height + 1
	}
	TestBlocks[name] = block
	return block
}
