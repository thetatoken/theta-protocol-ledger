// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/consensus"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func TestConsensusBaseCase(t *testing.T) {
	assert := assert.New(t)

	simnet := p2psim.NewSimnet()

	validators := consensus.NewTestValidatorSet([]string{"v1", "v2", "v3", "v4"})
	nodes := []consensus.Engine{}

	for _, v := range validators.Validators() {
		nodes = append(nodes, consensus.NewEngine(blockchain.CreateTestChain(), simnet.AddEndpoint(v.ID()), validators))
	}

	testConsensus(assert, simnet, nodes, 5*time.Second, 1)
}
