// +build unit

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/rlp"
)

func TestEncoding(t *testing.T) {
	assert := assert.New(t)

	votes := NewVoteSet()
	votes.AddVote(Vote{
		Block: CreateTestBlock("", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	})
	votes.AddVote(Vote{
		Block: CreateTestBlock("", "").Hash(),
		ID:    common.HexToAddress("A2"),
		Epoch: 1,
	})

	votes2 := NewVoteSet()
	b, err := rlp.EncodeToBytes(votes)
	assert.Nil(err)
	err = rlp.DecodeBytes(b, &votes2)
	assert.Nil(err)

	vs := votes2.Votes()
	vs0 := votes.Votes()

	assert.Equal(2, len(vs))
	assert.Equal(common.HexToAddress("A1"), vs[0].ID)
	assert.NotNil(vs[0].Block)
	assert.Equal(vs0[0].Block, vs[0].Block)

	assert.Equal(common.HexToAddress("A2"), vs[1].ID)
	assert.NotNil(vs[1].Block)
	assert.Equal(vs0[1].Block, vs[1].Block)
}
