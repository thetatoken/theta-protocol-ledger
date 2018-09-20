package core

import (
	"fmt"
	"io"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/rlp"
)

// Proposal represents a proposal of a new block.
type Proposal struct {
	Block             Block
	ProposerID        string
	CommitCertificate *CommitCertificate `rlp:"nil"`
}

func (p Proposal) String() string {
	return fmt.Sprintf("Proposal{block: %v, proposer: %v, CC: %v}", p.Block, p.ProposerID, p.CommitCertificate)
}

// CommitCertificate represents a commit made a majority of validators.
type CommitCertificate struct {
	Votes     *VoteSet `rlp:"nil"`
	BlockHash common.Bytes
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
	Block *BlockHeader `rlp:"nil"`
	ID    string
	Epoch uint32
}

func (v Vote) String() string {
	return fmt.Sprintf("Vote{block: %s, ID: %s, Epoch: %v}", v.Block, v.ID, v.Epoch)
}

// VoteSet represents a set of votes on a proposal.
type VoteSet struct {
	votes map[string]Vote
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
	s.votes[vote.ID] = vote
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
	return ret
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
		s.votes[v.ID] = v
	}
	return nil
}
