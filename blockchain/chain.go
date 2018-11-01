package blockchain

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/store"
)

// Chain represents the blockchain and also is the interface to underlying store.
type Chain struct {
	store store.Store

	ChainID string
	Root    *core.ExtendedBlock `rlp:"nil"`

	mu *sync.Mutex
}

// NewChain creates a new Chain instance.
func NewChain(chainID string, store store.Store, root *core.Block) *Chain {
	chain := &Chain{
		ChainID: chainID,
		store:   store,
		mu:      &sync.Mutex{},
	}
	rootBlock, _ := chain.AddBlock(root)
	chain.Root = rootBlock
	return chain
}

// AddBlock adds a block to the chain and underlying store.
func (ch *Chain) AddBlock(block *core.Block) (*core.ExtendedBlock, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if block.ChainID != ch.ChainID {
		return nil, errors.Errorf("ChainID mismatch: block.ChainID(%s) != %s", block.ChainID, ch.ChainID)
	}

	val := &core.ExtendedBlock{}
	err := ch.store.Get(block.Hash, val)
	if err == nil {
		// Block has already been added.
		return val, errors.New("Block has already been added")
	}

	if len(block.Parent) != 0 {
		var parentBlock core.ExtendedBlock
		err = ch.store.Get(block.Parent, &parentBlock)
		if err == store.ErrKeyNotFound {
			// Parent block is not known yet, abandon block.
			return nil, errors.Errorf("Unknown parent block: %s", block.Parent)
		}
		if err != nil {
			return nil, errors.Wrap(err, "Failed to find parent block")
		}

		parentBlock.Children = append(parentBlock.Children, block.Hash)
		ch.SaveBlock(&parentBlock)
	}

	extendedBlock := &core.ExtendedBlock{Block: block}

	err = ch.SaveBlock(extendedBlock)
	if err != nil {
		log.Panic(err)
	}

	ch.AddTxsToIndex(extendedBlock, false)

	return extendedBlock, nil
}

// FindDeepestDescendant finds the deepest descendant of given block.
func (ch *Chain) FindDeepestDescendant(hash common.Bytes) (n *core.ExtendedBlock, depth int) {
	// TODO: replace recursive implementation with stack-based implementation.
	n, err := ch.FindBlock(hash)
	if err != nil {
		return nil, -1
	}
	depth = 0
	for _, child := range n.Children {
		ret, retDepth := ch.FindDeepestDescendant(child)
		if retDepth+1 > depth {
			n = ret
			depth = retDepth + 1
		}
	}
	return
}

func (ch *Chain) IsOrphan(block *core.Block) bool {
	var val core.ExtendedBlock
	err := ch.store.Get(block.Parent, &val)
	return err != nil
}

// SaveBlock updates a previously stored block.
func (ch *Chain) SaveBlock(block *core.ExtendedBlock) error {
	return ch.store.Put(block.Hash, *block)
}

// FindBlock tries to retrieve a block by hash.
func (ch *Chain) FindBlock(hash common.Bytes) (*core.ExtendedBlock, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	var block core.ExtendedBlock
	err := ch.store.Get(hash, &block)
	if err != nil {
		return nil, err
	}

	// Returns a copy of the block.
	ret := &core.ExtendedBlock{
		Block:             block.Block,
		Children:          make([]common.Bytes, len(block.Children)),
		CommitCertificate: block.CommitCertificate,
	}
	copy(ret.Children, block.Children)
	return ret, nil
}

// IsDescendant determines whether one block is the ascendant of another block.
func (ch *Chain) IsDescendant(ascendantHash common.Bytes, descendantHash common.Bytes, maxDistance int) bool {
	hash := descendantHash
	for i := 0; i < maxDistance; i++ {
		if bytes.Compare(hash, ascendantHash) == 0 {
			return true
		}
		var currBlock core.ExtendedBlock
		err := ch.store.Get(hash, &currBlock)
		if err != nil {
			return false
		}
		hash = currBlock.Parent
	}
	return false
}

// PrintBranch return the string describing path from root to given leaf.
func (ch *Chain) PrintBranch(hash common.Bytes) string {
	ret := []string{}
	for {
		var currBlock core.ExtendedBlock
		err := ch.store.Get(hash, &currBlock)
		if err != nil {
			break
		}
		ret = append(ret, hash.String())
		hash = currBlock.Parent
	}
	return fmt.Sprintf("%v", ret)
}
