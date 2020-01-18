package core

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
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

	// Test FilterByValidators()
	votes3 := NewVoteSet()
	votes3.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A1"),
		Epoch: 1,
	})
	votes3.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A2"),
		Epoch: 1,
	})
	votes3.AddVote(Vote{
		Block: CreateTestBlock("B1", "").Hash(),
		ID:    common.HexToAddress("A3"),
		Epoch: 1,
	})

	vs := NewValidatorSet()
	vs.AddValidator(NewValidator(common.HexToAddress("A1").Hex(), big.NewInt(1e10)))
	vs.AddValidator(NewValidator(common.HexToAddress("A3").Hex(), big.NewInt(1e10)))
	vs.AddValidator(NewValidator(common.HexToAddress("A4").Hex(), big.NewInt(1e10)))
	res = votes3.FilterByValidators(vs)
	assert.Equal(2, len(res.Votes()))
	assert.Equal(res.Votes()[0].ID, common.HexToAddress("A1"))
	assert.Equal(res.Votes()[1].ID, common.HexToAddress("A3"))
}

func TestCommitCertificate(t *testing.T) {
	assert := assert.New(t)

	// Beginning of setup.
	ten18 := new(big.Int).SetUint64(1e18) // 10^18

	va1Stake := new(big.Int).Mul(new(big.Int).SetUint64(100000001), ten18) // 100 million + 1
	va2Stake := new(big.Int).Mul(new(big.Int).SetUint64(100000000), ten18) // 100 million
	va3Stake := new(big.Int).Mul(new(big.Int).SetUint64(50000000), ten18)  // 50 million
	va4Stake := new(big.Int).Mul(new(big.Int).SetUint64(50000000), ten18)  // 50 million

	priv1, _, _ := crypto.GenerateKeyPair()
	va1Addr := priv1.PublicKey().Address()
	va1AddrStr := va1Addr.Hex()
	va1 := NewValidator(va1AddrStr, va1Stake)

	priv2, _, _ := crypto.GenerateKeyPair()
	va2Addr := priv2.PublicKey().Address()
	va2AddrStr := va2Addr.Hex()
	va2 := NewValidator(va2AddrStr, va2Stake)

	priv3, _, _ := crypto.GenerateKeyPair()
	va3Addr := priv3.PublicKey().Address()
	va3AddrStr := va3Addr.Hex()
	va3 := NewValidator(va3AddrStr, va3Stake)

	priv4, _, _ := crypto.GenerateKeyPair()
	va4Addr := priv4.PublicKey().Address()
	va4AddrStr := va4Addr.Hex()
	va4 := NewValidator(va4AddrStr, va4Stake)

	vs := NewValidatorSet()
	vs.AddValidator(va1)
	vs.AddValidator(va2)
	vs.AddValidator(va3)
	vs.AddValidator(va4)

	blockHash := common.HexToHash("a1")
	vote1 := Vote{ID: va1Addr, Block: blockHash, Height: 1}
	vote1.Sign(priv1)
	vote2 := Vote{ID: va2Addr, Block: blockHash, Height: 1}
	vote2.Sign(priv2)
	vote3 := Vote{ID: va3Addr, Block: blockHash, Height: 1}
	vote3.Sign(priv3)
	vote4 := Vote{ID: va4Addr, Block: blockHash, Height: 1}
	vote4.Sign(priv4)

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

	// Reject nil voteset.
	cc := CommitCertificate{}
	assert.False(cc.IsValid(vs))

	// Reject invalid voteset.
	cc = CommitCertificate{Votes: invalidVoteSet, BlockHash: blockHash}
	assert.False(cc.IsValid(vs))

	// Reject empty block hash.
	cc = CommitCertificate{Votes: validVoteSet}
	assert.False(cc.IsValid(vs))

	// Accept valid voteset.
	cc = CommitCertificate{Votes: validVoteSet, BlockHash: blockHash}
	assert.True(cc.IsValid(vs))

	// Reject voteset with duplicate votes from same voter
	voteSet := NewVoteSet()
	voteSet.AddVote(Vote{ID: va1Addr})
	voteSet.AddVote(Vote{ID: va2Addr})
	voteSet.AddVote(Vote{ID: va1Addr})
	cc = CommitCertificate{Votes: invalidVoteSet, BlockHash: blockHash}
	assert.False(cc.IsValid(vs))

	// Reject voteset with votes for other blocks
	voteSet = NewVoteSet()
	voteSet.AddVote(Vote{ID: va1Addr})
	voteSet.AddVote(Vote{ID: va2Addr})
	voteSet.AddVote(Vote{ID: va3Addr, Block: common.HexToHash("0x11")})
	cc = CommitCertificate{Votes: invalidVoteSet, BlockHash: blockHash}
	assert.False(cc.IsValid(vs))
}
