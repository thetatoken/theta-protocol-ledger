package core

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto/sha3"
	"github.com/thetatoken/ukulele/rlp"
)

const (
	// MaxNumRegularTxsPerBlock represents the max number of regular transaction can be inclulded in one block
	MaxNumRegularTxsPerBlock int = 100
)

// Block represents a block in chain.
type Block struct {
	*BlockHeader
	Txs []common.Bytes
}

// NewBlock creates a new Block.
func NewBlock() *Block {
	return &Block{BlockHeader: &BlockHeader{}}
}

func (b *Block) String() string {
	return fmt.Sprintf("Block{Header: %v, Txs: %d}", b.BlockHeader, b.Txs)
}

// BlockHeader contains the essential information of a block.
type BlockHeader struct {
	ChainID   string
	Epoch     uint64
	Hash      common.Bytes
	Height    uint64
	Parent    common.Bytes
	TxHash    common.Bytes
	StateHash common.Bytes
	Timestamp *big.Int
	Proposer  common.Address
}

// UpdateHash calculate and set Hash field of header.
func (h *BlockHeader) UpdateHash() {
	h.Hash = nil
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, h)
	var buff common.Hash
	hw.Sum(buff[:0])
	h.Hash = buff[:]
}

func (h *BlockHeader) String() string {
	return fmt.Sprintf("{ChainID: %v, Epoch: %d, Hash: %v. Parent: %v, Height: %v, TxHash: %v, StateHash: %v, Timestamp: %v, Proposer: %s}",
		h.ChainID, h.Epoch, h.Hash, h.Parent, h.Height, h.TxHash, h.StateHash, h.Timestamp, h.Proposer)
}

// ExtendedBlock is wrapper over Block, containing extra information related to the block.
type ExtendedBlock struct {
	*Block
	Children          []common.Bytes
	CommitCertificate *CommitCertificate `rlp:"nil"`
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
