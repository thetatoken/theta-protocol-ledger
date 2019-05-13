package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/trie"
)

const (
	// MaxNumRegularTxsPerBlock represents the max number of regular transaction can be inclulded in one block
	MaxNumRegularTxsPerBlock int = 8192
)

var (
	EmptyRootHash = CalculateRootHash([]common.Bytes{})
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
	if b == nil {
		return "nil"
	}
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
	b.TxHash = CalculateRootHash(b.Txs)
	b.ReceiptHash = EmptyRootHash
}

// Validate checks the block is legitimate.
func (b *Block) Validate(chainID string) result.Result {
	res := b.BlockHeader.Validate(chainID)
	if res.IsError() {
		return res
	}
	if b.TxHash != CalculateRootHash(b.Txs) {
		return result.Error("TxHash does not match")
	}
	return result.OK
}

func CalculateRootHash(items []common.Bytes) common.Hash {
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
	HCC         CommitCertificate
	TxHash      common.Hash
	ReceiptHash common.Hash `json:"-"`
	Bloom       Bloom       `json:"-"`
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
		h.hash = h.calculateHash()
	}
	return h.hash
}

// UpdateHash recalculate hash of header.
func (h *BlockHeader) UpdateHash() common.Hash {
	if h == nil {
		return common.Hash{}
	}
	h.hash = h.calculateHash()
	return h.hash
}

func (h *BlockHeader) calculateHash() common.Hash {
	raw, _ := rlp.EncodeToBytes(h)
	return crypto.Keccak256Hash(raw)
}

func (h *BlockHeader) CalculateHash() common.Hash {
	return h.calculateHash()
}

func (h *BlockHeader) String() string {
	return fmt.Sprintf("{ChainID: %v, Epoch: %d, Hash: %v. Parent: %v, HCC: %s, Height: %v, TxHash: %v, StateHash: %v, Timestamp: %v, Proposer: %s}",
		h.ChainID, h.Epoch, h.Hash().Hex(), h.Parent.Hex(), h.HCC, h.Height, h.TxHash.Hex(), h.StateHash.Hex(), h.Timestamp, h.Proposer)
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
func (h *BlockHeader) Validate(chainID string) result.Result {
	if chainID != h.ChainID {
		return result.Error("ChainID mismatch")
	}
	if h.Parent.IsEmpty() {
		return result.Error("Parent is empty")
	}
	if h.HCC.BlockHash.IsEmpty() {
		return result.Error("HCC is empty")
	}
	if h.Timestamp == nil {
		return result.Error("Timestamp is missing")
	}
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

/*
Block status transitions:

+-------+          +-------+                          +-------------------+
|Pending+---+------>Invalid|                    +----->IndirectlyFinalized|
+-------+   |      +-------+                    |     +-------------------+
            |                                   |
            |      +-----+        +---------+   |     +-----------------+
            +------>Valid+-------->Committed+---+----->DirectlyFinalized|
                   +-----+        +---------+         +-----------------+

*/
const (
	BlockStatusPending BlockStatus = BlockStatus(iota)
	BlockStatusValid
	BlockStatusInvalid
	BlockStatusCommitted
	BlockStatusDirectlyFinalized
	BlockStatusIndirectlyFinalized
	BlockStatusTrusted
)

func (bs BlockStatus) IsPending() bool {
	return bs == BlockStatusPending
}

func (bs BlockStatus) IsCommitted() bool {
	return bs == BlockStatusCommitted
}

func (bs BlockStatus) IsFinalized() bool {
	return (bs == BlockStatusDirectlyFinalized) || (bs == BlockStatusIndirectlyFinalized) ||
		(bs == BlockStatusTrusted)
}

func (bs BlockStatus) IsDirectlyFinalized() bool {
	return bs == BlockStatusDirectlyFinalized
}

func (bs BlockStatus) IsIndirectlyFinalized() bool {
	return bs == BlockStatusIndirectlyFinalized
}

func (bs BlockStatus) IsTrusted() bool {
	return bs == BlockStatusTrusted
}

func (bs BlockStatus) IsInvalid() bool {
	return bs == BlockStatusInvalid
}

// IsValid returns whether block has been validated.
func (bs BlockStatus) IsValid() bool {
	return bs != BlockStatusPending && bs != BlockStatusInvalid
}

// func (bs BlockStatus) MarshalJSON() ([]byte, error) {
// 	if bs == BlockStatusPending {
// 		return []byte("\"pending\""), nil
// 	}
// 	if bs == BlockStatusValid {
// 		return []byte("\"valid\""), nil
// 	}
// 	if bs == BlockStatusInvalid {
// 		return []byte("\"invalid\""), nil
// 	}
// 	if bs == BlockStatusCommitted {
// 		return []byte("\"committed\""), nil
// 	}
// 	if bs == BlockStatusDirectlyFinalized {
// 		return []byte("\"directly_finalized\""), nil
// 	}
// 	if bs == BlockStatusIndirectlyFinalized {
// 		return []byte("\"indirectly_finalized\""), nil
// 	}
// 	return []byte("\"trusted\""), nil
// }

// ExtendedBlock is wrapper over Block, containing extra information related to the block.
type ExtendedBlock struct {
	*Block
	Children           []common.Hash `json:"children"`
	Status             BlockStatus   `json:"status"`
	HasValidatorUpdate bool
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

type ExtendedBlockInnerJSON ExtendedBlock

// MarshalJSON implements json.Marshaler
func (eb *ExtendedBlock) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ExtendedBlockInnerJSON
		Hash common.Hash
	}{
		ExtendedBlockInnerJSON: ExtendedBlockInnerJSON(*eb),
		Hash:                   eb.Hash(),
	})
}
