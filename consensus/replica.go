package consensus

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/blockchain"
)

// ReplicaStrategy defines the replica interface that is used by DefaultEngine.
type ReplicaStrategy interface {
	Init(e *DefaultEngine)
	HandleProposal(p Proposal)
	EnterNewHeight(newHeight uint32)
}

var _ ReplicaStrategy = &DefaultReplicaStrategy{}

// DefaultReplicaStrategy is the default implementation of the ReplicaStrategy interface.
type DefaultReplicaStrategy struct {
	engine *DefaultEngine
}

// Init implements ReplicaStrategy interface.
func (rs *DefaultReplicaStrategy) Init(e *DefaultEngine) {
	rs.engine = e
}

// EnterNewHeight implements ReplicaStrategy interface.
func (rs *DefaultReplicaStrategy) EnterNewHeight(newHeight uint32) {
}

// HandleProposal implements ReplicaStrategy interface.
func (rs *DefaultReplicaStrategy) HandleProposal(p Proposal) {
	e := rs.engine

	// TODO: check if prososal is valid
	log.WithFields(log.Fields{"proposal": p, "id": e.ID()}).Debug("Received proposal")

	// Process block
	block, err := e.chain.AddBlock(&p.block)
	if err != nil {
		log.WithFields(log.Fields{"id": e.ID(), "block": p.block}).Error(err)
		panic(err)
	}

	// Process commit certificate
	if p.commitCertificate != nil {
		ccBlock, err := e.chain.FindBlock(p.commitCertificate.BlockHash)
		if err != nil {
			log.WithFields(log.Fields{"blockhash": fmt.Sprintf("%x", p.commitCertificate.BlockHash)}).Error("Blockhash in commit certificate is not found")
			return
		}
		ccBlock.CommitCertificate = p.commitCertificate

		e.chain.SaveBlock(ccBlock)
		log.WithFields(log.Fields{"id": e.ID(), "error": err, "block": ccBlock, "commitCertificate": p.commitCertificate}).Debug("Update block with commit certificate")

		e.processCCBlock(ccBlock)
	}

	tip := e.setTip()

	// Vote
	if e.lastVoteHeight >= tip.Height {
		log.WithFields(log.Fields{"id": e.ID(), "lastVoteHeight": e.lastVoteHeight, "block.Parent.Height": block.Parent.Height, "block.Hash": block.Hash, "tip": tip.Hash}).Debug("Skip voting since has already voted at height")
	}

	vote := blockchain.Vote{Block: &p.block.BlockHeader, ID: e.ID()}
	e.lastVoteHeight = p.block.Height

	log.WithFields(log.Fields{"vote.block.hash": vote.Block.Hash, "p.proposerID": p.proposerID, "id": e.ID()}).Debug("Sending vote")
	e.network.Broadcast(vote)
}
