package consensus

import (
	"fmt"

	"github.com/thetatoken/ukulele/blockchain"
)

// Proposal represents a proposal of a new block.
type Proposal struct {
	block             blockchain.Block
	proposerID        string
	commitCertificate *blockchain.CommitCertificate
}

func (p Proposal) String() string {
	return fmt.Sprintf("Proposal{block: %v, proposer: %v, CC: %v}", p.block, p.proposerID, p.commitCertificate)
}
