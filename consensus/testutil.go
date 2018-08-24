package consensus

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

// GetFinalizedBlocks drains the FinalizedBlocks channel and return a slice of block hashes.
func GetFinalizedBlocks(ch chan *blockchain.Block) []string {
	res := []string{}
loop:
	for {
		select {
		case block := <-ch:
			res = append(res, block.Hash.String())
		default:
			break loop
		}
	}
	return res
}

// AssertFinalizedBlocks asserts finalized blocks are as expected.
func AssertFinalizedBlocks(assert *assert.Assertions, expected []string, ch chan *blockchain.Block) {
	assert.Equal(expected, GetFinalizedBlocks(ch))
}

// AssertFinalizedBlocksNotConflicting asserts two chains are not conflicting.
func AssertFinalizedBlocksNotConflicting(assert *assert.Assertions, c1 []string, c2 []string, msg string) {
	length := len(c2)
	if len(c1) < len(c2) {
		length = len(c1)
	}
	for i := 0; i < length; i++ {
		if c1[i] != c2[i] {
			assert.Failf(msg, "Conflicts found: %v, %v", c1, c2)
		}
	}
}

func newValidatorSet(ids []string) *ValidatorSet {
	s := NewValidatorSet()
	for _, id := range ids {
		v := NewValidator(id, 1)
		s.AddValidator(v)
	}
	return s
}

func testConsensus(assert *assert.Assertions, simnet *p2psim.Simnet, nodes []Engine, duration time.Duration, minFinalized int) {
	l := &sync.Mutex{}
	wg := sync.WaitGroup{}
	ctx, stop := context.WithCancel(context.Background())

	log.Info("Start simulation")

	simnet.Start(ctx)
	finalizedBlocksByNode := make(map[string][]common.Bytes)
	for _, node := range nodes {
		finalizedBlocksByNode[node.ID()] = []common.Bytes{}
	}
	for _, node := range nodes {
		node.Start(ctx)
		wg.Add(1)
		go func(n Engine) {
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
	var nodeWithHighestBlock Engine
	for _, node := range nodes {
		finalizedBlocks := finalizedBlocksByNode[node.ID()]
		assert.Truef(len(finalizedBlocks) > minFinalized, "Node %v didn't finalize many blocks: %v", node.ID(), len(finalizedBlocks))

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
