package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func TestValidatorSet(t *testing.T) {
	assert := assert.New(t)

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

	assert.True(vs.TotalStake().Cmp(new(big.Int).Mul(new(big.Int).SetUint64(300000001), ten18)) == 0)

	vax, err := vs.GetValidator(va1Addr)
	assert.Equal(vax.ID(), va1Addr)
	assert.Nil(err)

	vax, err = vs.GetValidator(common.HexToAddress("0x555"))
	assert.NotNil(err)

	vote1 := Vote{ID: va1Addr}
	vote2 := Vote{ID: va2Addr}
	vote3 := Vote{ID: va3Addr}
	vote4 := Vote{ID: va4Addr}

	voteSet0 := NewVoteSet()
	assert.False(vs.HasMajority(voteSet0)) // empty vote set

	voteSet1 := NewVoteSet()
	voteSet1.AddVote(vote1)
	assert.False(vs.HasMajority(voteSet1)) // about 1/3

	voteSet2 := NewVoteSet()
	voteSet2.AddVote(vote2)
	voteSet2.AddVote(vote3)
	voteSet2.AddVote(vote4)
	assert.False(vs.HasMajority(voteSet2)) // slightly less than 2/3

	voteSet3 := NewVoteSet()
	voteSet3.AddVote(vote1)
	voteSet3.AddVote(vote2)
	assert.True(vs.HasMajority(voteSet3)) // slightly above 2/3

	voteSet4 := NewVoteSet()
	voteSet4.AddVote(vote1)
	voteSet4.AddVote(vote2)
	voteSet4.AddVote(vote3)
	voteSet4.AddVote(vote4)
	assert.True(vs.HasMajority(voteSet4)) // full set
}
