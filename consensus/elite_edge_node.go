package consensus

import (
	"context"
	"fmt"
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
	maxEENRound               = 20
)

type EliteEdgeNodeEngine struct {
	logger *log.Entry

	engine  *ConsensusEngine
	privKey *bls.SecretKey

	voteBookkeeper *EENVoteBookkeeper

	// State for current voting
	block    common.Hash
	round    uint32
	currVote *core.AggregatedEENVotes
	nextVote *core.AggregatedEENVotes
	eenp     *core.EliteEdgeNodePool

	evIncoming  chan *core.EENVote
	aevIncoming chan *core.AggregatedEENVotes
	mu          *sync.Mutex
}

func NewEliteEdgeNodeEngine(c *ConsensusEngine, privateKey *bls.SecretKey) *EliteEdgeNodeEngine {
	return &EliteEdgeNodeEngine{
		logger:  util.GetLoggerForModule("elite edge node"),
		engine:  c,
		privKey: privateKey,

		voteBookkeeper: CreateEENVoteBookkeeper(DefaultMaxNumVotesCached),

		evIncoming:  make(chan *core.EENVote, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		aevIncoming: make(chan *core.AggregatedEENVotes, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		mu:          &sync.Mutex{},
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

	if e.round < maxEENRound {
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
		case ev, ok := <-e.evIncoming:
			if ok {
				e.processVote(ev)
			}
		case aev, ok := <-e.aevIncoming:
			if ok {
				e.processAggregatedVote(aev)
			}
		}
	}
}

func (e *EliteEdgeNodeEngine) processVote(vote *core.EENVote) {
	e.mu.Lock()
	defer e.mu.Unlock()

	logger.Debugf("Process edge node vote {%v : %v}", vote.Address, vote.Block.Hex())

	if !e.validateVote(vote) {
		return
	}

	logger.Debugf("Validated edge node vote {%v : %v}", vote.Address, vote.Block.Hex())

	aggregatedVote, err := e.convertVote(vote)
	if err != nil {
		logger.Warnf("Discard vote from edge node %v, reason: %v", vote.Address, err)
		return
	}

	logger.Debugf("Converted edge node vote to aggregated vote {%v : %v}", vote.Address, vote.Block.Hex())

	e.aevIncoming <- aggregatedVote
}

// convertVote converts an EENVote into an AggregatedEENVotes
func (e *EliteEdgeNodeEngine) convertVote(ev *core.EENVote) (*core.AggregatedEENVotes, error) {
	if e.eenp == nil {
		return nil, fmt.Errorf("The elite edge node pool is nil, cannot convert vote")
	}

	signerIdx := e.eenp.WithStake().IndexWithHolderAddress(ev.Address)
	if signerIdx < 0 {
		return nil, fmt.Errorf("Elite edge node %v not found in the Elite edge node pool", ev.Address)
	}

	eenv := core.NewAggregatedEENVotes(ev.Block, e.eenp)
	eenv.Multiplies[signerIdx] = 1
	eenv.Signature.Aggregate(ev.Signature)

	logger.Infof("converted edge node vote for block %v from edge node %v to an aggregated vote", ev.Block.Hex(), ev.Address)

	return eenv, nil
}

func (e *EliteEdgeNodeEngine) processAggregatedVote(vote *core.AggregatedEENVotes) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.validateAggregatedVote(vote) {
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
		}).Info("Failed to merge aggregated elite edge node vote")
	}
	if candidate == nil {
		// Incoming vote is subset of the current nextVote.
		e.logger.WithFields(log.Fields{
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Debug("Skipping aggregated elite edge node vote: no new index")
		return
	}

	if !e.checkMultipliesForRound(candidate, e.round+1) {
		e.logger.WithFields(log.Fields{
			"local.block":           e.block.Hex(),
			"local.round":           e.round,
			"vote.block":            vote.Block.Hex(),
			"vote.Mutiplies":        vote.Multiplies,
			"local.vote.Multiplies": e.nextVote.Multiplies,
		}).Info("Skipping aggregated elite edge node vote: candidate vote overflows")
		return
	}

	e.nextVote = candidate

	e.logger.WithFields(log.Fields{
		"local.block":           e.block.Hex(),
		"local.round":           e.round,
		"local.vote.Multiplies": e.nextVote.Multiplies,
	}).Info("New aggregated elite edge node vote")
}

func (e *EliteEdgeNodeEngine) HandleVote(vote *core.EENVote) {
	if e.voteBookkeeper.HasSeen(vote) {
		logger.Debugf("Received edge node vote {%v : %v} earlier, safely ignore", vote.Address, vote.Block.Hex())
		return
	}
	e.voteBookkeeper.Record(vote)

	logger.Debugf("Received edge node vote {%v : %v} for the first time", vote.Address, vote.Block.Hex())

	select {
	case e.evIncoming <- vote:
		return
	default:
		e.logger.Debug("EliteEdgeNodeEngine queue is full, discarding elite edge node vote: %v", vote)
	}
}

func (e *EliteEdgeNodeEngine) HandleAggregatedVote(vote *core.AggregatedEENVotes) {
	select {
	case e.aevIncoming <- vote:
		return
	default:
		e.logger.Debug("EliteEdgeNodeEngine queue is full, discarding aggregated elite edge node vote: %v", vote)
	}
}

func (e *EliteEdgeNodeEngine) validateVote(vote *core.EENVote) (res bool) {
	if e.block.IsEmpty() {
		e.logger.WithFields(log.Fields{
			"local.block": e.block.Hex(),
			"local.round": e.round,
			"vote.block":  vote.Block.Hex(),
		}).Info("Ignoring elite edge node vote: local not ready")
		return
	}
	if vote.Block != e.block {
		e.logger.WithFields(log.Fields{
			"local.block": e.block.Hex(),
			"local.round": e.round,
			"vote.block":  vote.Block.Hex(),
		}).Info("Ignoring elite edge node vote: block hash does not match with local candidate")
		return
	}
	if result := vote.Validate(e.eenp); result.IsError() {
		e.logger.WithFields(log.Fields{
			"local.block": e.block.Hex(),
			"local.round": e.round,
			"vote.block":  vote.Block.Hex(),
			"reason":      result.Message,
		}).Info("Ignoring elite edge node vote: vote validation failed")
		return
	}
	res = true
	return
}

func (e *EliteEdgeNodeEngine) validateAggregatedVote(vote *core.AggregatedEENVotes) (res bool) {
	if e.block.IsEmpty() {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring aggregated elite edge node vote: local not ready")
		return
	}
	if vote.Block != e.block {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring aggregated elite edge node vote: block hash does not match with local candidate")
		return
	}
	if !e.checkMultipliesForRound(vote, e.round) {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
		}).Info("Ignoring aggregated elite edge node vote: mutiplies exceed limit for round")
		return
	}
	if result := vote.Validate(e.eenp); result.IsError() {
		e.logger.WithFields(log.Fields{
			"local.block":    e.block.Hex(),
			"local.round":    e.round,
			"vote.block":     vote.Block.Hex(),
			"vote.Mutiplies": vote.Multiplies,
			"error":          result.Message,
		}).Info("Ignoring aggregated elite edge node vote: invalid vote")
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
