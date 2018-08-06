// +build experimental

package consensus

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/p2p"
	"github.com/thetatoken/ukulele/util"
)

type RandomProposerStrategy struct {
	*DefaultProposerStrategy
}

func (s *RandomProposerStrategy) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.engine.epochManager.C:
			if s.shouldPropose() {
				s.propose()
			}
		}
	}
}

func (s *RandomProposerStrategy) shouldPropose() bool {
	const Probability = 0.3
	validators := s.engine.validatorManager.GetValidatorSetForHeight(0).Validators()
	return rand.Float32() <= Probability && (s.engine.ID() == validators[0].ID() ||
		s.engine.ID() == validators[1].ID())
}

// In the presence of competing proposals, current impletation's average chance of finalization at a certain level lowers, resulting in longer finalization interval or even lack of liveness. Need to investigate no how to support competing proposals.
// Or even separating two processes. One consensus process to elect proposer, there might be competing proposers, but need to be resolved at this level. Another consensus process is to allow the elected proposer to quickly generate N blocks quickly.
//
func TestConsensusRandomProposers(t *testing.T) {
	assert := assert.New(t)

	rand.Seed(1)
	simnet := p2p.NewSimnet()

	validators := newValidatorSet([]string{"v1", "v2", "v3", "v4"})
	nodes := []Engine{}

	for _, v := range validators.Validators() {
		node := NewEngine(blockchain.CreateTestChain(), simnet.AddEndpoint(v.ID()), validators)
		node.SetProposerStrategy(&RandomProposerStrategy{&DefaultProposerStrategy{}})
		nodes = append(nodes, node)
	}

	simnet.Start()

	for _, node := range nodes {
		node.Start(context.Background())
	}

	log.Info("Start sleeping")
	time.Sleep(10 * time.Second)
	log.Info("End sleeping")

	// Verify safety by checking finalized blocks for each replica.
	longestFinalizedBlocks := []string{}
	longest := -1
	for i, node := range nodes {
		finalizedBlocks := GetFinalizedBlocks(node.FinalizedBlocks())
		if i != 0 {
			AssertFinalizedBlocksNotConflicting(assert, longestFinalizedBlocks, finalizedBlocks, fmt.Sprintf("Comparing %v with %v", nodes[longest].ID(), nodes[i].ID()))
		}

		// Verify liveness.
		assert.True(len(finalizedBlocks) > 0, fmt.Sprintf("len(finalizedBlocks) should > 0: %v, %v", len(finalizedBlocks), finalizedBlocks))

		if len(finalizedBlocks) > len(longestFinalizedBlocks) {
			longestFinalizedBlocks = finalizedBlocks
			longest = i
		}
	}
}

type CompetingProposerStrategy struct {
	*DefaultProposerStrategy
}

func (s *CompetingProposerStrategy) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(viper.GetInt(util.CfgConsesusMaxEpochLength)) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if s.shouldPropose() {
				s.propose()
			}
		}
	}
}

func (s *CompetingProposerStrategy) shouldPropose() bool {
	validators := s.engine.validatorManager.GetValidatorSetForHeight(0).Validators()
	return s.engine.ID() == validators[0].ID() ||
		s.engine.ID() == validators[1].ID()
}

// In the presence of competing proposals, current impletation's average chance of finalization at a certain level lowers, resulting in longer finalization interval or even lack of liveness. Need to investigate no how to support competing proposals.
// Or even separating two processes. One consensus process to elect proposer, there might be competing proposers, but need to be resolved at this level. Another consensus process is to allow the elected proposer to quickly generate N blocks quickly.
//
func TestConsensusCompetingProposers(t *testing.T) {
	assert := assert.New(t)

	rand.Seed(1)
	simnet := p2p.NewSimnet()

	validators := newValidatorSet([]string{"v1", "v2", "v3", "v4"})
	nodes := []Engine{}

	for _, v := range validators.Validators() {
		node := NewEngine(blockchain.CreateTestChain(), simnet.AddEndpoint(v.ID()), validators)
		node.SetProposerStrategy(&CompetingProposerStrategy{&DefaultProposerStrategy{}})
		nodes = append(nodes, node)
	}

	simnet.Start()

	for _, node := range nodes {
		node.Start(context.Background())
	}

	log.Info("Start sleeping")
	time.Sleep(10 * time.Second)
	log.Info("End sleeping")

	// Verify safety by checking finalized blocks for each replica.
	longestFinalizedBlocks := []string{}
	longest := -1
	for i, node := range nodes {
		finalizedBlocks := GetFinalizedBlocks(node.FinalizedBlocks())
		if i != 0 {
			AssertFinalizedBlocksNotConflicting(assert, longestFinalizedBlocks, finalizedBlocks, fmt.Sprintf("Comparing %v with %v", nodes[longest].ID(), nodes[i].ID()))
		}

		// Verify liveness.
		assert.True(len(finalizedBlocks) > 0, fmt.Sprintf("len(finalizedBlocks) should > 0: %v", finalizedBlocks))

		if len(finalizedBlocks) > len(longestFinalizedBlocks) {
			longestFinalizedBlocks = finalizedBlocks
			longest = i
		}
	}
}
