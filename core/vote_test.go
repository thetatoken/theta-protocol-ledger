package core

import (
	"bytes"
	"math/big"
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

func TestCommitCertificate(t *testing.T) {
	assert := assert.New(t)

	// Begining of setup.
	ten18 := new(big.Int).SetUint64(1000000000000000000) // 10^18

	va1Stake := new(big.Int).Mul(new(big.Int).SetUint64(100000001), ten18) // 100 million + 1
	va2Stake := new(big.Int).Mul(new(big.Int).SetUint64(100000000), ten18) // 100 million
	va3Stake := new(big.Int).Mul(new(big.Int).SetUint64(50000000), ten18)  // 50 million
	va4Stake := new(big.Int).Mul(new(big.Int).SetUint64(50000000), ten18)  // 50 million

	va1AddrStr := "0x111"
	va1Addr := common.HexToAddress(va1AddrStr)
	va1 := NewValidator(va1AddrStr, va1Stake)

	va2AddrStr := "0x222"
	va2Addr := common.HexToAddress(va2AddrStr)
	va2 := NewValidator(va2AddrStr, va2Stake)

	va3AddrStr := "0x333"
	va3Addr := common.HexToAddress(va3AddrStr)
	va3 := NewValidator(va3AddrStr, va3Stake)

	va4AddrStr := "0x444"
	va4Addr := common.HexToAddress(va4AddrStr)
	va4 := NewValidator(va4AddrStr, va4Stake)

	vs := NewValidatorSet()
	vs.AddValidator(va1)
	vs.AddValidator(va2)
	vs.AddValidator(va3)
	vs.AddValidator(va4)

	vote1 := Vote{ID: va1Addr}
	vote2 := Vote{ID: va2Addr}
	vote3 := Vote{ID: va3Addr}
	vote4 := Vote{ID: va4Addr}

	invalidVoteSet := NewVoteSet()
	invalidVoteSet.AddVote(vote1)
	assert.False(vs.HasMajority(invalidVoteSet))

	validVoteSet := NewVoteSet()
	validVoteSet.AddVote(vote1)
	validVoteSet.AddVote(vote2)
	validVoteSet.AddVote(vote3)
	validVoteSet.AddVote(vote4)

	assert.True(vs.HasMajority(validVoteSet))
	// End of setup.

	// Allows nil voteset.
	cc := CommitCertificate{}
	assert.True(cc.IsValid(vs))
	assert.False(cc.IsProven(vs))

	// Reject invalid voteset.
	cc = CommitCertificate{Votes: invalidVoteSet}
	assert.False(cc.IsValid(vs))
	assert.False(cc.IsProven(vs))

	// Accept valid voteset.
	cc = CommitCertificate{Votes: validVoteSet}
	assert.True(cc.IsValid(vs))
	assert.True(cc.IsProven(vs))

	// Reject voteset with duplicate votes from same voter
	voteSet := NewVoteSet()
	voteSet.AddVote(Vote{ID: va1Addr})
	voteSet.AddVote(Vote{ID: va2Addr})
	voteSet.AddVote(Vote{ID: va1Addr})
	cc = CommitCertificate{Votes: invalidVoteSet}
	assert.False(cc.IsValid(vs))
	assert.False(cc.IsProven(vs))

	// Reject voteset with votes for other blocks
	voteSet = NewVoteSet()
	voteSet.AddVote(Vote{ID: va1Addr})
	voteSet.AddVote(Vote{ID: va2Addr})
	voteSet.AddVote(Vote{ID: va3Addr, Block: common.HexToHash("0x11")})
	cc = CommitCertificate{Votes: invalidVoteSet}
	assert.False(cc.IsValid(vs))
	assert.False(cc.IsProven(vs))
}
