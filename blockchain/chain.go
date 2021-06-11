package blockchain

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/store"
)

const maxDistance = 2000

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "blockchain"})

// Chain represents the blockchain and also is the interface to underlying store.
type Chain struct {
	store store.Store

	ChainID string
	root    common.Hash

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
		logger.WithFields(log.Fields{"Hash": root.Hash().Hex()}).Info("Root block is not found in chain. Adding block.")
		rootBlock, err = chain.AddSnapshotRoot(root)
		if err != nil {
			logger.Panic(err)
		}
	}
	chain.FinalizePreviousBlocks(rootBlock.Hash())
	chain.root = rootBlock.Hash()
	return chain
}

// Root returns the root block
func (ch *Chain) Root() *core.ExtendedBlock {
	ret, _ := ch.FindBlock(ch.root)
	return ret
}

// AddSnapshotRoot adds the root block of the chain
func (ch *Chain) AddSnapshotRoot(block *core.Block) (*core.ExtendedBlock, error) {
	return ch.addBlock(block, true)
}

// AddBlock adds a block to the chain and underlying store
func (ch *Chain) AddBlock(block *core.Block) (*core.ExtendedBlock, error) {
	return ch.addBlock(block, false)
}

func (ch *Chain) addBlock(block *core.Block, isSnapshotRoot bool) (*core.ExtendedBlock, error) {
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

	// Update parent if present.
	if !block.Parent.IsEmpty() && !isSnapshotRoot {
		parentBlock, err := ch.findBlock(block.Parent)
		if err == nil {
			parentBlock.Children = append(parentBlock.Children, hash)
			err = ch.saveBlock(parentBlock)
			if err != nil {
				log.Panic(err)
			}
		}
	}

	extendedBlock := &core.ExtendedBlock{Block: block}

	// Update children if present.
	children := ch.findBlocksByHeight(block.Height + 1)
	extendedBlock.Children = []common.Hash{}
	for i := 0; i < len(children); i++ {
		if children[i].Parent != block.Hash() {
			continue
		}
		extendedBlock.Children = append(extendedBlock.Children, children[i].Hash())
	}

	err = ch.saveBlock(extendedBlock)
	if err != nil {
		logger.Panic(err)
	}

	ch.AddBlockByHeightIndex(extendedBlock.Height, extendedBlock.Hash())
	ch.AddTxsToIndex(extendedBlock, false)

	return extendedBlock, nil
}

// FixBlockIndex fixes index for given block.
func (ch *Chain) FixBlockIndex(block *core.ExtendedBlock) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.AddBlockByHeightIndex(block.Height, block.Hash())
	ch.AddTxsToIndex(block, false)
}

// FixMissingChildren removes dead links to missing children blocks.
func (ch *Chain) FixMissingChildren(block *core.ExtendedBlock) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	newChildren := []common.Hash{}
	for _, hash := range block.Children {
		_, err := ch.findBlock(hash)
		if err != nil {
			logger.Warningf("Removing dead link from block $v to block %v", block.Hash().Hex(), hash.Hex())
		} else {
			newChildren = append(newChildren, hash)
		}
	}
	block.Children = newChildren

	err := ch.saveBlock(block)
	if err != nil {
		logger.Panic(err)
	}
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

func (ch *Chain) MarkBlockValid(hash common.Hash) *core.ExtendedBlock {
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
	return block
}

func (ch *Chain) MarkBlockInvalid(hash common.Hash) *core.ExtendedBlock {
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
	return block
}

func (ch *Chain) MarkBlockHasValidatorUpdate(hash common.Hash) *core.ExtendedBlock {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	block, err := ch.findBlock(hash)
	if err != nil {
		logger.Panic(err)
	}
	block.HasValidatorUpdate = true
	err = ch.saveBlock(block)
	if err != nil {
		logger.Panic(err)
	}
	return block
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

func (ch *Chain) FinalizePreviousBlocks(hash common.Hash) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	status := core.BlockStatusDirectlyFinalized
	for !hash.IsEmpty() {
		block, err := ch.findBlock(hash)
		if err != nil || block.Status.IsFinalized() {
			return nil
		}
		if block.Status == core.BlockStatusDisposed {
			return errors.New("Cannot finalize disposed branch")
		}
		block.Status = status
		status = core.BlockStatusIndirectlyFinalized // Only the first block is marked as directly finalized
		err = ch.saveBlock(block)
		if err != nil {
			logger.Panic(err)
		}

		// Force update TX index on block finalization so that the index doesn't point to
		// duplicate TX in fork.
		ch.AddTxsToIndex(block, true)

		hash = block.Parent
	}
	return nil
}

func (ch *Chain) IsOrphan(block *core.Block) bool {
	_, err := ch.FindBlock(block.Parent)
	return err != nil
}

// saveBlock updates a previously stored block.
func (ch *Chain) saveBlock(block *core.ExtendedBlock) error {
	hash := block.Hash()
	return ch.store.Put(hash[:], block)
}

func (ch *Chain) SaveBlock(block *core.ExtendedBlock) error {
	return ch.saveBlock(block)
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
func (ch *Chain) IsDescendant(ascendantHash common.Hash, descendantHash common.Hash) bool {
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
