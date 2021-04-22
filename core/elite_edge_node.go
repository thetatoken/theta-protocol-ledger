package core

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/rlp"
)

//
// ------- EENVote ------- //
//

// EENVote represents the vote for a block from an elite edge node.
type EENVote struct {
	Block     common.Hash    // Hash of the block.
	Address   common.Address // Address of the edge node.
	Signature *bls.Signature // Aggregated signiature.
}

type EENBlsSigMsg struct {
	Block common.Hash
}

// signBytes returns the bytes to be signed.
func (e *EENVote) signBytes() common.Bytes {
	tmp := &EENBlsSigMsg{
		Block: e.Block,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// Validate verifies the vote.
func (e *EENVote) Validate(eenBLSPubkey *bls.PublicKey) result.Result {
	if e.Signature == nil {
		return result.Error("signature cannot be nil")
	}
	if !e.Signature.Verify(e.signBytes(), eenBLSPubkey) {
		return result.Error("elite edge node vote signature validation failed")
	}
	return result.OK
}

func (e *EENVote) String() string {
	return fmt.Sprintf("EENVote{Block: %s, Address: %v, Signature: %v}",
		e.Block.Hex(), e.Address, e.signBytes())
}

//
// ------- AggregatedEENVotes ------- //
//

// AggregatedEENVotes represents the aggregated elite edge node votes on a block.
type AggregatedEENVotes struct {
	Block      common.Hash    // Hash of the block.
	Multiplies []uint32       // Multiplies of each signer.
	Signature  *bls.Signature // Aggregated signature.
}

func NewAggregatedEENVotes(block common.Hash, eenpWithStake *EliteEdgeNodePool) *AggregatedEENVotes {
	return &AggregatedEENVotes{
		Block:      block,
		Multiplies: make([]uint32, eenpWithStake.Len()),
		Signature:  bls.NewAggregateSignature(),
	}
}

func (a *AggregatedEENVotes) String() string {
	return fmt.Sprintf("AggregatedEENVotes{Block: %s, Multiplies: %v}", a.Block.Hex(), a.Multiplies)
}

// signBytes returns the bytes to be signed.
func (a *AggregatedEENVotes) signBytes() common.Bytes {
	// tmp := &AggregatedEENVotes{
	// 	Block: a.Block,
	// }
	tmp := &EENBlsSigMsg{
		Block: a.Block,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// // Sign adds signer's signature. Returns false if signer has already signed.
// func (a *AggregatedEENVotes) Sign(key *bls.SecretKey, signerIdx int) bool {
// 	if a.Multiplies[signerIdx] > 0 {
// 		// Already signed, do nothing.
// 		return false
// 	}

// 	a.Multiplies[signerIdx] = 1
// 	a.Signature.Aggregate(key.Sign(a.signBytes()))
// 	return true
// }

// Merge creates a new aggregation that combines two vote sets. Returns nil, nil if input vote
// is a subset of current vote.
func (a *AggregatedEENVotes) Merge(b *AggregatedEENVotes) (*AggregatedEENVotes, error) {
	if a.Block != b.Block {
		return nil, errors.New("Cannot merge incompatible votes")
	}
	newMultiplies := make([]uint32, len(a.Multiplies))
	isSubset := true
	for i := 0; i < len(a.Multiplies); i++ {
		newMultiplies[i] = a.Multiplies[i] + b.Multiplies[i]
		if newMultiplies[i] < a.Multiplies[i] || newMultiplies[i] < b.Multiplies[i] {
			return nil, errors.New("Signiature multipliers overflowed")
		}
		if a.Multiplies[i] == 0 && b.Multiplies[i] != 0 {
			isSubset = false
		}
	}
	if isSubset {
		// The other vote is a subset of current vote
		return nil, nil
	}
	newSig := a.Signature.Copy()
	newSig.Aggregate(b.Signature)
	return &AggregatedEENVotes{
		Block:      a.Block,
		Multiplies: newMultiplies,
		Signature:  newSig,
	}, nil
}

// Abs returns the number of voted elite edge nodes in the vote
func (a *AggregatedEENVotes) Abs() int {
	ret := 0
	for i := 0; i < len(a.Multiplies); i++ {
		if a.Multiplies[i] != 0 {
			ret += 1
		}
	}
	return ret
}

// Pick selects better vote from two votes.
func (a *AggregatedEENVotes) Pick(b *AggregatedEENVotes) (*AggregatedEENVotes, error) {
	if a.Block != b.Block {
		return nil, errors.New("Cannot compare incompatible votes")
	}
	if b.Abs() > a.Abs() {
		return b, nil
	}
	return a, nil
}

// Validate verifies the voteset.
func (a *AggregatedEENVotes) Validate(eenpWithStake *EliteEdgeNodePool) result.Result {
	if len(a.Multiplies) != eenpWithStake.Len() {
		return result.Error("multiplies size %d is not equal to eenp size %d", len(a.Multiplies), eenpWithStake.Len())
	}
	if a.Signature == nil {
		return result.Error("signature cannot be nil")
	}
	pubKeys := eenpWithStake.PubKeys()
	aggPubkey := bls.AggregatePublicKeysVec(pubKeys, a.Multiplies)
	if !a.Signature.Verify(a.signBytes(), aggPubkey) {
		return result.Error("aggregated elite edge node votes signature verification failed")
	}
	return result.OK
}

// Copy clones the aggregated votes
func (a *AggregatedEENVotes) Copy() *AggregatedEENVotes {
	clone := &AggregatedEENVotes{
		Block: a.Block,
	}
	if a.Multiplies != nil {
		clone.Multiplies = make([]uint32, len(a.Multiplies))
		copy(clone.Multiplies, a.Multiplies)
	}
	if a.Signature != nil {
		clone.Signature = a.Signature.Copy()
	}

	return clone
}

var (
	MinEliteEdgeNodeStakeDeposit *big.Int
	MaxEliteEdgeNodeStakeDeposit *big.Int
)

func init() {
	// Each elite edge node stake deposit needs to be at least 10,000 TFuel
	MinEliteEdgeNodeStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(10000), new(big.Int).SetUint64(1e18))

	// Each elite edge node stake deposit should not exceed 500,000 TFuel
	MaxEliteEdgeNodeStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(500000), new(big.Int).SetUint64(1e18))
}

//
// ------- EliteEdgeNode ------- //
//

type EliteEdgeNode struct {
	*StakeHolder
	Pubkey *bls.PublicKey `json:"-"`
}

func NewEliteEdgeNode(stakeHolder *StakeHolder, pubkey *bls.PublicKey) *EliteEdgeNode {
	return &EliteEdgeNode{
		StakeHolder: stakeHolder,
		Pubkey:      pubkey,
	}
}

func (een *EliteEdgeNode) String() string {
	return fmt.Sprintf("{holder: %v, pubkey: %v, stakes :%v}", een.Holder, een.Pubkey.String(), een.Stakes)
}

func (een *EliteEdgeNode) DepositStake(source common.Address, amount *big.Int) error {
	return een.StakeHolder.depositStake(source, amount)
}

func (een *EliteEdgeNode) WithdrawStake(source common.Address, currentHeight uint64) (*Stake, error) {
	return een.StakeHolder.withdrawStake(source, currentHeight)
}

func (een *EliteEdgeNode) ReturnStake(source common.Address, currentHeight uint64) (*Stake, error) {
	return een.StakeHolder.returnStake(source, currentHeight)
}

//
// ------- EliteEdgeNodePool ------- //
//

type EliteEdgeNodePool interface {
	Contains(eenAddr common.Address) bool
	Get(eenAddr common.Address) *EliteEdgeNode
	Upsert(een *EliteEdgeNode)
	GetAll(withstake bool) []*EliteEdgeNode
	DepositStake(source common.Address, holder common.Address, amount *big.Int, pubkey *bls.PublicKey, blockHeight uint64) (err error)
	WithdrawStake(source common.Address, holder common.Address, currentHeight uint64) (*Stake, error)
}
