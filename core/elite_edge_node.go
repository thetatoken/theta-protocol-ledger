package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto"
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
	CheckpointHash common.Hash
}

// signBytes returns the bytes to be signed.
func (e *EENVote) signBytes() common.Bytes {
	tmp := &EENBlsSigMsg{
		CheckpointHash: e.Block,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// Validate verifies the vote.
func (e *EENVote) Validate(eenp *EliteEdgeNodePool) result.Result {
	if e.Signature == nil {
		return result.Error("signature cannot be nil")
	}
	eenIdx := eenp.IndexWithHolderAddress(e.Address)
	if eenIdx < 0 {
		return result.Error("cannot find elite edge node %v", e.Address)
	}
	eenBLSPubkey := eenp.SortedEliteEdgeNodes[eenIdx].Pubkey
	if !e.Signature.Verify(e.signBytes(), eenBLSPubkey) {
		return result.Error("signature verification failed")
	}
	return result.OK
}

//
// ------- AggregatedEENVotes ------- //
//

// AggregatedEENVotes represents the aggregated elite edge node votes on a block.
type AggregatedEENVotes struct {
	Block      common.Hash    // Hash of the block.
	Multiplies []uint32       // Multiplies of each signer.
	Signature  *bls.Signature // Aggregated signiature.
}

func NewAggregatedEENVotes(block common.Hash, eenp *EliteEdgeNodePool) *AggregatedEENVotes {
	return &AggregatedEENVotes{
		Block:      block,
		Multiplies: make([]uint32, eenp.WithStake().Len()),
		Signature:  bls.NewAggregateSignature(),
	}
}

func (a *AggregatedEENVotes) String() string {
	return fmt.Sprintf("AggregatedEENVotes{Block: %s, Multiplies: %v}", a.Block.Hex(), a.Multiplies)
}

// signBytes returns the bytes to be signed.
func (a *AggregatedEENVotes) signBytes() common.Bytes {
	tmp := &AggregatedEENVotes{
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
func (a *AggregatedEENVotes) Validate(eenp *EliteEdgeNodePool) result.Result {
	if len(a.Multiplies) != eenp.WithStake().Len() {
		return result.Error("multiplies size %d is not equal to eenp size %d", len(a.Multiplies), eenp.WithStake().Len())
	}
	if a.Signature == nil {
		return result.Error("signature cannot be nil")
	}
	pubKeys := eenp.WithStake().PubKeys()
	aggPubkey := bls.AggregatePublicKeysVec(pubKeys, a.Multiplies)
	if !a.Signature.Verify(a.signBytes(), aggPubkey) {
		return result.Error("signature verification failed")
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

//
// ------- EliteEdgeNodePool ------- //
//

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

type EliteEdgeNodePool struct {
	SortedEliteEdgeNodes []*EliteEdgeNode // Elite edge nodes sorted by holder address.
}

// NewEliteEdgeNodePool creates a new instance of EliteEdgeNodePool.
func NewEliteEdgeNodePool() *EliteEdgeNodePool {
	return &EliteEdgeNodePool{
		SortedEliteEdgeNodes: []*EliteEdgeNode{},
	}
}

// Add inserts elite edge node into the pool; returns false if guardian is already added.
func (eenp *EliteEdgeNodePool) Add(een *EliteEdgeNode) bool {
	k := sort.Search(eenp.Len(), func(i int) bool {
		return bytes.Compare(eenp.SortedEliteEdgeNodes[i].Holder.Bytes(), een.Holder.Bytes()) >= 0
	})

	if k == eenp.Len() {
		eenp.SortedEliteEdgeNodes = append(eenp.SortedEliteEdgeNodes, een)
		return true
	}

	// Elite edge node is already added.
	if eenp.SortedEliteEdgeNodes[k].Holder == een.Holder {
		return false
	}
	eenp.SortedEliteEdgeNodes = append(eenp.SortedEliteEdgeNodes, nil)
	copy(eenp.SortedEliteEdgeNodes[k+1:], eenp.SortedEliteEdgeNodes[k:])
	eenp.SortedEliteEdgeNodes[k] = een
	return true
}

// Remove removes an elite edge node from the pool; returns false if guardian is not found.
func (eenp *EliteEdgeNodePool) Remove(een common.Address) bool {
	k := sort.Search(eenp.Len(), func(i int) bool {
		return bytes.Compare(eenp.SortedEliteEdgeNodes[i].Holder.Bytes(), een.Bytes()) >= 0
	})

	if k == eenp.Len() || bytes.Compare(eenp.SortedEliteEdgeNodes[k].Holder.Bytes(), een.Bytes()) != 0 {
		return false
	}
	eenp.SortedEliteEdgeNodes = append(eenp.SortedEliteEdgeNodes[:k], eenp.SortedEliteEdgeNodes[k+1:]...)
	return true
}

// Contains checks if given address is in the pool.
func (eenp *EliteEdgeNodePool) Contains(een common.Address) bool {
	k := sort.Search(eenp.Len(), func(i int) bool {
		return bytes.Compare(eenp.SortedEliteEdgeNodes[i].Holder.Bytes(), een.Bytes()) >= 0
	})

	if k == eenp.Len() || eenp.SortedEliteEdgeNodes[k].Holder != een {
		return false
	}
	return true
}

// WithStake returns a new pool with withdrawn elite edge nodes filtered out.
func (eenp *EliteEdgeNodePool) WithStake() *EliteEdgeNodePool {
	ret := NewEliteEdgeNodePool()
	for _, een := range eenp.SortedEliteEdgeNodes {
		// Skip if guardian dons't have non-withdrawn stake
		hasStake := false
		for _, stake := range een.Stakes {
			if !stake.Withdrawn {
				hasStake = true
				break
			}
		}
		if !hasStake {
			continue
		}

		ret.Add(een)
	}
	return ret
}

// IndexWithHolderAddress returns index of a stake holder address in the pool. Returns -1 if not found.
func (eenp *EliteEdgeNodePool) IndexWithHolderAddress(addr common.Address) int {
	for i, een := range eenp.SortedEliteEdgeNodes {
		if een.Holder == addr {
			return i
		}
	}
	return -1
}

// Index returns index of a public key in the pool. Returns -1 if not found.
func (eenp *EliteEdgeNodePool) Index(pubkey *bls.PublicKey) int {
	for i, een := range eenp.SortedEliteEdgeNodes {
		if pubkey.Equals(een.Pubkey) {
			return i
		}
	}
	return -1
}

// PubKeys exports guardians' public keys.
func (eenp *EliteEdgeNodePool) PubKeys() []*bls.PublicKey {
	ret := make([]*bls.PublicKey, eenp.Len())
	for i, een := range eenp.SortedEliteEdgeNodes {
		ret[i] = een.Pubkey
	}
	return ret
}

// Implements sort.Interface for Guardians based on
// the Address field.
func (eenp *EliteEdgeNodePool) Len() int {
	return len(eenp.SortedEliteEdgeNodes)
}
func (eenp *EliteEdgeNodePool) Swap(i, j int) {
	eenp.SortedEliteEdgeNodes[i], eenp.SortedEliteEdgeNodes[j] = eenp.SortedEliteEdgeNodes[j], eenp.SortedEliteEdgeNodes[i]
}
func (eenp *EliteEdgeNodePool) Less(i, j int) bool {
	return bytes.Compare(eenp.SortedEliteEdgeNodes[i].Holder.Bytes(), eenp.SortedEliteEdgeNodes[j].Holder.Bytes()) < 0
}

// Hash calculates the hash of elite edge node pool.
func (eenp *EliteEdgeNodePool) Hash() common.Hash {
	raw, err := rlp.EncodeToBytes(eenp)
	if err != nil {
		logger.Panic(err)
	}
	return crypto.Keccak256Hash(raw)
}

func (eenp *EliteEdgeNodePool) DepositStake(source common.Address, holder common.Address, amount *big.Int, pubkey *bls.PublicKey, blockHeight uint64) (err error) {
	minEliteEdgeNodeStake := MinEliteEdgeNodeStakeDeposit
	maxEliteEdgeNodeStake := MaxEliteEdgeNodeStakeDeposit
	if amount.Cmp(minEliteEdgeNodeStake) < 0 {
		return fmt.Errorf("Elite edge node staking amount below the lower limit: %v", amount)
	}
	if amount.Cmp(maxEliteEdgeNodeStake) > 0 {
		return fmt.Errorf("Elite edge node staking amount above the upper limit: %v", amount)
	}

	matchedHolderFound := false
	for _, een := range eenp.SortedEliteEdgeNodes {
		if een.Holder == holder {
			currentStake := een.TotalStake()
			expectedStake := big.NewInt(0).Add(currentStake, amount)
			if expectedStake.Cmp(maxEliteEdgeNodeStake) > 0 {
				return fmt.Errorf("Elite edge node stake would exceed the cap: %v", expectedStake)
			}

			matchedHolderFound = true
			err = een.depositStake(source, amount)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		newEliteEdgeNode := &EliteEdgeNode{
			StakeHolder: newStakeHolder(holder, []*Stake{newStake(source, amount)}),
			Pubkey:      pubkey,
		}
		eenp.Add(newEliteEdgeNode)
	}
	return nil
}

func (eenp *EliteEdgeNodePool) WithdrawStake(source common.Address, holder common.Address, currentHeight uint64) error {
	matchedHolderFound := false
	for _, een := range eenp.SortedEliteEdgeNodes {
		if een.Holder == holder {
			matchedHolderFound = true
			err := een.withdrawStake(source, currentHeight)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		return fmt.Errorf("No matched stake holder address found: %v", holder)
	}
	return nil
}

func (eenp *EliteEdgeNodePool) ReturnStakes(currentHeight uint64) []*Stake {
	returnedStakes := []*Stake{}

	// need to iterate in the reverse order, since we may delete elemements
	// from the slice while iterating through it
	for cidx := eenp.Len() - 1; cidx >= 0; cidx-- {
		een := eenp.SortedEliteEdgeNodes[cidx]
		numStakeSources := len(een.Stakes)
		for sidx := numStakeSources - 1; sidx >= 0; sidx-- { // similar to the outer loop, need to iterate in the reversed order
			stake := een.Stakes[sidx]
			if (stake.Withdrawn) && (currentHeight >= stake.ReturnHeight) {
				logger.Printf("Stake to be returned: source = %v, amount = %v", stake.Source, stake.Amount)
				source := stake.Source
				returnedStake, err := een.returnStake(source, currentHeight)
				if err != nil {
					logger.Errorf("Failed to return stake: %v, error: %v", source, err)
					continue
				}
				returnedStakes = append(returnedStakes, returnedStake)
			}
		}

		if len(een.Stakes) == 0 { // the candidate's stake becomes zero, no need to keep track of the candidate anymore
			eenp.Remove(een.Holder)
		}
	}
	return returnedStakes
}

//
// ------- EliteEdgeNode ------- //
//

type EliteEdgeNode struct {
	*StakeHolder
	Pubkey *bls.PublicKey `json:"-"`
}

func (een *EliteEdgeNode) String() string {
	return fmt.Sprintf("{holder: %v, pubkey: %v, stakes :%v}", een.Holder, een.Pubkey.String(), een.Stakes)
}
