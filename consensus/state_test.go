// +build unit

package consensus

import (
	"bytes"
	"testing"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/store"

	"github.com/stretchr/testify/assert"
)

func TestConsensusStateBasic(t *testing.T) {
	assert := assert.New(t)

	db := store.NewMemKVStore()
	chain := blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"A2", "A1",
	})
	cc, _ := chain.FindBlock(blockchain.ParseHex("A1"))

	state1 := NewState(db, chain)
	state1.SetEpoch(3)
	state1.SetLastVoteHeight(10)
	state1.SetHighestCCBlock(cc)

	state2 := NewState(db, chain)
	state2.Load()
	assert.Equal(uint32(3), state2.GetEpoch())
	assert.Equal(uint32(10), state2.GetLastVoteHeight())
	assert.NotNil(state2.GetHighestCCBlock())
	assert.Equal(0, bytes.Compare(blockchain.ParseHex("A1"), state2.GetHighestCCBlock().Hash))
	assert.NotNil(state2.GetTip())
	assert.Equal(0, bytes.Compare(blockchain.ParseHex("A2"), state2.GetTip().Hash))
	assert.NotNil(state2.GetLastFinalizedBlock())
	assert.Equal(0, bytes.Compare(blockchain.ParseHex("A0"), state2.GetLastFinalizedBlock().Hash))
}

func TestConsensusStateVoteSet(t *testing.T) {
	assert := assert.New(t)

	db := store.NewMemKVStore()
	chain := blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"A2", "A1",
	})
	block1 := blockchain.CreateTestBlock("A1", "A0")
	block2 := blockchain.CreateTestBlock("A2", "A1")

	state1 := NewState(db, chain)
	vote1 := &core.Vote{
		Block: block1.BlockHeader,
		ID:    "Alice",
		Epoch: 13,
	}
	vote2 := &core.Vote{
		Block: block2.BlockHeader,
		ID:    "Alice",
		Epoch: 20,
	}
	vote3 := &core.Vote{
		Block: block1.BlockHeader,
		ID:    "Bob",
		Epoch: 20,
	}
	state1.AddVote(vote1)
	state1.AddVote(vote2)
	state1.AddVote(vote3)

	state2 := NewState(db, chain)
	state2.Load()
	vs1, _ := state2.GetEpochVotes()
	votes := vs1.Votes()
	assert.Equal(2, len(votes))
	assert.Equal("Alice", votes[0].ID)
	assert.Equal(uint32(20), votes[0].Epoch)
}
