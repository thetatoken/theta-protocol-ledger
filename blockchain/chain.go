package blockchain

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/store"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "blockchain"})

// Chain represents the blockchain and also is the interface to underlying store.
type Chain struct {
	store store.Store

	ChainID string
	Root    *core.ExtendedBlock `rlp:"nil"`

	mu *sync.RWMutex
}

// NewChain creates a new Chain instance.
func NewChain(chainID string, store store.Store, root *core.Block) *Chain {
	chain := &Chain{
		ChainID: chainID,
		store:   store,
		mu:      &sync.RWMutex{},
	}
	rootBlock, err := chain.FindBlock(root.Hash())
	if err != nil {
		logger.WithFields(log.Fields{"Hash": root.Hash().Hex(), "error": err}).Info("Root block is not found in chain. Adding block.")
		rootBlock, err = chain.AddBlock(root)
		if err != nil {
			logger.Panic(err)
		}
	}
	chain.FinalizePreviousBlocks(rootBlock.Hash())
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

		err = ch.saveBlock(parentBlock)
		if err != nil {
			logger.Panic(err)
		}
	}

	extendedBlock := &core.ExtendedBlock{Block: block}

	err = ch.saveBlock(extendedBlock)
	if err != nil {
		logger.Panic(err)
	}

	ch.AddBlockByHeightIndex(extendedBlock.Height, extendedBlock.Hash())
	ch.AddTxsToIndex(extendedBlock, false)

	return extendedBlock, nil
}

// blockByHeightIndexKey constructs the DB key for the given block height.
func blockByHeightIndexKey(height uint64) common.Bytes {
	// convert uint64 to []byte
	buf := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(buf, height)
	b := buf[:n]
	return append(common.Bytes("bh/"), b...)
}

type BlockByHeightIndexEntry struct {
	Blocks []common.Hash
}

func (ch *Chain) AddBlockByHeightIndex(height uint64, block common.Hash) {
	key := blockByHeightIndexKey(height)
	blockByHeightIndexEntry := BlockByHeightIndexEntry{
		Blocks: []common.Hash{},
	}

	ch.store.Get(key, &blockByHeightIndexEntry)

	// Check if block has already been added to index.
	for _, b := range blockByHeightIndexEntry.Blocks {
		if block == b {
			return
		}
	}

	blockByHeightIndexEntry.Blocks = append(blockByHeightIndexEntry.Blocks, block)

	err := ch.store.Put(key, blockByHeightIndexEntry)
	if err != nil {
		logger.Panic(err)
	}
}

// FindBlocksByHeight tries to retrieve blocks by height.
func (ch *Chain) FindBlocksByHeight(height uint64) []*core.ExtendedBlock {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.findBlocksByHeight(height)
}

// findBlocksByHeight is the non-locking version of FindBlockByHeight.
func (ch *Chain) findBlocksByHeight(height uint64) []*core.ExtendedBlock {
	key := blockByHeightIndexKey(height)
	blockByHeightIndexEntry := BlockByHeightIndexEntry{
		Blocks: []common.Hash{},
	}
	ch.store.Get(key, &blockByHeightIndexEntry)

	ret := []*core.ExtendedBlock{}
	for _, hash := range blockByHeightIndexEntry.Blocks {
		block, err := ch.findBlock(hash)
		if err == nil {
			ret = append(ret, block)
		}

	}
	return ret
}

func (ch *Chain) MarkBlockValid(hash common.Hash) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	block, err := ch.findBlock(hash)
	if err != nil {
		logger.Panic(err)
	}
	block.Status = core.BlockStatusValid
	err = ch.saveBlock(block)
	if err != nil {
		logger.Panic(err)
	}
}

func (ch *Chain) MarkBlockInvalid(hash common.Hash) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	block, err := ch.findBlock(hash)
	if err != nil {
		logger.Panic(err)
	}
	block.Status = core.BlockStatusInvalid
	err = ch.saveBlock(block)
	if err != nil {
		logger.Panic(err)
	}
}

func (ch *Chain) CommitBlock(hash common.Hash) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	block, err := ch.findBlock(hash)
	if err != nil {
		logger.Panic(err)
	}
	block.Status = core.BlockStatusCommitted
	err = ch.saveBlock(block)
	if err != nil {
		logger.Panic(err)
	}
}

func (ch *Chain) FinalizePreviousBlocks(hash common.Hash) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	status := core.BlockStatusDirectlyFinalized
	for !hash.IsEmpty() {
		block, err := ch.findBlock(hash)
		if err != nil || block.Status.IsFinalized() {
			return
		}
		block.Status = status
		status = core.BlockStatusIndirectlyFinalized // Only the first block is marked as directly finalized
		err = ch.saveBlock(block)
		if err != nil {
			logger.Panic(err)
		}
		hash = block.Parent
	}
}

// FindDeepestDescendant finds the deepest descendant of given block.
func (ch *Chain) FindDeepestDescendant(hash common.Hash) (n *core.ExtendedBlock, depth int) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.findDeepestDescendant(hash)
}

// findDeepestDescendant is the non-locking version of FindDeepestDescendant.
func (ch *Chain) findDeepestDescendant(hash common.Hash) (n *core.ExtendedBlock, depth int) {
	// TODO: replace recursive implementation with stack-based implementation.
	n, err := ch.findBlock(hash)
	if err != nil || !n.Status.IsValid() {
		return nil, -1
	}
	depth = 0
	for _, child := range n.Children {
		ret, retDepth := ch.findDeepestDescendant(child)
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

// saveBlock updates a previously stored block.
func (ch *Chain) saveBlock(block *core.ExtendedBlock) error {
	hash := block.Hash()
	return ch.store.Put(hash[:], *block)
}

// FindBlock tries to retrieve a block by hash.
func (ch *Chain) FindBlock(hash common.Hash) (*core.ExtendedBlock, error) {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
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
