package integration

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/netsync"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func testConsensus(assert *assert.Assertions, simnet *p2psim.Simnet, nodes []consensus.Engine, duration time.Duration, minFinalized int) {
	l := &sync.Mutex{}
	wg := sync.WaitGroup{}
	ctx, stop := context.WithCancel(context.Background())

	log.Info("Start simulation")

	syncManagers := []*netsync.SyncManager{}

	finalizedBlocksByNode := make(map[string][]common.Bytes)
	for _, node := range nodes {
		finalizedBlocksByNode[node.ID()] = []common.Bytes{}
		syncMgr := netsync.NewSyncManager(node.Chain(), node)
		node.Network().AddMessageHandler(syncMgr)
		syncManagers = append(syncManagers, syncMgr)
	}
	simnet.Start(ctx)
	for i, node := range nodes {
		syncManagers[i].Start(ctx)
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
		}(node)
	}

	time.Sleep(duration)
	stop()

	for _, node := range nodes {
		node.Wait()
	}
	wg.Wait()

	log.Info("End simulation")

	var highestFinalizedBlock *blockchain.ExtendedBlock
	var nodeWithHighestBlock consensus.Engine
	for _, node := range nodes {
		finalizedBlocks := finalizedBlocksByNode[node.ID()]
		assert.Truef(len(finalizedBlocks) >= minFinalized, "Node %v didn't finalize many blocks: %v", node.ID(), len(finalizedBlocks))

		lastFinalizedBlock, _ := node.Chain().FindBlock(finalizedBlocks[len(finalizedBlocks)-1])
		if highestFinalizedBlock == nil {
			highestFinalizedBlock = lastFinalizedBlock
			nodeWithHighestBlock = node
		} else if bytes.Compare(highestFinalizedBlock.Hash, lastFinalizedBlock.Hash) != 0 {
			if highestFinalizedBlock.Height < lastFinalizedBlock.Height {
				nodeWithHighestBlock = node
				assert.Truef(nodeWithHighestBlock.Chain().IsDescendant(highestFinalizedBlock.Hash, lastFinalizedBlock.Hash), "Conflict found in finalized blocks: %v, %v, %v, %v", highestFinalizedBlock.Hash, lastFinalizedBlock.Hash, nodeWithHighestBlock.Chain().PrintBranch(highestFinalizedBlock.Hash), nodeWithHighestBlock.Chain().PrintBranch(lastFinalizedBlock.Hash))
				highestFinalizedBlock = lastFinalizedBlock
			} else {
				assert.Truef(nodeWithHighestBlock.Chain().IsDescendant(lastFinalizedBlock.Hash, highestFinalizedBlock.Hash), "Conflict found in finalized blocks: %v, %v, %v, %v", highestFinalizedBlock.Hash, lastFinalizedBlock.Hash, nodeWithHighestBlock.Chain().PrintBranch(highestFinalizedBlock.Hash), nodeWithHighestBlock.Chain().PrintBranch(lastFinalizedBlock.Hash))
			}
		}
	}
}
