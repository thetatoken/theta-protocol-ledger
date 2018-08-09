package consensus

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"io"
	"math/rand"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/blockchain"
)

// ProposerStrategy defines the proposer interface that is used by DefaultEngine.
type ProposerStrategy interface {
	Init(e *DefaultEngine)
	HandleVote(v blockchain.Vote)
	EnterNewHeight(newHeight uint32)
}

var _ ProposerStrategy = &DefaultProposerStrategy{}

// DefaultProposerStrategy is the default implementation of the ProposerStrategy interface.
type DefaultProposerStrategy struct {
	engine *DefaultEngine
	rand   *rand.Rand
}

// Init implements ProposerStrategy interface.
func (ps *DefaultProposerStrategy) Init(e *DefaultEngine) {
	ps.engine = e

	h := md5.New()
	io.WriteString(h, e.ID())
	seed := binary.BigEndian.Uint64(h.Sum(nil))
	ps.rand = rand.New(rand.NewSource(int64(seed)))
}

// EnterNewHeight implements ProposerStrategy interface.
func (ps *DefaultProposerStrategy) EnterNewHeight(newHeight uint32) {
	if ps.shouldPropose(newHeight) {
		ps.propose()
	}
}

func (ps *DefaultProposerStrategy) randHex() []byte {
	bytes := make([]byte, 10)
	ps.rand.Read(bytes)
	return bytes
}

func (ps *DefaultProposerStrategy) shouldPropose(height uint32) bool {
	proposer := ps.engine.validatorManager.GetProposerForHeight(height)
	return proposer.ID() == ps.engine.ID()
}

// HandleVote implements ProposerStrategy interface.
func (ps *DefaultProposerStrategy) HandleVote(vote blockchain.Vote) {
	e := ps.engine
	log.WithFields(log.Fields{"vote": vote, "id": e.ID()}).Debug("Received vote")

	hs := hex.EncodeToString(vote.Block.Hash)
	block, err := e.Chain().FindBlock(vote.Block.Hash)
	if err != nil {
		log.WithFields(log.Fields{"vote.block.hash": vote.Block.Hash}).Warn("Block hash in vote is not found")
		return
	}
	votes, ok := e.collectedVotes[hs]
	if !ok {
		votes = blockchain.NewVoteSet()
		e.collectedVotes[hs] = votes
	}
	votes.AddVote(vote)

	validators := e.validatorManager.GetValidatorSetForHeight(e.height)
	if validators.HasMajority(votes) {
		cc := &blockchain.CommitCertificate{Votes: votes, BlockHash: vote.Block.Hash}
		block.CommitCertificate = cc

		e.chain.SaveBlock(block)
		e.processCCBlock(block)
	}
}

func (ps *DefaultProposerStrategy) propose() {
	e := ps.engine

	tip := ps.engine.getTip()
	if _, ok := e.voteLog[tip.Height]; ok {
		return
	}

	block := blockchain.Block{}
	block.ChainID = e.chain.ChainID
	block.Hash = ps.randHex()
	block.Height = e.height
	block.ParentHash = tip.Hash

	lastCC := e.highestCCBlock
	proposal := Proposal{block: block, proposerID: e.ID()}
	if lastCC.CommitCertificate != nil {
		proposal.commitCertificate = lastCC.CommitCertificate.Copy()
	}

	log.WithFields(log.Fields{"proposal": proposal, "id": e.ID()}).Info("Making proposal")
	e.network.Broadcast(proposal)
}
