// +build integration

package integration

import (
	"testing"
	"time"

	"github.com/thetatoken/ukulele/store/database/backend"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/node"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func TestConsensusBaseCase(t *testing.T) {
	assert := assert.New(t)

	simnet := p2psim.NewSimnet()

	nodes := []*node.Node{}

	validators := consensus.NewTestValidatorSet([]string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
	})

	for _, v := range validators.Validators() {
		db := backend.NewMemDatabase()
		chainID := "testchain"
		root := &core.Block{}
		root.ChainID = chainID
		root.Epoch = 0
		root.Hash = blockchain.ParseHex("a0")

		params := &node.Params{
			DB:         db,
			ChainID:    chainID,
			Root:       root,
			Validators: validators,
			Network:    simnet.AddEndpoint(v.ID()),
		}
		nodes = append(nodes, node.NewNode(params))
	}

	testConsensus(assert, simnet, nodes, 5*time.Second, 1)
}
