package core

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
)

func TestStakeBasics(t *testing.T) {
	assert := assert.New(t)

	addr := common.HexToAddress("0x123")
	amount := new(big.Int).SetUint64(1000)
	stake := NewStake(addr, amount)
	assert.True(stake.Source == addr)
	assert.True(stake.Amount.Cmp(amount) == 0)
	assert.False(stake.Withdrawn)
	assert.True(stake.ReturnHeight > 10000000)
}

func TestStakeDeposit(t *testing.T) {
	assert := assert.New(t)

	sourceAddr1 := common.HexToAddress("0x111")
	stake1AmountInvalid := new(big.Int).SetInt64(-1)
	stake1Amount1 := new(big.Int).SetUint64(1000)
	stake1Amount2 := new(big.Int).SetUint64(4000)

	sourceAddr2 := common.HexToAddress("0x222")
	stake2Amount1 := new(big.Int).SetUint64(8000)

	sourceAddr3 := common.HexToAddress("0x333")
	stake3Amount1 := new(big.Int).SetUint64(500)
	stake3Amount2 := new(big.Int).SetUint64(200)
	stake3Amount3 := new(big.Int).SetUint64(900)

	holderAddr := common.HexToAddress("0xabc")
	stakeHolder := NewStakeHolder(holderAddr, []*Stake{NewStake(sourceAddr1, stake1Amount1)})
	assert.True(stakeHolder.TotalStake().Cmp(stake1Amount1) == 0)
	assert.Equal(len(stakeHolder.Stakes), 1)

	assert.NotNil(stakeHolder.depositStake(sourceAddr1, stake1AmountInvalid)) // negative stake not allowed
	assert.Nil(stakeHolder.depositStake(sourceAddr2, stake2Amount1))
	assert.Nil(stakeHolder.depositStake(sourceAddr1, stake1Amount2))
	assert.Equal(len(stakeHolder.Stakes), 2)

	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount1))
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(13500)) == 0)
	assert.Equal(len(stakeHolder.Stakes), 3)

	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount2))
	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount3))
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(14600)) == 0)
	assert.Equal(len(stakeHolder.Stakes), 3)
}

func TestStakeWithdraw(t *testing.T) {
	assert := assert.New(t)

	sourceAddr1 := common.HexToAddress("0x111")
	stake1Amount1 := new(big.Int).SetUint64(1000)
	stake1Amount2 := new(big.Int).SetUint64(4000)

	sourceAddr2 := common.HexToAddress("0x222")
	stake2Amount1 := new(big.Int).SetUint64(8000)

	sourceAddr3 := common.HexToAddress("0x333")
	stake3Amount1 := new(big.Int).SetUint64(500)
	stake3Amount2 := new(big.Int).SetUint64(200)
	stake3Amount3 := new(big.Int).SetUint64(900)

	sourceAddr4 := common.HexToAddress("0x444")

	currentHeight := uint64(10000)

	holderAddr := common.HexToAddress("0xabc")
	stakeHolder := NewStakeHolder(holderAddr, []*Stake{NewStake(sourceAddr1, stake1Amount1)})
	assert.Nil(stakeHolder.depositStake(sourceAddr2, stake2Amount1))
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(9000)) == 0)

	assert.Nil(stakeHolder.withdrawStake(sourceAddr1, currentHeight))
	assert.NotNil(stakeHolder.withdrawStake(sourceAddr1, currentHeight)) // cannot withdraw twice
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(8000)) == 0)

	assert.NotNil(stakeHolder.depositStake(sourceAddr1, stake1Amount2)) // sourceAddr1 cannot deposit more stake since it is is in the withdrawal locking period
	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount1))    // another account sourceAddr3 should still be able to deposit
	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount2))
	assert.Nil(stakeHolder.depositStake(sourceAddr3, stake3Amount3))
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(9600)) == 0)

	assert.NotNil(stakeHolder.withdrawStake(sourceAddr4, currentHeight)) // sourceAddr4 never deposited, should not be able to withdraw
}

func TestStakeReturn(t *testing.T) {
	assert := assert.New(t)

	sourceAddr1 := common.HexToAddress("0x111")
	stake1Amount1 := new(big.Int).SetUint64(1000)
	stake1Amount2 := new(big.Int).SetUint64(4000)

	sourceAddr2 := common.HexToAddress("0x222")
	stake2Amount1 := new(big.Int).SetUint64(8000)

	sourceAddr3 := common.HexToAddress("0x333")

	initHeight := uint64(10000)

	holderAddr := common.HexToAddress("0xabc")
	stakeHolder := NewStakeHolder(holderAddr, []*Stake{})
	stakeHolder.depositStake(sourceAddr1, stake1Amount1)
	stakeHolder.depositStake(sourceAddr1, stake1Amount2)
	stakeHolder.depositStake(sourceAddr2, stake2Amount1)
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(13000)) == 0)

	assert.Nil(stakeHolder.withdrawStake(sourceAddr1, initHeight))
	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(8000)) == 0)
	assert.Equal(2, len(stakeHolder.Stakes))

	Height1 := uint64(10500)

	returnedStake, err := stakeHolder.returnStake(sourceAddr1, Height1)
	assert.Nil(returnedStake) // should still within the withdrawal locking period
	assert.NotNil(err)

	Height2 := initHeight + ReturnLockingPeriod - 1
	returnedStake, err = stakeHolder.returnStake(sourceAddr1, Height2)
	assert.Nil(returnedStake) // should still within the withdrawal locking period
	assert.NotNil(err)

	Height3 := initHeight + ReturnLockingPeriod
	returnedStake, err = stakeHolder.returnStake(sourceAddr1, Height3)
	assert.NotNil(returnedStake) // should return the stake
	assert.Nil(err)
	assert.True(returnedStake.Amount.Cmp(new(big.Int).SetUint64(5000)) == 0) // should return the full amount

	assert.True(stakeHolder.TotalStake().Cmp(new(big.Int).SetUint64(8000)) == 0)
	assert.Equal(1, len(stakeHolder.Stakes))

	returnedStake, err = stakeHolder.returnStake(sourceAddr2, Height3)
	assert.Nil(returnedStake) // sourceAddr2's stake not withdrawn yet, cannot return
	assert.NotNil(err)

	returnedStake, err = stakeHolder.returnStake(sourceAddr3, Height3)
	assert.Nil(returnedStake) // sourceAddr3 never deposited any stake, so cannot return
	assert.NotNil(err)
}
