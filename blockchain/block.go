package blockchain

import (
	"fmt"

	"github.com/thetatoken/ukulele/types"
)

// Block represents a block in chain.
type Block struct {
	BlockHeader
	Txs []Tx
}

func (b *Block) String() string {
	return fmt.Sprintf("Block{Header: %v, Txs: %d}", b.BlockHeader, len(b.Txs))
}

// BlockHeader contains the essential information of a block.
type BlockHeader struct {
	ChainID    string
	Epoch      uint32
	Hash       types.Bytes
	ParentHash types.Bytes
}

func (h BlockHeader) String() string {
	return fmt.Sprintf("{ChainID: %v, Epoch: %d, Hash: %v. Parent: %v}", h.ChainID, h.Epoch, h.Hash, h.ParentHash)
}

// Tx represents a transaction.
type Tx struct {
}
