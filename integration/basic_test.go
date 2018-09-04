// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/node"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
	"github.com/thetatoken/ukulele/store"
)

func TestConsensusBaseCase(t *testing.T) {
	assert := assert.New(t)

	simnet := p2psim.NewSimnet()

	nodes := []*node.Node{}

	validators := consensus.NewTestValidatorSet([]string{"v1", "v2", "v3", "v4"})

	for _, v := range validators.Validators() {
		store := store.NewMemKVStore()
		chainID := "testchain"
		root := &blockchain.Block{}
		root.ChainID = chainID
		root.Epoch = 0
		root.Hash = blockchain.ParseHex("a0")

		params := &node.Params{
			Store:      store,
			ChainID:    chainID,
			Root:       root,
			Validators: validators,
			Network:    simnet.AddEndpoint(v.ID()),
		}
		nodes = append(nodes, node.NewNode(params))
	}

	testConsensus(assert, simnet, nodes, 5*time.Second, 1)
}
