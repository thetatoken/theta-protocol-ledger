package consensus

import (
	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto/bls"
)

const (
	maxLogNeighbors uint32 = 3 // Estimated number of neighbors during gossip = 2**3 = 8
	maxRound               = 10
)

type GuardianEngine struct {
	logger *log.Entry

	engine  *ConsensusEngine
	privKey *bls.SecretKey

	// State for current voting
	block       common.Hash
	round       uint32
	vote        *core.AggregatedVotes
	gcp         *core.GuardianCandidatePool
	gcpHash     common.Hash
	updated     bool // Whether vote has changed since last broadcast
	signerIndex int  // Signer's index in current gcp
}

func NewGuardianEngine(c *ConsensusEngine, privateKey *bls.SecretKey) *GuardianEngine {
	return &GuardianEngine{
		logger:  util.GetLoggerForModule("guardian"),
		engine:  c,
		privKey: privateKey,
	}
}

func (g *GuardianEngine) IsGuardian() bool {
	return g.signerIndex >= 0
}

func (g *GuardianEngine) StartNewBlock(block common.Hash) {
	g.block = block

	gcp, err := g.engine.GetLedger().GetFinalizedGuardianCandidatePool(block)
	if err != nil {
		// Should not happen
		g.logger.Panic(err)
	}
	g.gcp = gcp
	g.gcpHash = gcp.Hash()
	g.signerIndex = gcp.WithStake().Index(g.privKey.PublicKey())

	if !g.IsGuardian() {
		return
	}

	g.logger.WithFields(log.Fields{
		"block":       block.Hex(),
		"gcp":         g.gcpHash.Hex(),
		"signerIndex": g.signerIndex,
	}).Debug("Starting new block")

	g.vote = core.NewAggregateVotes(block, gcp)
	g.vote.Sign(g.privKey, g.signerIndex)

	g.round = 1
}

func (g *GuardianEngine) StartNewRound() {
	if g.round < maxRound {
		g.round++
	}
}

func (g *GuardianEngine) GetVoteToBroadcast() *core.AggregatedVotes {
	if !g.IsGuardian() {
		return nil
	}

	g.updated = false
	return g.vote
}

func (g *GuardianEngine) HandleVote(vote *core.AggregatedVotes) {
	if !g.IsGuardian() {
		return
	}
	if !g.validateVote(vote) {
		return
	}

	mergedVote, err := g.vote.Merge(vote)
	if err != nil {
		g.logger.WithFields(log.Fields{
			"local.block":           g.block.Hex(),
			"local.round":           g.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"local.vote.Multiplies": g.vote.Multiplies,
			"error":                 err.Error(),
		}).Info("Failed to merge guardian vote")
	}
	if mergedVote == nil {
		// Incoming vote is subset of current vote.
		return
	}
	if !g.checkMultipliesForRound(mergedVote, g.round+1) {
		g.logger.WithFields(log.Fields{
			"local.block":           g.block.Hex(),
			"local.round":           g.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"local.vote.Multiplies": g.vote.Multiplies,
		}).Info("Skipping vote: merged vote overflows")
		return
	}
	g.updated = true

	g.vote = mergedVote

	g.logger.WithFields(log.Fields{
		"local.block":           g.block.Hex(),
		"local.round":           g.round,
		"local.vote.Multiplies": g.vote.Multiplies,
	}).Info("Merged guardian vote")
}

func (g *GuardianEngine) validateVote(vote *core.AggregatedVotes) (res bool) {
	if g.block.IsEmpty() {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring guardian vote: local not ready")
		return
	}
	if vote.Block != g.block {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring guardian vote: block hash does not match with local candidate")
		return
	}
	if vote.Gcp != g.gcpHash {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
		}).Info("Ignoring guardian vote: gcp hash does not match with local value")
		return
	}
	if !g.checkMultipliesForRound(vote, g.round) {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
		}).Info("Ignoring guardian vote: mutiplies exceed limit for round")
		return
	}
	if result := vote.Validate(g.gcp); result.IsError() {
		g.logger.WithFields(log.Fields{
			"local.block":    g.block.Hex(),
			"local.round":    g.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"vote.gcp":       vote.Gcp.Hex(),
			"local.gcp":      g.gcpHash.Hex(),
			"error":          result.Message,
		}).Info("Ignoring guardian vote: invalid vote")
		return
	}
	res = true
	return
}

func (g *GuardianEngine) checkMultipliesForRound(vote *core.AggregatedVotes, k uint32) bool {
	for _, m := range vote.Multiplies {
		if m > g.maxMultiply(k) {
			return false
		}
	}
	return true
}

func (g *GuardianEngine) maxMultiply(k uint32) uint32 {
	return 1 << (k * maxLogNeighbors)
}
