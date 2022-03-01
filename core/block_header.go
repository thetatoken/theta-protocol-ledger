package core

import (
	"fmt"
	"io"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

// BlockHeader contains the essential information of a block.
type BlockHeader interface {
	GetChainID() string
	GetEpoch() uint64
	GetHeight() uint64
	GetParent() common.Hash
	GetHCC() *CommitCertificate
	GetGuardianVotes() *AggregatedVotes
	GetEliteEdgeNodeVotes() *AggregatedEENVotes
	GetTxHash() common.Hash
	GetReceiptHash() common.Hash
	GetBloom() Bloom
	GetStateHash() common.Hash
	GetTimestamp() *big.Int
	GetProposer() common.Address
	GetSignature() *crypto.Signature

	SetChainID(chainID string)
	SetEpoch(epoch uint64)
	SetHeight(height uint64)
	SetParent(parent common.Hash)
	SetHCC(hcc CommitCertificate)
	SetGuardianVotes(guardianVotes *AggregatedVotes)
	SetEliteEdgeNodeVotes(eliteEdgeNodeVotes *AggregatedEENVotes)
	SetTxHash(txHash common.Hash)
	SetReceiptHash(receiptHash common.Hash)
	SetBloom(bloom Bloom)
	SetStateHash(stateHash common.Hash)
	SetTimestamp(timestamp *big.Int)
	SetProposer(proposer common.Address)

	EncodeRLP(w io.Writer) error
	DecodeRLP(stream *rlp.Stream) error
	Hash() common.Hash
	UpdateHash() common.Hash
	CalculateHash() common.Hash
	String() string
	SignBytes() common.Bytes
	SetSignature(sig *crypto.Signature)
	Validate(chainID string) result.Result
}

type ThetaBlockHeader struct {
	ChainID            string
	Epoch              uint64
	Height             uint64
	Parent             common.Hash
	HCC                CommitCertificate
	GuardianVotes      *AggregatedVotes    `rlp:"nil"` // Added in Theta2.0 fork.
	EliteEdgeNodeVotes *AggregatedEENVotes `rlp:"nil"` // Added in Theta3.0 fork.
	TxHash             common.Hash
	ReceiptHash        common.Hash `json:"-"`
	Bloom              Bloom       `json:"-"`
	StateHash          common.Hash
	Timestamp          *big.Int
	Proposer           common.Address
	Signature          *crypto.Signature

	hash common.Hash // Cache of calculated hash.
}

var _ rlp.Encoder = (*ThetaBlockHeader)(nil)

func (h *ThetaBlockHeader) GetChainID() string                         { return h.ChainID }
func (h *ThetaBlockHeader) GetEpoch() uint64                           { return h.Epoch }
func (h *ThetaBlockHeader) GetHeight() uint64                          { return h.Height }
func (h *ThetaBlockHeader) GetParent() common.Hash                     { return h.Parent }
func (h *ThetaBlockHeader) GetHCC() *CommitCertificate                 { return &h.HCC }
func (h *ThetaBlockHeader) GetGuardianVotes() *AggregatedVotes         { return h.GuardianVotes }
func (h *ThetaBlockHeader) GetEliteEdgeNodeVotes() *AggregatedEENVotes { return h.EliteEdgeNodeVotes }
func (h *ThetaBlockHeader) GetTxHash() common.Hash                     { return h.TxHash }
func (h *ThetaBlockHeader) GetReceiptHash() common.Hash                { return h.ReceiptHash }
func (h *ThetaBlockHeader) GetBloom() Bloom                            { return h.Bloom }
func (h *ThetaBlockHeader) GetStateHash() common.Hash                  { return h.StateHash }
func (h *ThetaBlockHeader) GetTimestamp() *big.Int                     { return h.Timestamp }
func (h *ThetaBlockHeader) GetProposer() common.Address                { return h.Proposer }
func (h *ThetaBlockHeader) GetSignature() *crypto.Signature            { return h.Signature }

func (h *ThetaBlockHeader) SetChainID(chainID string)    { h.ChainID = chainID }
func (h *ThetaBlockHeader) SetEpoch(epoch uint64)        { h.Epoch = epoch }
func (h *ThetaBlockHeader) SetHeight(height uint64)      { h.Height = height }
func (h *ThetaBlockHeader) SetParent(parent common.Hash) { h.Parent = parent }
func (h *ThetaBlockHeader) SetHCC(hcc CommitCertificate) { h.HCC = hcc }
func (h *ThetaBlockHeader) SetGuardianVotes(guardianVotes *AggregatedVotes) {
	h.GuardianVotes = guardianVotes
}
func (h *ThetaBlockHeader) SetEliteEdgeNodeVotes(eliteEdgeNodeVotes *AggregatedEENVotes) {
	h.EliteEdgeNodeVotes = eliteEdgeNodeVotes
}
func (h *ThetaBlockHeader) SetTxHash(txHash common.Hash)           { h.TxHash = txHash }
func (h *ThetaBlockHeader) SetReceiptHash(receiptHash common.Hash) { h.ReceiptHash = receiptHash }
func (h *ThetaBlockHeader) SetBloom(bloom Bloom)                   { h.Bloom = bloom }
func (h *ThetaBlockHeader) SetStateHash(stateHash common.Hash)     { h.StateHash = stateHash }
func (h *ThetaBlockHeader) SetTimestamp(timestamp *big.Int)        { h.Timestamp = timestamp }
func (h *ThetaBlockHeader) SetProposer(proposer common.Address)    { h.Proposer = proposer }

// EncodeRLP implements RLP Encoder interface.
func (h *ThetaBlockHeader) EncodeRLP(w io.Writer) error {
	if h == nil {
		return rlp.Encode(w, &ThetaBlockHeader{})
	}
	if h.Height < common.HeightEnableTheta2 {
		return rlp.Encode(w, []interface{}{
			h.ChainID,
			h.Epoch,
			h.Height,
			h.Parent,
			h.HCC,
			h.TxHash,
			h.ReceiptHash,
			h.Bloom,
			h.StateHash,
			h.Timestamp,
			h.Proposer,
			h.Signature,
		})
	}

	// Theta2.0 fork
	if h.Height >= common.HeightEnableTheta2 && h.Height < common.HeightEnableTheta3 {
		return rlp.Encode(w, []interface{}{
			h.ChainID,
			h.Epoch,
			h.Height,
			h.Parent,
			h.HCC,
			h.TxHash,
			h.ReceiptHash,
			h.Bloom,
			h.StateHash,
			h.Timestamp,
			h.Proposer,
			h.Signature,
			h.GuardianVotes,
		})
	}

	// Theta3.0 fork
	return rlp.Encode(w, []interface{}{
		h.ChainID,
		h.Epoch,
		h.Height,
		h.Parent,
		h.HCC,
		h.TxHash,
		h.ReceiptHash,
		h.Bloom,
		h.StateHash,
		h.Timestamp,
		h.Proposer,
		h.Signature,
		h.GuardianVotes,
		h.EliteEdgeNodeVotes,
	})
}

var _ rlp.Decoder = (*ThetaBlockHeader)(nil)

// DecodeRLP implements RLP Decoder interface.
func (h *ThetaBlockHeader) DecodeRLP(stream *rlp.Stream) error {
	_, err := stream.List()
	if err != nil {
		return err
	}

	err = stream.Decode(&h.ChainID)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Epoch)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Height)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Parent)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.HCC)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.TxHash)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.ReceiptHash)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Bloom)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.StateHash)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Timestamp)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Proposer)
	if err != nil {
		return err
	}

	err = stream.Decode(&h.Signature)
	if err != nil {
		return err
	}

	// Theta2.0 fork
	if h.Height >= common.HeightEnableTheta2 {
		raw, err := stream.Raw()
		if err != nil {
			return err
		}
		if common.Bytes2Hex(raw) == "c0" {
			h.GuardianVotes = nil
		} else {
			gvotes := &AggregatedVotes{}
			// err = stream.Decode(gvotes)
			rlp.DecodeBytes(raw, gvotes)
			if err != nil {
				return err
			}
			h.GuardianVotes = gvotes
		}
	}

	// Theta3.0 fork
	if h.Height >= common.HeightEnableTheta3 {
		raw, err := stream.Raw()
		if err != nil {
			return err
		}
		if common.Bytes2Hex(raw) == "c0" {
			h.EliteEdgeNodeVotes = nil
		} else {
			evotes := &AggregatedEENVotes{}
			rlp.DecodeBytes(raw, evotes)
			if err != nil {
				return err
			}
			h.EliteEdgeNodeVotes = evotes
		}
	}

	return stream.ListEnd()
}

