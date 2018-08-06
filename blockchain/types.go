package blockchain

import (
	"fmt"

	"github.com/thetatoken/ukulele/types"
)

// CommitCertificate represents a commit made a majority of validators.
type CommitCertificate struct {
	Votes     []Vote
	BlockHash types.Bytes
}

func (cc *CommitCertificate) String() string {
	return fmt.Sprintf("CC{block: %v, votes: %v}", cc.BlockHash, cc.Votes)
}

// IsValid checks if a CommitCertificate is valid.
func (cc *CommitCertificate) IsValid() bool {
	return len(cc.Votes) > 0
}

// Vote represents a vote on a block by a validaor.
type Vote struct {
	Block *BlockHeader
	ID    string
}

func (v Vote) String() string {
	return fmt.Sprintf("Vote{block: %s, ID: %s}", v.Block.Hash, v.ID)
}
