package consensus

import (
	"bytes"
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

	// Process block
	block, err := e.chain.AddBlock(&p.block)
	if err != nil {
		log.WithFields(log.Fields{"id": e.ID(), "block": p.block}).Error(err)
		panic(err)
	}

	// Vote
	lastVoteHeight := e.lastVoteHeight
	tip := e.tip
	// Note: tip's height can be smaller than lastVoteHeight due to CC branch switch.
	if lastVoteHeight > block.Parent.Height || (lastVoteHeight == block.Parent.Height && bytes.Compare(block.Parent.Hash, tip.Hash) != 0) {
		log.WithFields(log.Fields{"id": e.ID(), "lastVoteHeight": lastVoteHeight, "block.Parent.Height": block.Parent.Height, "p.block.Hash": p.block.Hash}).Debug("Skip voting since has already voted at height")
		return
	}

	vote := blockchain.Vote{Block: &p.block.BlockHeader, ID: e.ID()}
	tip, err = e.chain.FindBlock(p.block.Hash)
	if err != nil {
		// Should not happen since we just added block a few lines above.
		panic(err)
	}

	e.tip = tip
	e.lastVoteHeight = p.block.Height

	log.WithFields(log.Fields{"vote.block.hash": vote.Block.Hash, "p.proposerID": p.proposerID, "id": e.ID()}).Debug("Sending vote")
	e.network.Send(p.proposerID, vote)
}
