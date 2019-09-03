package core

import (
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/rlp"
)

//
// ------- AggregatedVotes ------- //
//

// AggregatedVotes represents votes on a block.
type AggregatedVotes struct {
	Block      common.Hash    // Hash of the block.
	Height     uint64         // Height of the block.
	Gcp        common.Hash    // Hash of guardian candidate pool.
	Multiplies []uint32       `rlp:"-"` // Multiplies of each signer.
	Signature  *bls.Signature `rlp:"-"` // Aggregated signiature.
}

// signBytes returns the bytes to be signed.
func (a *AggregatedVotes) signBytes() common.Bytes {
	b, _ := rlp.EncodeToBytes(a)
	return b
}

// Sign adds signer's signature.
func (a *AggregatedVotes) Sign(key *bls.SecretKey, signerIdx int) {
	if a.Multiplies[signerIdx] > 0 {
		// Already signed, do nothing.
		return
	}

	a.Multiplies[signerIdx] = 1
	a.Signature.Aggregate(key.Sign(a.signBytes(), bls.DomainGuardian))
}

//
// ------- GuardianCandidatePool ------- //
//

type GuardianCandidatePool struct {
	SortedGuardians []*Guardian
}

func (gcp *GuardianCandidatePool) Size() int {
	return len(gcp.SortedGuardians)
}

func (gcp *GuardianCandidatePool) ToPubKeys() []*bls.PublicKey {
	ret := make([]*bls.PublicKey, gcp.Size())
	for i, g := range gcp.SortedGuardians {
		ret[i] = g.Pubkey
	}
	return ret
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
