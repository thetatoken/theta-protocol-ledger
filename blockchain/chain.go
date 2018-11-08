package blockchain

import (
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
	rootBlock, err := chain.FindBlock(root.Hash())
	if err != nil {
		log.WithFields(log.Fields{"Hash": root.Hash().Hex(), "error": err}).Info("Root block is not found in chain. Adding block.")
		rootBlock, err = chain.AddBlock(root)
		if err != nil {
			log.Panic(err)
		}
	}
	chain.FinalizePreviousBlocks(rootBlock)
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
	hash := block.Hash()
	err := ch.store.Get(hash[:], val)
	if err == nil {
		// Block has already been added.
		return val, fmt.Errorf("Block has already been added: %X", hash[:])
	}

	if !block.Parent.IsEmpty() {
		parentBlock, err := ch.findBlock(block.Parent)
		if err == store.ErrKeyNotFound {
			// Parent block is not known yet, abandon block.
			return nil, errors.Errorf("Unknown parent block: %s", block.Parent)
		}
		if err != nil {
			return nil, errors.Wrap(err, "Failed to find parent block")
		}

		parentBlock.Children = append(parentBlock.Children, hash)
		ch.SaveBlock(parentBlock)
	}

	extendedBlock := &core.ExtendedBlock{Block: block}

	err = ch.SaveBlock(extendedBlock)
	if err != nil {
		log.Panic(err)
	}

	ch.AddTxsToIndex(extendedBlock, false)

	return extendedBlock, nil
}

func (ch *Chain) FinalizePreviousBlocks(block *core.ExtendedBlock) {
	var err error
	for block != nil && !block.Finalized {
		block.Finalized = true
		err = ch.SaveBlock(block)
		if err != nil {
			log.Panic(err)
		}
		block, err = ch.FindBlock(block.Parent)
		if err != nil {
			return
		}
	}
}

// FindDeepestDescendant finds the deepest descendant of given block.
func (ch *Chain) FindDeepestDescendant(hash common.Hash) (n *core.ExtendedBlock, depth int) {
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
	_, err := ch.FindBlock(block.Parent)
	return err != nil
}

// SaveBlock updates a previously stored block.
func (ch *Chain) SaveBlock(block *core.ExtendedBlock) error {
	hash := block.Hash()
	return ch.store.Put(hash[:], *block)
}

// FindBlock tries to retrieve a block by hash.
func (ch *Chain) FindBlock(hash common.Hash) (*core.ExtendedBlock, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()
	return ch.findBlock(hash)
}

// findBlock is the non-locking version of FindBlock.
func (ch *Chain) findBlock(hash common.Hash) (*core.ExtendedBlock, error) {
	var block core.ExtendedBlock
	err := ch.store.Get(hash[:], &block)
	if err != nil {
		return nil, err
	}
	return &block, nil
}

// IsDescendant determines whether one block is the ascendant of another block.
func (ch *Chain) IsDescendant(ascendantHash common.Hash, descendantHash common.Hash, maxDistance int) bool {
	hash := descendantHash
	for i := 0; i < maxDistance; i++ {
		if hash == ascendantHash {
			return true
		}
		currBlock, err := ch.FindBlock(hash)
		if err != nil {
			return false
		}
		hash = currBlock.Parent
	}
	return false
}

// PrintBranch return the string describing path from root to given leaf.
func (ch *Chain) PrintBranch(hash common.Hash) string {
	ret := []string{}
	for {
		var currBlock core.ExtendedBlock
		err := ch.store.Get(hash[:], &currBlock)
		if err != nil {
			break
		}
		ret = append(ret, hash.String())
		hash = currBlock.Parent
	}
	return fmt.Sprintf("%v", ret)
}
