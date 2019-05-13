package core

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

var TestBlocks map[string]*Block = make(map[string]*Block)
var TestBlocksLock = &sync.Mutex{}

var DefaultSigner *crypto.PrivateKey
var epoch uint64

func init() {
	DefaultSigner, _, _ = crypto.GenerateKeyPair()
	epoch = 1
}

func ResetTestBlocks() {
	TestBlocksLock.Lock()
	defer TestBlocksLock.Unlock()

	TestBlocks = make(map[string]*Block)
}

func GetTestBlock(name string) *Block {
	TestBlocksLock.Lock()
	defer TestBlocksLock.Unlock()

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

	block, ok := TestBlocks[name]
	if ok {
		return block
	}

	block = NewBlock()
	block.ChainID = "testchain"
	block.StateHash = common.HexToHash(name)

	var pBlock *Block
	if parent != "" {
		var ok bool
		pBlock, ok = TestBlocks[parent]
		block.Parent = pBlock.Hash()
		block.Height = pBlock.Height + 1
		if !ok {
			panic(fmt.Sprintf("Failed to find test block %v", parent))
		}
	}

	epoch++
	block.Epoch = epoch
	block.HCC.BlockHash = block.Parent
	block.Proposer = DefaultSigner.PublicKey().Address()
	block.AddTxs([]common.Bytes{})
	block.Timestamp = big.NewInt(time.Now().Unix())
	block.Signature, _ = DefaultSigner.Sign(block.SignBytes())

	TestBlocksLock.Lock()
	defer TestBlocksLock.Unlock()
	TestBlocks[name] = block

	return block
}
