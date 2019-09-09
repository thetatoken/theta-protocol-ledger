package core

import (
	"bytes"
	"errors"
	"sort"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/rlp"
)

//
// ------- AggregatedVotes ------- //
//

// AggregatedVotes represents votes on a block.
type AggregatedVotes struct {
	Block      common.Hash    // Hash of the block.
	Gcp        common.Hash    // Hash of guardian candidate pool.
	Multiplies []uint32       `rlp:"-"` // Multiplies of each signer.
	Signature  *bls.Signature `rlp:"-"` // Aggregated signiature.
}

func NewAggregateVotes(block common.Hash, gcp *GuardianCandidatePool) *AggregatedVotes {
	return &AggregatedVotes{
		Block:      block,
		Gcp:        gcp.Hash(),
		Multiplies: make([]uint32, gcp.Len()),
		Signature:  bls.NewAggregateSignature(),
	}
}

// signBytes returns the bytes to be signed.
func (a *AggregatedVotes) signBytes() common.Bytes {
	tmp := &AggregatedVotes{
		Block: a.Block,
		Gcp:   a.Gcp,
	}
	b, _ := rlp.EncodeToBytes(tmp)
	return b
}

// Sign adds signer's signature. Returns false if signer has already signed.
func (a *AggregatedVotes) Sign(key *bls.SecretKey, signerIdx int) bool {
	if a.Multiplies[signerIdx] > 0 {
		// Already signed, do nothing.
		return false
	}

	a.Multiplies[signerIdx] = 1
	a.Signature.Aggregate(key.Sign(a.signBytes()))
	return true
}

// Merge creates a new aggregation that combines two vote sets.
func (a *AggregatedVotes) Merge(b *AggregatedVotes) (*AggregatedVotes, error) {
	if a.Block != b.Block || a.Gcp != b.Gcp {
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
		return nil, errors.New("The other vote is a subset of current vote")
	}
	newSig := a.Signature.Copy()
	newSig.Aggregate(b.Signature)
	return &AggregatedVotes{
		Block:      a.Block,
		Gcp:        a.Gcp,
		Multiplies: newMultiplies,
		Signature:  newSig,
	}, nil
}

// Validate verifies the voteset.
func (a *AggregatedVotes) Validate(gcp *GuardianCandidatePool) result.Result {
	if gcp.Hash() != a.Gcp {
		return result.Error("gcp hash mismatch: gcp.Hash(): %s, vote.Gcp: %s", gcp.Hash().Hex(), a.Gcp.Hex())
	}
	if len(a.Multiplies) != gcp.Len() {
		return result.Error("multiplies size %d is not equal to gcp size %d", len(a.Multiplies), gcp.Len())
	}
	pubKeys := gcp.PubKeys()
	aggPubkey := bls.AggregatePublicKeysVec(pubKeys, a.Multiplies)
	if !a.Signature.Verify(a.signBytes(), aggPubkey) {
		return result.Error("signature verification failed")
	}
	return result.OK
}

//
// ------- GuardianCandidatePool ------- //
//

type GuardianCandidatePool struct {
	SortedGuardians []*Guardian
}

// NewGuardianCandidatePool creates a new instance of GuardianCandidatePool.
func NewGuardianCandidatePool() *GuardianCandidatePool {
	return &GuardianCandidatePool{
		SortedGuardians: []*Guardian{},
	}
}

// Add inserts guardian into the pool; returns false if guradian is already added.
func (gcp *GuardianCandidatePool) Add(g *Guardian) bool {
	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedGuardians[i].Holder.Bytes(), g.Holder.Bytes()) >= 0
	})

	if k == gcp.Len() {
		gcp.SortedGuardians = append(gcp.SortedGuardians, g)
		return true
	}

	// Guardian is already added.
	if gcp.SortedGuardians[k].Holder == g.Holder {
		return false
	}
	gcp.SortedGuardians = append(gcp.SortedGuardians, nil)
	copy(gcp.SortedGuardians[k+1:], gcp.SortedGuardians[k:])
	gcp.SortedGuardians[k] = g
	return true
}

// Remove removes a guardian from the pool; returns false if guradian is not found.
func (gcp *GuardianCandidatePool) Remove(g common.Address) bool {
	k := sort.Search(gcp.Len(), func(i int) bool {
		return bytes.Compare(gcp.SortedGuardians[i].Holder.Bytes(), g.Bytes()) >= 0
	})

	if k == gcp.Len() || bytes.Compare(gcp.SortedGuardians[k].Holder.Bytes(), g.Bytes()) != 0 {
		return false
	}
	gcp.SortedGuardians = append(gcp.SortedGuardians[:k], gcp.SortedGuardians[k+1:]...)
	return true
}

// PubKeys exports guardians' public keys.
func (gcp *GuardianCandidatePool) PubKeys() []*bls.PublicKey {
	ret := make([]*bls.PublicKey, gcp.Len())
	for i, g := range gcp.SortedGuardians {
		ret[i] = g.Pubkey
	}
	return ret
}

// Implements sort.Interface for Guardians based on
// the Address field.
func (gcp *GuardianCandidatePool) Len() int {
	return len(gcp.SortedGuardians)
}
func (gcp *GuardianCandidatePool) Swap(i, j int) {
	gcp.SortedGuardians[i], gcp.SortedGuardians[j] = gcp.SortedGuardians[j], gcp.SortedGuardians[i]
}
func (gcp *GuardianCandidatePool) Less(i, j int) bool {
	return bytes.Compare(gcp.SortedGuardians[i].Holder.Bytes(), gcp.SortedGuardians[j].Holder.Bytes()) < 0
}

// Hash calculates the hash of gcp.
func (gcp *GuardianCandidatePool) Hash() common.Hash {
	raw, err := rlp.EncodeToBytes(gcp)
	if err != nil {
		panic(err)
	}
	return crypto.Keccak256Hash(raw)
}

//
// ------- Guardian ------- //
//

type Guardian struct {
	Holder common.Address
	Pubkey *bls.PublicKey
	Pop    *bls.Signature // Proof of possesion of the corresponding BLS private key.
	Stakes []*Stake
}
