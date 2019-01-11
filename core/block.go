package core

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/trie"
)

const (
	// MaxNumRegularTxsPerBlock represents the max number of regular transaction can be inclulded in one block
	MaxNumRegularTxsPerBlock int = 1024
)

var (
	EmptyRootHash = calculateRootHash([]common.Bytes{})
)

// Block represents a block in chain.
type Block struct {
	*BlockHeader
	Txs []common.Bytes `json:"transactions"`
}

// NewBlock creates a new Block.
func NewBlock() *Block {
	return &Block{BlockHeader: &BlockHeader{}}
}

func (b *Block) String() string {
	txs := []string{}
	for _, tx := range b.Txs {
		txs = append(txs, hex.EncodeToString(tx))
	}
	return fmt.Sprintf("Block{Header: %v, Txs: %v}", b.BlockHeader, txs)
}

// AddTxs adds transactions to the block and update transaction root hash.
func (b *Block) AddTxs(txs []common.Bytes) {
	b.Txs = append(b.Txs, txs...)
	b.updateTxHash()
}

// updateTxHash calculate transaction root hash.
func (b *Block) updateTxHash() {
	b.TxHash = calculateRootHash(b.Txs)
	b.ReceiptHash = EmptyRootHash
}

func calculateRootHash(items []common.Bytes) common.Hash {
	keybuf := new(bytes.Buffer)
	trie := new(trie.Trie)
	for i := 0; i < len(items); i++ {
		keybuf.Reset()
		rlp.Encode(keybuf, uint(i))
		trie.Update(keybuf.Bytes(), items[i])
	}
	return trie.Hash()
}

// BlockHeader contains the essential information of a block.
type BlockHeader struct {
	ChainID     string
	Epoch       uint64
	Height      uint64
	Parent      common.Hash
	HCC         common.Hash
	TxHash      common.Hash
	ReceiptHash common.Hash
	Bloom       Bloom
	StateHash   common.Hash
	Timestamp   *big.Int
	Proposer    common.Address
	Signature   *crypto.Signature

	hash common.Hash // Cache of calculated hash.
}

// Hash of header.
func (h *BlockHeader) Hash() common.Hash {
	if h == nil {
		return common.Hash{}
	}
	if h.hash.IsEmpty() {
		raw, _ := rlp.EncodeToBytes(h)
		h.hash = crypto.Keccak256Hash(raw)
	}
	return h.hash
}

func (h *BlockHeader) String() string {
	return fmt.Sprintf("{ChainID: %v, Epoch: %d, Hash: %v. Parent: %v, Height: %v, TxHash: %v, StateHash: %v, Timestamp: %v, Proposer: %s}",
		h.ChainID, h.Epoch, h.Hash().Hex(), h.Parent.Hex(), h.Height, h.TxHash.Hex(), h.StateHash.Hex(), h.Timestamp, h.Proposer)
}

// SignBytes returns raw bytes to be signed.
func (h *BlockHeader) SignBytes() common.Bytes {
	r := BlockHeader{
		ChainID:     h.ChainID,
		Epoch:       h.Epoch,
		Height:      h.Height,
		Parent:      h.Parent,
		HCC:         h.HCC,
		TxHash:      h.TxHash,
		ReceiptHash: h.ReceiptHash,
		Bloom:       h.Bloom,
		StateHash:   h.StateHash,
		Timestamp:   h.Timestamp,
		Proposer:    h.Proposer,
	}
	raw, _ := rlp.EncodeToBytes(r)
	return raw
}

// SetSignature sets given signature in header.
func (h *BlockHeader) SetSignature(sig *crypto.Signature) {
	h.Signature = sig
}

// Validate checks the header is legitimate.
func (h *BlockHeader) Validate() result.Result {
	if h.Proposer.IsEmpty() {
		return result.Error("Proposer is not specified")
	}
	if h.Signature == nil || h.Signature.IsEmpty() {
		return result.Error("Block is not signed")
	}
	if !h.Signature.Verify(h.SignBytes(), h.Proposer) {
		return result.Error("Signature verification failed")
	}
	return result.OK
}

type BlockStatus byte

const (
	BlockStatusPending BlockStatus = BlockStatus(iota)
	BlockStatusCommitted
	BlockStatusFinalized
)

// ExtendedBlock is wrapper over Block, containing extra information related to the block.
type ExtendedBlock struct {
	*Block
	Children []common.Hash `json:"children"`
	Status   BlockStatus   `json:"status"`
}

// Hash of header.
func (eb *ExtendedBlock) Hash() common.Hash {
	if eb.Block == nil {
		return common.Hash{}
	}
	return eb.BlockHeader.Hash()
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
	return fmt.Sprintf("ExtendedBlock{Block: %v, Parent: %v, Children: %v, Status: %v}", eb.Block, eb.Parent.String(), children, eb.Status)
}

// ShortString returns a short string describing the block.
func (eb *ExtendedBlock) ShortString() string {
	return eb.Hash().String()
}
