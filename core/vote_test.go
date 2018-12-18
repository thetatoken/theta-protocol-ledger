package core

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
)

func TestVoteEncoding(t *testing.T) {
	assert := assert.New(t)

	privKey, _, _ := crypto.GenerateKeyPair()

	v1 := Vote{
		Block: CreateTestBlock("", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	}

	sig, err := privKey.Sign(v1.SignBytes())
	assert.Nil(err)

	v1.SetSignature(sig)

	v2 := Vote{}
	b, err := rlp.EncodeToBytes(v1)
	assert.Nil(err)
	err = rlp.DecodeBytes(b, &v2)
	assert.Nil(err)

	assert.Equal(v1.Block, v2.Block)
	assert.Equal(v1.Epoch, v2.Epoch)
	assert.NotNil(v1.Signature)
	assert.NotNil(v2.Signature)
	assert.True(bytes.Equal(v1.Signature.ToBytes(), v2.Signature.ToBytes()))
}

func TestVoteSetEncoding(t *testing.T) {
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

func TestDedup(t *testing.T) {
	assert := assert.New(t)

	votes1 := NewVoteSet()
	votes1.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	})
	// Duplicate votes
	votes1.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	})
	votes1.AddVote(Vote{
		Block: CreateTestBlock("B2", "").Hash(),
		ID:    common.HexToAddress("A2"),
		Epoch: 1,
	})
	assert.Equal(2, len(votes1.Votes()))

	votes2 := NewVoteSet()
	// Duplicate vote.
	votes2.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	})
	// Duplicate vote from newer epoch.
	votes2.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 3,
	})
	// Duplcate vote from same voter
	votes2.AddVote(Vote{
		Block: CreateTestBlock("B3", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 5,
	})
	// Duplcate vote from same voter
	votes2.AddVote(Vote{
		Block: CreateTestBlock("B4", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 4,
	})
	votes2.AddVote(Vote{
		Block: CreateTestBlock("B2", "").Hash(),
		ID:    common.HexToAddress("A3"),
		Epoch: 1,
	})

	votes := votes1.Merge(votes2)
	assert.Equal(6, len(votes.Votes()))

	res := votes.UniqueVoterAndBlock()
	assert.Equal(5, len(res.Votes()))

	res = votes.UniqueVoter()
	assert.Equal(3, len(res.Votes()))
	v := res.Votes()[0]
	assert.Equal(v.ID, common.HexToAddress("A1"))
	assert.Equal(uint64(5), v.Epoch)
}
