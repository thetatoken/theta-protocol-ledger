package consensus

import (
	"fmt"

	"github.com/thetatoken/ukulele/blockchain"
)

// Proposal represents a proposal of a new block.
type Proposal struct {
	Block             blockchain.Block
	ProposerID        string
	CommitCertificate *blockchain.CommitCertificate
}

func (p Proposal) String() string {
	return fmt.Sprintf("Proposal{block: %v, proposer: %v, CC: %v}", p.Block, p.ProposerID, p.CommitCertificate)
}
