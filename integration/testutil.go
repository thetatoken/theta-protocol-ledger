package integration

import (
	"bytes"
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/node"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func testConsensus(assert *assert.Assertions, simnet *p2psim.Simnet, nodes []*node.Node, duration time.Duration, minFinalized int) {
	l := &sync.Mutex{}
	wg := sync.WaitGroup{}
	ctx, stop := context.WithCancel(context.Background())

	log.Info("Start simulation")

	finalizedBlocksByNode := make(map[string][]common.Bytes)
	for _, node := range nodes {
		finalizedBlocksByNode[node.Consensus.ID()] = []common.Bytes{}
	}
	simnet.Start(ctx)
	for _, node := range nodes {
		node.Start(ctx)
		wg.Add(1)
		go func(n consensus.Engine) {
			defer func() {
				wg.Done()
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case block := <-n.FinalizedBlocks():
					l.Lock()
					finalizedBlocksByNode[n.ID()] = append(finalizedBlocksByNode[n.ID()], block.Hash)
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

	var highestFinalizedBlock *blockchain.ExtendedBlock
	var nodeWithHighestBlock *node.Node
	for _, node := range nodes {
		finalizedBlocks := finalizedBlocksByNode[node.Consensus.ID()]
		assert.Truef(len(finalizedBlocks) >= minFinalized, "Node %v didn't finalize many blocks: %v", node.Consensus.ID(), len(finalizedBlocks))

		lastFinalizedBlock, _ := node.Chain.FindBlock(finalizedBlocks[len(finalizedBlocks)-1])
		if highestFinalizedBlock == nil {
			highestFinalizedBlock = lastFinalizedBlock
			nodeWithHighestBlock = node
		} else if bytes.Compare(highestFinalizedBlock.Hash, lastFinalizedBlock.Hash) != 0 {
			if highestFinalizedBlock.Height < lastFinalizedBlock.Height {
				nodeWithHighestBlock = node
				assert.Truef(nodeWithHighestBlock.Chain.IsDescendant(highestFinalizedBlock.Hash, lastFinalizedBlock.Hash, 1000), "Conflict found in finalized blocks: %v, %v, %v, %v", highestFinalizedBlock.Hash, lastFinalizedBlock.Hash, nodeWithHighestBlock.Chain.PrintBranch(highestFinalizedBlock.Hash), nodeWithHighestBlock.Chain.PrintBranch(lastFinalizedBlock.Hash))
				highestFinalizedBlock = lastFinalizedBlock
			} else {
				assert.Truef(nodeWithHighestBlock.Chain.IsDescendant(lastFinalizedBlock.Hash, highestFinalizedBlock.Hash, 1000), "Conflict found in finalized blocks: %v, %v, %v, %v", lastFinalizedBlock.Hash, highestFinalizedBlock.Hash, nodeWithHighestBlock.Chain.PrintBranch(highestFinalizedBlock.Hash), nodeWithHighestBlock.Chain.PrintBranch(lastFinalizedBlock.Hash))
			}
		}
	}
}
