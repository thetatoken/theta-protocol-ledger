// +build integration

package integration

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
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

	viper.Set(common.CfgLogPrintSelfID, true)

	simnet := p2psim.NewSimnet()

	nodes := []*node.Node{}

	validators := consensus.NewTestValidatorSet([]string{
		"042CA7FFB62122A220C72AA7CD87C252B21D72273275682386A099F0983C135659FF93E2E8756011074706E18113AA6529CD5833DD6463266980C6973895153C7C",
		"048E8D53FD435265AD074597CC3E202F8E935CFB57925BB51316252027CB08767FB8099226414732543C4B5CBAA64B4EE8F173BA559258A0B5F633A0D11509E78B",
		"0479188733862EBB3FE98A92315556D5214D908941CDC8D6C8700EEEAE5F90A6177A37E23B33B81B9FAC3A98EE2382AB24B1C92384FC151D07E36AC7209702D353",
		"0455BDC5CF697F9519DF40E837BEE3E246C8D47C1B58CD1892FD3B0F780D2C09E718FF50A5929B86B8B88C7031164BDE553E285103F1B4DF668B44AFC907264C1C",
	})

	privKeys := []string{
		"A249A82C42A282E87B2DDEF63404D9DFCF6EA501DCAF5D447761765BD74F666D",
		"93A90EA508331DFDF27FB79757D4250B4E84954927BA0073CD67454AC432C737",
		"D0D53AC0B4CD47D0CE0060DDDC179D04145FEA2EE2E0B66C3EE1699C6B492013",
		"83F0BB8655139CEF4657F90DB64A7BB57847038A9BD0CCD87C9B0828E9CBF76D",
	}

	for i, v := range validators.Validators() {
		privateKeyBytes, _ := hex.DecodeString(privKeys[i])
		privateKey, _ := crypto.PrivateKeyFromBytes(privateKeyBytes)
		db := backend.NewMemDatabase()
		chainID := "testchain"
		root := &core.Block{}
		root.ChainID = chainID
		root.Epoch = 0
		root.Hash = blockchain.ParseHex("a0")

		params := &node.Params{
			PrivateKey: privateKey,
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
