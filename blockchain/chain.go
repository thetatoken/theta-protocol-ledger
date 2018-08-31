package blockchain

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/pkg/errors"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/store"
)

// ExtendedBlock is wrapper over Block, containing extra information related to the block.
type ExtendedBlock struct {
	*Block
	Height            uint32
	Children          []common.Bytes
	Parent            common.Bytes
	CommitCertificate *CommitCertificate
}

func (eb *ExtendedBlock) String() string {
	children := bytes.NewBufferString("[")
	start := true
	for _, c := range eb.Children {
		if !start {
			children.WriteString(",")
			start = false
		}
		children.WriteString(c.String())
	}
	children.WriteString("]")
	return fmt.Sprintf("ExtendedBlock{Block: %v, Parent: %v, Children: %v, CC: %v}", eb.Block, eb.Parent.String(), children, eb.CommitCertificate)
}

// ShortString returns a short string describing the block.
func (eb *ExtendedBlock) ShortString() string {
	return eb.Hash.String()
}

// Chain represents the blockchain and also is the interface to underlying store.
type Chain struct {
	store store.Store

	ChainID string
	Root    *ExtendedBlock

	mu *sync.Mutex
}

// NewChain creates a new Chain instance.
func NewChain(chainID string, store store.Store, root *Block) *Chain {
	rootBlock := &ExtendedBlock{Block: root}
	chain := &Chain{
		ChainID: chainID,
		store:   store,
		Root:    rootBlock,
		mu:      &sync.Mutex{},
	}
	chain.SaveBlock(rootBlock)
	return chain
}

// AddBlock adds a block to the chain and underlying store.
func (ch *Chain) AddBlock(block *Block) (*ExtendedBlock, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if block.ChainID != ch.ChainID {
		return nil, errors.Errorf("ChainID mismatch: block.ChainID(%s) != %s", block.ChainID, ch.ChainID)
	}

	_, err := ch.store.Get(block.Hash)
	if err != store.ErrKeyNotFound {
		// Block has already been added.
		return nil, errors.New("Block has already been added")
	}

	if block.ParentHash == nil {
		// Parent block hash cannot be empty.
		return nil, errors.New("Parent block hash cannot be empty")
	}
	parentRaw, err := ch.store.Get(block.ParentHash)
	if err == store.ErrKeyNotFound {
		// Parent block is not known yet, abandon block.
		return nil, errors.Errorf("Unknown parent block: %s", block.ParentHash)
	}
	if err != nil {
		return nil, errors.Wrap(err, "Failed to find parent block")
	}

	parentBlock := parentRaw.(*ExtendedBlock)
	extendedBlock := &ExtendedBlock{Block: block, Parent: parentBlock.Hash, Height: parentBlock.Height + 1}
	parentBlock.Children = append(parentBlock.Children, extendedBlock.Hash)
	ch.SaveBlock(parentBlock)
	ch.SaveBlock(extendedBlock)

	return extendedBlock, nil
}

// FindDeepestDescendant finds the deepest descendant of given block.
func (ch *Chain) FindDeepestDescendant(hash common.Bytes) (n *ExtendedBlock, depth int) {
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

func (ch *Chain) IsOrphan(block *Block) bool {
	_, err := ch.store.Get(block.ParentHash)
	return err != nil
}

// SaveBlock updates a previously stored block.
func (ch *Chain) SaveBlock(block *ExtendedBlock) {
	ch.store.Put(block.Hash, block)
}

// FindBlock tries to retrieve a block by hash.
func (ch *Chain) FindBlock(hash common.Bytes) (*ExtendedBlock, error) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	res, err := ch.store.Get(hash)
	if err != nil {
		return nil, err
	}
	block := res.(*ExtendedBlock)

	// Returns a copy of the block.
	ret := &ExtendedBlock{
		Block:             block.Block,
		Height:            block.Height,
		Parent:            block.Parent,
		Children:          make([]common.Bytes, len(block.Children)),
		CommitCertificate: block.CommitCertificate,
	}
	copy(ret.Children, block.Children)
	return ret, nil
}

// IsDescendant determines whether one block is the ascendant of another block.
func (ch *Chain) IsDescendant(ascendantHash common.Bytes, descendantHash common.Bytes) bool {
	const maxDistance = 50
	hash := descendantHash
	for i := 0; i < maxDistance; i++ {
		if bytes.Compare(hash, ascendantHash) == 0 {
			return true
		}
		currBlockRaw, err := ch.store.Get(hash)
		if err != nil {
			return false
		}
		currBlock := currBlockRaw.(*ExtendedBlock)
		hash = currBlock.ParentHash
	}
	return false
}

// PrintBranch return the string describing path from root to given leaf.
func (ch *Chain) PrintBranch(hash common.Bytes) string {
	ret := []string{}
	for {
		currBlockRaw, err := ch.store.Get(hash)
		if err != nil {
			break
		}
		currBlock := currBlockRaw.(*ExtendedBlock)
		ret = append(ret, hash.String())
		hash = currBlock.ParentHash
	}
	return fmt.Sprintf("%v", ret)
}
