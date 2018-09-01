// +build experimental

package consensus

import (
	"context"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	p2p "github.com/thetatoken/ukulele/p2p/simulation"
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
	validators := s.engine.validatorManager.GetValidatorSetForEpoch(0).Validators()
	return rand.Float32() <= Probability && (s.engine.ID() == validators[0].ID() ||
		s.engine.ID() == validators[1].ID())
}

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

	testConsensus(assert, simnet, nodes, 20*time.Second, 100)
}

type CompetingProposerStrategy struct {
	*DefaultProposerStrategy
}

func (s *CompetingProposerStrategy) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(viper.GetInt(common.CfgConsensusMaxEpochLength)) * time.Second)
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
	validators := s.engine.validatorManager.GetValidatorSetForEpoch(0).Validators()
	return s.engine.ID() == validators[0].ID() ||
		s.engine.ID() == validators[1].ID()
}

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

	testConsensus(assert, simnet, nodes, 20*time.Second, 100)
}
