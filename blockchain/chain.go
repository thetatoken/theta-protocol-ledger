package blockchain

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"

	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/types"
)

// ExtendedBlock is wrapper over Block, containing extra information related to the block.
type ExtendedBlock struct {
	*Block
	Height            uint32
	Children          []*ExtendedBlock
	Parent            *ExtendedBlock
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
		children.WriteString(c.ShortString())
	}
	children.WriteString("]")
	return fmt.Sprintf("ExtendedBlock{Block: %v, Parent: %v, Children: %v, CC: %v}", eb.Block, eb.Parent.ShortString(), children, eb.CommitCertificate)
}

// ShortString returns a short string describing the block.
func (eb *ExtendedBlock) ShortString() string {
	return eb.Hash.String()
}

// FindDeepestDescendant finds the deepest descendant of given block.
func (eb *ExtendedBlock) FindDeepestDescendant() (n *ExtendedBlock, depth int) {
	// TODO: replace recursive implementation with stack-based implementation.
	n = eb
	depth = 0
	for _, child := range eb.Children {
		ret, retDepth := child.FindDeepestDescendant()
		if retDepth+1 > depth {
			n = ret
			depth = retDepth + 1
		}
	}
	return
}

// Chain represents the blockchain and also is the interface to underlying store.
type Chain struct {
	store store.Store

	ChainID string
	Root    *ExtendedBlock
}

// NewChain creates a new Chain instance.
func NewChain(chainID string, store store.Store, root *Block) *Chain {
	rootBlock := &ExtendedBlock{Block: root}
	chain := &Chain{ChainID: chainID, store: store, Root: rootBlock}
	chain.SaveBlock(rootBlock)
	return chain
}

// AddBlock adds a block to the chain and underlying store.
func (ch *Chain) AddBlock(block *Block) (*ExtendedBlock, error) {
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
	extendedBlock := &ExtendedBlock{Block: block, Parent: parentBlock, Height: parentBlock.Height + 1}
	parentBlock.Children = append(parentBlock.Children, extendedBlock)
	ch.SaveBlock(parentBlock)
	ch.SaveBlock(extendedBlock)

	return extendedBlock, nil
}

// SaveBlock updates a previously stored block.
func (ch *Chain) SaveBlock(block *ExtendedBlock) {
	ch.store.Put(block.Hash, block)
}

// FindBlock tries to retrieve a block by hash.
func (ch *Chain) FindBlock(hash types.Bytes) (*ExtendedBlock, error) {
	res, err := ch.store.Get(hash)
	if err != nil {
		return nil, err
	}
	block := res.(*ExtendedBlock)
	return block, nil
}

// IsDescendant determines whether one block is the ascendant of another block.
func (ch *Chain) IsDescendant(ascendantHash types.Bytes, descendantHash types.Bytes) bool {
	i := 0
	hash := descendantHash
	for i < 5 {
		if bytes.Compare(hash, ascendantHash) == 0 {
			return true
		}
		currBlockRaw, err := ch.store.Get(hash)
		if err != nil {
			return false
		}
		currBlock := currBlockRaw.(*ExtendedBlock)
		hash = currBlock.ParentHash
		i++
	}
	return false
}
