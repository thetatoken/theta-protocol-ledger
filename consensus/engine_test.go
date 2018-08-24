// +build integration

package consensus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/blockchain"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func TestConsensusBaseCase(t *testing.T) {
	assert := assert.New(t)

	simnet := p2psim.NewSimnet()

	validators := newValidatorSet([]string{"v1", "v2", "v3", "v4"})
	nodes := []Engine{}

	for _, v := range validators.Validators() {
		nodes = append(nodes, NewEngine(blockchain.CreateTestChain(), simnet.AddEndpoint(v.ID()), validators))
	}

	testConsensus(assert, simnet, nodes, 5*time.Second, 10)
}
