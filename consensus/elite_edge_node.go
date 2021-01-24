package consensus

import (
	"context"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto/bls"
)

const (
	maxEENLogNeighbors uint32 = 3 // Estimated number of neighbors during gossip = 2**3 = 8
	maxEENRound               = 10
)

type EliteEdgeNodeEngine struct {
	logger *log.Entry

	engine  *ConsensusEngine
	privKey *bls.SecretKey

	// State for current voting
	block    common.Hash
	round    uint32
	currVote *core.AggregatedEENVotes
	nextVote *core.AggregatedEENVotes
	eenp     *core.EliteEdgeNodePool

	incoming chan *core.AggregatedEENVotes
	mu       *sync.Mutex
}

func NewEliteEdgeNodeEngine(c *ConsensusEngine, privateKey *bls.SecretKey) *EliteEdgeNodeEngine {
	return &EliteEdgeNodeEngine{
		logger:  util.GetLoggerForModule("elite edge node"),
		engine:  c,
		privKey: privateKey,

		incoming: make(chan *core.AggregatedEENVotes, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		mu:       &sync.Mutex{},
	}
}

func (e *EliteEdgeNodeEngine) StartNewBlock(block common.Hash) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.block = block
	e.nextVote = nil
	e.currVote = nil
	e.round = 1

	eenp, err := e.engine.GetLedger().GetEliteEdgeNodePool(block)
	if err != nil {
		// Should not happen
		e.logger.Panic(err)
	}
	e.eenp = eenp

	e.logger.WithFields(log.Fields{
		"block": block.Hex(),
	}).Debug("Starting new block")
}

func (e *EliteEdgeNodeEngine) StartNewRound() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.round < maxRound {
		e.round++
		if e.nextVote != nil {
			e.currVote = e.nextVote.Copy()
		}
	}
}

func (e *EliteEdgeNodeEngine) GetVoteToBroadcast() *core.AggregatedEENVotes {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.currVote
}

func (e *EliteEdgeNodeEngine) GetBestVote() *core.AggregatedEENVotes {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.nextVote
}

func (e *EliteEdgeNodeEngine) Start(ctx context.Context) {
	go e.mainLoop(ctx)
}

func (e *EliteEdgeNodeEngine) mainLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case vote, ok := <-e.incoming:
			if ok {
				e.processVote(vote)
			}
		}
	}
}

func (e *EliteEdgeNodeEngine) processVote(vote *core.AggregatedEENVotes) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.validateVote(vote) {
		return
	}

	if e.nextVote == nil {
		e.nextVote = vote
		return
	}

	var candidate *core.AggregatedEENVotes
	var err error

	candidate, err = e.nextVote.Merge(vote)
	if err != nil {
		e.logger.WithFields(log.Fields{
			"e.block":               e.block.Hex(),
			"e.round":               e.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"e.nextVote.Multiplies": e.nextVote.Multiplies,
			"e.nextVote.Block":      e.nextVote.Block.Hex(),
			"error":                 err.Error(),
		}).Info("Failed to merge elite edge node vote")
	}
	if candidate == nil {
		// Incoming vote is subset of the current nextVote.
		e.logger.WithFields(log.Fields{
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Debug("Skipping vote: no new index")
		return
	}

	if !e.checkMultipliesForRound(candidate, e.round+1) {
		e.logger.WithFields(log.Fields{
			"local.block":           e.block.Hex(),
			"local.round":           e.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"local.vote.Multiplies": e.nextVote.Multiplies,
		}).Info("Skipping vote: candidate vote overflows")
		return
	}

	e.nextVote = candidate

	e.logger.WithFields(log.Fields{
		"local.block":           e.block.Hex(),
		"local.round":           e.round,
		"local.vote.Multiplies": e.nextVote.Multiplies,
	}).Info("New elite edge node vote")
}

func (e *EliteEdgeNodeEngine) HandleVote(vote *core.AggregatedEENVotes) {
	select {
	case e.incoming <- vote:
		return
	default:
		e.logger.Debug("EliteEdgeNodeEngine queue is full, discarding vote: %v", vote)
	}
}

func (e *EliteEdgeNodeEngine) validateVote(vote *core.AggregatedEENVotes) (res bool) {
	if e.block.IsEmpty() {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring elite edge node vote: local not ready")
		return
	}
	if vote.Block != e.block {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring elite edge node vote: block hash does not match with local candidate")
		return
	}
	if !e.checkMultipliesForRound(vote, e.round) {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring elite edge node vote: mutiplies exceed limit for round")
		return
	}
	if result := vote.Validate(e.eenp); result.IsError() {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"error":          result.Message,
		}).Info("Ignoring elite edge node vote: invalid vote")
		return
	}
	res = true
	return
}

func (e *EliteEdgeNodeEngine) checkMultipliesForRound(vote *core.AggregatedEENVotes, k uint32) bool {
	// for _, m := range vote.Multiplies {
	// 	if m > g.maxMultiply(k) {
	// 		return false
	// 	}
	// }
	return true
}

func (e *EliteEdgeNodeEngine) maxMultiply(k uint32) uint32 {
	return 1 << (k * maxEENLogNeighbors)
}
