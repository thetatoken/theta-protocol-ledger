package integration

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/node"
	p2psim "github.com/thetatoken/theta/p2p/simulation"
)

func testConsensus(assert *assert.Assertions, simnet *p2psim.Simnet, nodes []*node.Node, duration time.Duration, minFinalized int) {
	l := &sync.Mutex{}
	wg := sync.WaitGroup{}
	ctx, stop := context.WithCancel(context.Background())

	log.Info("Start simulation")

	finalizedBlocksByNode := make(map[string][]common.Hash)
	for _, node := range nodes {
		finalizedBlocksByNode[node.Consensus.ID()] = []common.Hash{}
	}
	simnet.Start(ctx)
	for _, node := range nodes {
		node.Start(ctx)
		wg.Add(1)
		go func(n core.ConsensusEngine) {
			defer func() {
				wg.Done()
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case block := <-n.FinalizedBlocks():
					l.Lock()
					finalizedBlocksByNode[n.ID()] = append(finalizedBlocksByNode[n.ID()], block.Hash())
					l.Unlock()
				}

			}
		}(node.Consensus)
	}

	time.Sleep(duration)
	stop()

	for _, node := range nodes {
		node.Wait()
	}
	wg.Wait()

	log.Info("End simulation")

	var highestFinalizedBlock *core.ExtendedBlock
	var nodeWithHighestBlock *node.Node
	for _, node := range nodes {
		finalizedBlocks := finalizedBlocksByNode[node.Consensus.ID()]
		assert.Truef(len(finalizedBlocks) >= minFinalized, "Node %v didn't finalize many blocks: %v", node.Consensus.ID(), len(finalizedBlocks))

		lastFinalizedBlock, _ := node.Chain.FindBlock(finalizedBlocks[len(finalizedBlocks)-1])
		if highestFinalizedBlock == nil {
			highestFinalizedBlock = lastFinalizedBlock
			nodeWithHighestBlock = node
		} else if highestFinalizedBlock.Hash() != lastFinalizedBlock.Hash() {
			if highestFinalizedBlock.Height < lastFinalizedBlock.Height {
				nodeWithHighestBlock = node
				assert.Truef(nodeWithHighestBlock.Chain.IsDescendant(highestFinalizedBlock.Hash(), lastFinalizedBlock.Hash()), "Conflict found in finalized blocks: %v, %v, %v, %v", highestFinalizedBlock.Hash(), lastFinalizedBlock.Hash(), nodeWithHighestBlock.Chain.PrintBranch(highestFinalizedBlock.Hash()), nodeWithHighestBlock.Chain.PrintBranch(lastFinalizedBlock.Hash()))
				highestFinalizedBlock = lastFinalizedBlock
			} else {
				assert.Truef(nodeWithHighestBlock.Chain.IsDescendant(lastFinalizedBlock.Hash(), highestFinalizedBlock.Hash()), "Conflict found in finalized blocks: %v, %v, %v, %v", lastFinalizedBlock.Hash(), highestFinalizedBlock.Hash(), nodeWithHighestBlock.Chain.PrintBranch(highestFinalizedBlock.Hash()), nodeWithHighestBlock.Chain.PrintBranch(lastFinalizedBlock.Hash()))
			}
		}
	}
}
