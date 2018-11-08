package core

import (
	"bytes"
	"fmt"
	"io"
	"sort"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
)

// Proposal represents a proposal of a new block.
type Proposal struct {
	Block             *Block `rlp:"nil"`
	ProposerID        common.Address
	CommitCertificate *CommitCertificate `rlp:"nil"`
}

func (p Proposal) String() string {
	return fmt.Sprintf("Proposal{block: %v, proposer: %v, CC: %v}", p.Block, p.ProposerID, p.CommitCertificate)
}

// CommitCertificate represents a commit made a majority of validators.
type CommitCertificate struct {
	Votes     *VoteSet `rlp:"nil"`
	BlockHash common.Hash
}

// Copy creates a copy of this commit certificate.
func (cc *CommitCertificate) Copy() *CommitCertificate {
	ret := &CommitCertificate{
		BlockHash: cc.BlockHash,
	}
	if cc.Votes != nil {
		ret.Votes = cc.Votes.Copy()
	}
	return ret
}

func (cc *CommitCertificate) String() string {
	return fmt.Sprintf("CC{block: %v, votes: %v}", cc.BlockHash, cc.Votes)
}

// IsValid checks if a CommitCertificate is valid.
func (cc *CommitCertificate) IsValid() bool {
	return cc.Votes.Size() > 0
}

// Vote represents a vote on a block by a validaor.
type Vote struct {
	Block     common.Hash       // Hash of the tip as seen by the voter.
	Epoch     uint64            // Voter's current epoch. It doesn't need to equal the epoch in the block above.
	ID        common.Address    // Voter's address.
	Signature *crypto.Signature `rlp:"nil"`
}

func (v Vote) String() string {
	return fmt.Sprintf("Vote{ID: %s, block: %s,  Epoch: %v}", v.ID, v.Block, v.Epoch)
}

// SignBytes returns raw bytes to be signed.
func (v Vote) SignBytes() common.Bytes {
	vv := Vote{
		Block: v.Block,
		Epoch: v.Epoch,
		ID:    v.ID,
	}
	raw, _ := rlp.EncodeToBytes(vv)
	return raw
}

// SetSignature sets given signature in vote.
func (v Vote) SetSignature(sig *crypto.Signature) {
	v.Signature = sig
}

// Validate checks the vote is legitimate.
func (v Vote) Validate() result.Result {
	if v.ID.IsEmpty() {
		return result.Error("Voter is not specified")
	}
	if v.Signature.IsEmpty() {
		return result.Error("Vote is not signed")
	}
	return v.Signature.VerifyBytes(v.SignBytes(), v.ID)
}

// VoteSet represents a set of votes on a proposal.
type VoteSet struct {
	votes map[string]Vote // Voter ID to vote
}

// NewVoteSet creates an instance of VoteSet.
func NewVoteSet() *VoteSet {
	return &VoteSet{
		votes: make(map[string]Vote),
	}
}

// Copy creates a copy of this vote set.
func (s *VoteSet) Copy() *VoteSet {
	ret := NewVoteSet()
	for _, vote := range s.Votes() {
		ret.AddVote(vote)
	}
	return ret
}

// AddVote adds a vote to vote set.
func (s *VoteSet) AddVote(vote Vote) {
	s.votes[vote.ID.Hex()] = vote
}

// Size returns the number of votes in the vote set.
func (s *VoteSet) Size() int {
	return len(s.votes)
}

// Votes return a slice of votes in the vote set.
func (s *VoteSet) Votes() []Vote {
	ret := make([]Vote, 0, len(s.votes))
	for _, v := range s.votes {
		ret = append(ret, v)
	}
	sort.Sort(VoteByID(ret))
	return ret
}

// Validate checks the vote set is legitimate.
func (s *VoteSet) Validate() result.Result {
	for _, vote := range s.votes {
		if vote.Validate().IsError() {
			return result.Error("Contains invalid vote: %s", vote.String())
		}
	}
	return result.OK
}

func (s *VoteSet) String() string {
	return fmt.Sprintf("%v", s.Votes())
}

var _ rlp.Encoder = (*VoteSet)(nil)

// EncodeRLP implements RLP Encoder interface.
func (s *VoteSet) EncodeRLP(w io.Writer) error {
	if s == nil {
		return rlp.Encode(w, []Vote{})
	}
	return rlp.Encode(w, s.Votes())
}

var _ rlp.Decoder = (*VoteSet)(nil)

// DecodeRLP implements RLP Decoder interface.
func (s *VoteSet) DecodeRLP(stream *rlp.Stream) error {
	votes := []Vote{}
	err := stream.Decode(&votes)
	if err != nil {
		return err
	}
	s.votes = make(map[string]Vote)
	for _, v := range votes {
		s.votes[v.ID.Hex()] = v
	}
	return nil
}

// VoteByID implements sort.Interface for []Vote based on Voter's ID.
type VoteByID []Vote

func (a VoteByID) Len() int           { return len(a) }
func (a VoteByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a VoteByID) Less(i, j int) bool { return bytes.Compare(a[i].ID.Bytes(), a[j].ID.Bytes()) < 0 }
