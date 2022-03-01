package core

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/trie"
)

const (
	// MaxNumRegularTxsPerBlock represents the max number of regular transaction can be inclulded in one block
	MaxNumRegularTxsPerBlock int = 512
)

var (
	EmptyRootHash    = CalculateRootHash([]common.Bytes{})
	SuicidedCodeHash = common.HexToHash("deaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddeaddead")
)

// Block represents a block in chain.
type Block struct {
	BlockHeader
	Txs []common.Bytes `json:"transactions"`
}

// NewBlock creates a new Block.
func NewBlock(blockHeader BlockHeader) *Block {
	if blockHeader == nil {
		panic("blockHeader is nil")
	}
	return &Block{BlockHeader: blockHeader}
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

var _ rlp.Encoder = (*Block)(nil)

// EncodeRLP implements RLP Encoder interface.
func (b *Block) EncodeRLP(w io.Writer) error {
	if b == nil {
		return rlp.Encode(w, &Block{})
	}
	return rlp.Encode(w, []interface{}{
		b.BlockHeader,
		b.Txs,
	})
}

// DecodeRLP implements RLP Decoder interface.
func (b *Block) DecodeRLP(stream *rlp.Stream) error {
	_, err := stream.List()
	if err != nil {
		return err
	}

	err = stream.Decode(b.BlockHeader)
	if err != nil {
		return err
	}

	txs := []common.Bytes{}
	err = stream.Decode(&txs)
	if err != nil {
		return err
	}
	b.Txs = txs

	return stream.ListEnd()
}

// AddTxs adds transactions to the block and update transaction root hash.
func (b *Block) AddTxs(txs []common.Bytes) {
	b.Txs = append(b.Txs, txs...)
	b.updateTxHash()
}

// updateTxHash calculate transaction root hash.
func (b *Block) updateTxHash() {
	b.SetTxHash(CalculateRootHash(b.Txs))
	b.SetReceiptHash(EmptyRootHash)
}

// Validate checks the block is legitimate.
func (b *Block) Validate(chainID string) result.Result {
	res := b.BlockHeader.Validate(chainID)
	if res.IsError() {
		return res
	}
	if b.GetTxHash() != CalculateRootHash(b.Txs) {
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
	BlockStatusDisposed
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
	return bs == BlockStatusInvalid || bs == BlockStatusDisposed
}

// IsValid returns whether block has been validated.
func (bs BlockStatus) IsValid() bool {
	return bs != BlockStatusPending && bs != BlockStatusInvalid && bs != BlockStatusDisposed
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
	return fmt.Sprintf("ExtendedBlock{Block: %v, Parent: %v, Children: %v, Status: %v}", eb.Block, eb.GetParent().String(), children, eb.Status)
}

// ShortString returns a short string describing the block.
func (eb *ExtendedBlock) ShortString() string {
	return eb.Hash().String()
}

// DecodeRLP implements RLP Decoder interface.
func (eb *ExtendedBlock) DecodeRLP(stream *rlp.Stream) error {
	_, err := stream.List()
	if err != nil {
		return err
	}

	b := &Block{}
	err = stream.Decode(b)
	if err != nil {
		return err
	}
	eb.Block = b

	children := []common.Hash{}
	err = stream.Decode(&children)
	if err != nil {
		return err
	}
	eb.Children = children

	var status byte
	err = stream.Decode(&status)
	if err != nil {
		return err
	}
	eb.Status = BlockStatus(status)

	var hasValidatorUpdate bool
	err = stream.Decode(&hasValidatorUpdate)
	if err != nil {
		return err
	}
	eb.HasValidatorUpdate = hasValidatorUpdate

	return stream.ListEnd()
}

// EncodeRLP implements RLP Encoder interface.
func (eb *ExtendedBlock) EncodeRLP(w io.Writer) error {
	if eb == nil {
		return rlp.Encode(w, &ExtendedBlock{})
	}
	return rlp.Encode(w, []interface{}{
		eb.Block,
		eb.Children,
		eb.Status,
		eb.HasValidatorUpdate,
	})
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
