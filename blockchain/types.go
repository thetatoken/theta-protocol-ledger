package blockchain

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

// CommitCertificate represents a commit made a majority of validators.
type CommitCertificate struct {
	Votes     *VoteSet
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
	Block *BlockHeader
	ID    string
}

func (v Vote) String() string {
	return fmt.Sprintf("Vote{block: %s, ID: %s}", v.Block.Hash, v.ID)
}

// VoteSet represents a set of votes on a proposal.
type VoteSet struct {
	votes []Vote
}

// NewVoteSet creates an instance of VoteSet.
func NewVoteSet() *VoteSet {
	return &VoteSet{}
}

// Copy creates a copy of this vote set.
func (s *VoteSet) Copy() *VoteSet {
	ret := &VoteSet{}
	for _, vote := range s.Votes() {
		ret.AddVote(vote)
	}
	return ret
}

// AddVote adds a vote to vote set.
func (s *VoteSet) AddVote(vote Vote) {
	s.votes = append(s.votes, vote)
}

// Size returns the number of votes in the vote set.
func (s *VoteSet) Size() int {
	return len(s.votes)
}

// Votes return a slice of votes in the vote set.
func (s *VoteSet) Votes() []Vote {
	return s.votes
}
