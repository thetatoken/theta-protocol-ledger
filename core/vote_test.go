// +build unit

package core

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/rlp"
)

func TestEncoding(t *testing.T) {
	assert := assert.New(t)

	votes := NewVoteSet()
	votes.AddVote(Vote{
		Block: &BlockHeader{
			ChainID: "testchain",
			Hash:    common.Bytes("block1"),
			Epoch:   1,
		},
		ID:    "Alice",
		Epoch: 1,
	})
	votes.AddVote(Vote{
		Block: &BlockHeader{
			ChainID: "testchain",
			Hash:    common.Bytes("block2"),
			Epoch:   1,
		},
		ID:    "Bob",
		Epoch: 1,
	})

	votes2 := NewVoteSet()
	b, err := rlp.EncodeToBytes(votes)
	assert.Nil(err)
	err = rlp.DecodeBytes(b, &votes2)
	assert.Nil(err)

	vs := votes2.Votes()

	assert.Equal(2, len(vs))
	assert.Equal("Alice", vs[0].ID)
	assert.NotNil(vs[0].Block)
	assert.Equal(0, bytes.Compare(common.Bytes("block1"), vs[0].Block.Hash))

	assert.Equal("Bob", vs[1].ID)
	assert.NotNil(vs[1].Block)
	assert.Equal(0, bytes.Compare(common.Bytes("block2"), vs[1].Block.Hash))
}
