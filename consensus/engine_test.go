// +build integration

package consensus

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/p2p"
)

func TestConsensusBaseCase(t *testing.T) {
	assert := assert.New(t)

	simnet := p2p.NewSimnet()

	validators := ValidatorSet{"v1", "v2", "v3", "v4"}
	nodes := []Engine{}

	for _, v := range validators {
		nodes = append(nodes, NewEngine(blockchain.CreateTestChain(), simnet.AddEndpoint(v.ID()), validators))
	}

	simnet.Start()

	l := &sync.Mutex{}
	finalizedBlocks := make(map[string][]string)
	for _, node := range nodes {
		node.Start(context.Background())
		finalizedBlocks[node.ID()] = []string{}

		go func(n Engine) {
			for {
				block := <-n.FinalizedBlocks()
				l.Lock()
				finalizedBlocks[n.ID()] = append(finalizedBlocks[n.ID()], block.Hash.String())
				l.Unlock()
			}
		}(node)
	}

	log.Info("Start sleeping")
	time.Sleep(10 * time.Second)
	log.Info("End sleeping")

	l.Lock()
	defer l.Unlock()

	// Verify safety by checking finalized blocks for each replica.
	longestFinalizedBlocks := []string{}
	longest := -1
	for i, node := range nodes {
		finalizedBlocks := finalizedBlocks[node.ID()]
		if i != 0 {
			AssertFinalizedBlocksNotConflicting(assert, longestFinalizedBlocks, finalizedBlocks, fmt.Sprintf("Comparing %v with %v", nodes[longest].ID(), node.ID()))
		}

		// Verify liveness.
		assert.True(len(finalizedBlocks) > 800, fmt.Sprintf("actual len: %d", len(finalizedBlocks)))

		if len(finalizedBlocks) > len(longestFinalizedBlocks) {
			longestFinalizedBlocks = finalizedBlocks
			longest = i
		}
	}

}
