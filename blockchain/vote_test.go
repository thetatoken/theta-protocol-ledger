package blockchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
)

func TestVoteIndex(t *testing.T) {
	assert := assert.New(t)

	chain := CreateTestChain()
	block1 := core.CreateTestBlock("b1", "")
	block1.Height = 10
	chain.AddBlock(block1)
	b1Hash := core.GetTestBlock("b1").Hash()

	voteSet := chain.FindVotesByHash(b1Hash)
	assert.Equal(0, voteSet.Size())

	v1 := core.Vote{}
	// Should not panic
	chain.AddVoteToIndex(v1)

	v2 := core.Vote{
		Block: b1Hash,
		Epoch: 2,
		ID:    common.HexToAddress("a2"),
	}
	v3 := core.Vote{
		Block: b1Hash,
		Epoch: 3,
		ID:    common.HexToAddress("a3"),
	}
	v4 := core.Vote{
		Block: common.BytesToHash(common.Bytes("v4block")),
		Epoch: 2,
		ID:    common.HexToAddress("a4"),
	}
	chain.AddVoteToIndex(v2)
	chain.AddVoteToIndex(v3)
	chain.AddVoteToIndex(v4)

	voteSet = chain.FindVotesByHash(b1Hash)
	assert.Equal(2, voteSet.Size())
	assert.Equal(uint64(2), voteSet.Votes()[0].Epoch)
	assert.Equal(uint64(3), voteSet.Votes()[1].Epoch)
}