// Hash of header.
func (h *ThetaBlockHeader) Hash() common.Hash {
	if h == nil {
		return common.Hash{}
	}
	if h.hash.IsEmpty() {
		h.hash = h.calculateHash()
	}
	return h.hash
}

// UpdateHash recalculate hash of header.
func (h *ThetaBlockHeader) UpdateHash() common.Hash {
	if h == nil {
		return common.Hash{}
	}
	h.hash = h.calculateHash()
	return h.hash
}

func (h *ThetaBlockHeader) calculateHash() common.Hash {
	raw, _ := rlp.EncodeToBytes(h)
	return crypto.Keccak256Hash(raw)
}

func (h *ThetaBlockHeader) CalculateHash() common.Hash {
	return h.calculateHash()
}

func (h *ThetaBlockHeader) String() string {
	return fmt.Sprintf("{ChainID: %v, Epoch: %d, Hash: %v. Parent: %v, HCC: %s, Height: %v, TxHash: %v, StateHash: %v, Timestamp: %v, Proposer: %s}",
		h.ChainID, h.Epoch, h.Hash().Hex(), h.GetParent().Hex(), h.HCC, h.Height, h.GetTxHash().Hex(), h.GetStateHash().Hex(), h.Timestamp, h.Proposer)
}

// SignBytes returns raw bytes to be signed.
func (h *ThetaBlockHeader) SignBytes() common.Bytes {
	old := h.GetSignature()
	h.SetSignature(nil)
	raw, _ := rlp.EncodeToBytes(h)
	h.SetSignature(old)
	return raw
}

// SetSignature sets given signature in header.
func (h *ThetaBlockHeader) SetSignature(sig *crypto.Signature) {
	h.SetSignature(sig)
}

// Validate checks the header is legitimate.
func (h *ThetaBlockHeader) Validate(chainID string) result.Result {
	if chainID != h.GetChainID() {
		return result.Error("ChainID mismatch")
	}
	if h.GetParent().IsEmpty() {
		return result.Error("Parent is empty")
	}
	if h.GetHCC().BlockHash.IsEmpty() {
		return result.Error("HCC is empty")
	}
	if h.GetTimestamp() == nil {
		return result.Error("Timestamp is missing")
	}
	if h.GetProposer().IsEmpty() {
		return result.Error("Proposer is not specified")
	}
	if h.GetSignature() == nil || h.GetSignature().IsEmpty() {
		return result.Error("Block is not signed")
	}
	if !h.GetSignature().Verify(h.SignBytes(), h.GetProposer()) {
		return result.Error("Signature verification failed")
	}
	return result.OK
}
