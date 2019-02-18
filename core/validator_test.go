package core

import (
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
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

	vsc := vs.Copy()

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
	assert.False(vsc.HasMajority(voteSet0)) // empty vote set

	voteSet1 := NewVoteSet()
	voteSet1.AddVote(vote1)
	assert.False(vsc.HasMajority(voteSet1)) // about 1/3

	voteSet2 := NewVoteSet()
	voteSet2.AddVote(vote2)
	voteSet2.AddVote(vote3)
	voteSet2.AddVote(vote4)
	assert.False(vsc.HasMajority(voteSet2)) // slightly less than 2/3

	voteSet3 := NewVoteSet()
	voteSet3.AddVote(vote1)
	voteSet3.AddVote(vote2)
	assert.True(vsc.HasMajority(voteSet3)) // slightly above 2/3

	voteSet4 := NewVoteSet()
	voteSet4.AddVote(vote1)
	voteSet4.AddVote(vote2)
	voteSet4.AddVote(vote3)
	voteSet4.AddVote(vote4)
	assert.True(vsc.HasMajority(voteSet4)) // full set
}

func TestValidatorCandidatePool(t *testing.T) {
	assert := assert.New(t)

	sourceAddr1 := common.HexToAddress("0x111")
	stake1Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(1000), MinValidatorStakeDeposit)
	stake1Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(4000), MinValidatorStakeDeposit)

	sourceAddr2 := common.HexToAddress("0x222")
	stake2Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(8000), MinValidatorStakeDeposit)
	stake2Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(9000), MinValidatorStakeDeposit)

	sourceAddr3 := common.HexToAddress("0x333")
	stake3Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(500), MinValidatorStakeDeposit)
	stake3Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(200), MinValidatorStakeDeposit)
	stake3Amount3 := new(big.Int).Mul(new(big.Int).SetUint64(900), MinValidatorStakeDeposit)

	sourceAddr4 := common.HexToAddress("0x444")
	stake4Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(5000), MinValidatorStakeDeposit)

	sourceAddr5 := common.HexToAddress("0x555")
	stake5Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(40000), MinValidatorStakeDeposit)
	stake5Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(30000), MinValidatorStakeDeposit)

	sourceAddr6 := common.HexToAddress("0x666")
	stake6Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(8880), MinValidatorStakeDeposit)
	stake6Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(999), MinValidatorStakeDeposit)
	stake6Amount3 := new(big.Int).Mul(new(big.Int).SetUint64(1110), MinValidatorStakeDeposit)
	stake6Amount4 := new(big.Int).Mul(new(big.Int).SetUint64(222), MinValidatorStakeDeposit)
	stake6Amount5 := new(big.Int).Mul(new(big.Int).SetUint64(333), MinValidatorStakeDeposit)

	invalidStakeAmount := new(big.Int).Mul(new(big.Int).SetInt64(-10), MinValidatorStakeDeposit)
	insufficientStakeAmount := new(big.Int).SetInt64(1000)

	holderAddr1 := common.HexToAddress("0xf01")
	holderAddr2 := common.HexToAddress("0xf02")
	holderAddr3 := common.HexToAddress("0xf03")
	holderAddr4 := common.HexToAddress("0xf04")
	holderAddr5 := common.HexToAddress("0xf05")
	holderAddr6 := common.HexToAddress("0xf06")

	vcp := &ValidatorCandidatePool{}

	log.Infof("")
	log.Infof("----- The following source addresses deposit stakes ----")
	log.Infof("   addr: %v", sourceAddr1)
	log.Infof("   addr: %v", sourceAddr2)
	log.Infof("   addr: %v", sourceAddr3)
	log.Infof("   addr: %v", sourceAddr4)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	assert.Nil(vcp.DepositStake(sourceAddr1, holderAddr1, stake1Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr2, holderAddr1, stake2Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr1, stake3Amount2))

	assert.Nil(vcp.DepositStake(sourceAddr1, holderAddr2, stake1Amount2))
	assert.Nil(vcp.DepositStake(sourceAddr2, holderAddr2, stake2Amount2))
	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr2, stake3Amount2))

	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr3, stake3Amount1))

	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr4, stake3Amount3))
	assert.Nil(vcp.DepositStake(sourceAddr4, holderAddr4, stake4Amount1))

	assert.NotNil(vcp.DepositStake(sourceAddr4, holderAddr2, invalidStakeAmount))
	assert.NotNil(vcp.DepositStake(sourceAddr3, holderAddr6, insufficientStakeAmount))

	assert.True(len(vcp.SortedCandidates) == 4)
	assert.True(vcp.SortedCandidates[0].TotalStake().Cmp(new(big.Int).Mul(new(big.Int).SetUint64(13200), MinValidatorStakeDeposit)) == 0)
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	log.Infof("")
	log.Infof("----- The following source addresses withdraw stakes ---")
	log.Infof("   addr: %v", sourceAddr1)
	log.Infof("   addr: %v", sourceAddr2)
	log.Infof("   addr: %v", sourceAddr4)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	height1 := uint64(100000)
	assert.NotNil(vcp.WithdrawStake(sourceAddr4, holderAddr6, height1)) // no one deposited to holderAddr6 yet
	assert.NotNil(vcp.WithdrawStake(sourceAddr4, holderAddr1, height1)) // sourceAddr4 never deposited to holderAddr1, should fail
	assert.Nil(vcp.WithdrawStake(sourceAddr1, holderAddr2, height1))
	assert.Nil(vcp.WithdrawStake(sourceAddr2, holderAddr2, height1))
	assert.NotNil(vcp.WithdrawStake(sourceAddr2, holderAddr2, height1)) // sourceAddr2 cannot withdraw twice from holderAddr2

	assert.True(len(vcp.SortedCandidates) == 4)
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	log.Infof("")
	log.Infof("----- The following source addresses withdraw stakes ---")
	log.Infof("   addr: %v", sourceAddr3)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	assert.NotNil(vcp.WithdrawStake(sourceAddr1, holderAddr2, height1)) // sourceAddr1 cannot withdraw twice from holderAddr2
	assert.Nil(vcp.WithdrawStake(sourceAddr3, holderAddr2, height1))
	assert.True(len(vcp.SortedCandidates) == 4) // holderAddr1's stake not returned yet, it should still be in the candidate list
	assert.True(vcp.SortedCandidates[3].Holder == holderAddr2)
	assert.True(vcp.SortedCandidates[3].TotalStake().Cmp(Zero) == 0) // All stakes are withdrawn
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	log.Infof("")
	log.Infof("----- The following source addresses deposit stakes ----")
	log.Infof("   addr: %v", sourceAddr5)
	log.Infof("   addr: %v", sourceAddr6)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	assert.Nil(vcp.DepositStake(sourceAddr5, holderAddr5, stake5Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr5, holderAddr5, stake5Amount2))

	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount2))
	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount3))
	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount4))
	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount5))

	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	log.Infof("")
	log.Infof("----- The following source addresses withdraw stakes ---")
	log.Infof("   addr: %v", sourceAddr6)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	height2 := height1 + 500

	assert.NotNil(vcp.WithdrawStake(sourceAddr5, holderAddr6, height2)) // sourceAddr5 never deposited to holderAddr6, so cannot withraw from holderAddr6
	assert.Nil(vcp.WithdrawStake(sourceAddr6, holderAddr6, height2))
	assert.NotNil(vcp.DepositStake(sourceAddr6, holderAddr6, stake6Amount2)) // cannot deposit during the withdrawal locking period
	assert.True(len(vcp.SortedCandidates) == 6)                              // holderAddr6's stake not returned yet, should it should still be in the candidate list
	assert.True(vcp.SortedCandidates[5].Holder == holderAddr6)
	assert.True(vcp.SortedCandidates[5].TotalStake().Cmp(Zero) == 0) // All stakes are withdrawn
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	height3 := height1 + ReturnLockingPeriod - 1
	returnedStakes := vcp.ReturnStakes(height3)
	assert.True(len(returnedStakes) == 0) // has not reached the returned height yet

	log.Infof("")
	log.Infof("----- Return stakes after sufficient amount of time ---")
	log.Infof("   holder addr: %v", holderAddr2)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	height4 := height1 + ReturnLockingPeriod
	returnedStakes = vcp.ReturnStakes(height4)
	assert.True(len(returnedStakes) == 3)
	for _, rs := range returnedStakes {
		log.Infof("Stake returned to Source: %v, Stake: %v", rs.Source, rs.Amount)
	}
	assert.True(len(vcp.SortedCandidates) == 5)
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	height5 := height2 + ReturnLockingPeriod - 1
	returnedStakes = vcp.ReturnStakes(height5)
	assert.True(len(returnedStakes) == 0) // has not reached the returned height yet

	log.Infof("")
	log.Infof("----- Return stakes after sufficient amount of time ---")
	log.Infof("   holder addr: %v", holderAddr6)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	height6 := height2 + ReturnLockingPeriod
	returnedStakes = vcp.ReturnStakes(height6)
	assert.True(len(returnedStakes) == 1)
	assert.True(returnedStakes[0].Amount.Cmp(new(big.Int).Mul(new(big.Int).SetUint64(11544), MinValidatorStakeDeposit)) == 0)
	for _, rs := range returnedStakes {
		log.Infof("Stake returned to Source: %v, Stake: %v", rs.Source, rs.Amount)
	}
	assert.True(len(vcp.SortedCandidates) == 4)
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	log.Infof("")
	log.Infof("----- The following source addresses withdraw stakes ---")
	log.Infof("   addr: %v", sourceAddr1)
	log.Infof("   addr: %v", sourceAddr2)
	log.Infof("--------------------------------------------------------")
	log.Infof("")

	assert.Nil(vcp.WithdrawStake(sourceAddr1, holderAddr1, height6))
	assert.Nil(vcp.WithdrawStake(sourceAddr2, holderAddr1, height6))
	assert.NotNil(vcp.DepositStake(sourceAddr2, holderAddr1, stake2Amount2)) // cannot deposit during the withdrawal locking period
	assert.True(len(vcp.SortedCandidates) == 4)
	assert.True(len(vcp.SortedCandidates[3].Stakes) == 3)
	assert.True(vcp.SortedCandidates[3].TotalStake().Cmp(stake3Amount2) == 0) // Both sourceAddr1 and sourceAddr2 have withdrawn, only sourceAddr3's deposited stake is still effective
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)

	height7 := height6 + ReturnLockingPeriod - 1
	returnedStakes = vcp.ReturnStakes(height7)
	assert.True(len(returnedStakes) == 0)

	height8 := height6 + ReturnLockingPeriod
	returnedStakes = vcp.ReturnStakes(height8)
	assert.True(len(returnedStakes) == 2)
	assert.True(returnedStakes[1].Amount.Cmp(stake1Amount1) == 0)
	assert.True(returnedStakes[0].Amount.Cmp(stake2Amount1) == 0)
	for _, rs := range returnedStakes {
		log.Infof("Stake returned to Source: %v, Stake: %v", rs.Source, rs.Amount)
	}
	assert.True(len(vcp.SortedCandidates) == 4)
	assert.True(len(vcp.SortedCandidates[3].Stakes) == 1)
	checkAndPrintAllSortedCandidates(t, assert, vcp)
	checkAndPrintTopCandidates(t, assert, vcp, 3)
}

func TestValidatorSetUniqueSortedOrder(t *testing.T) {
	assert := assert.New(t)

	ten18 := new(big.Int).SetUint64(1000000000000000000) // 10^18
	stakeAmountA := new(big.Int).Mul(new(big.Int).SetUint64(50000000), ten18)
	stakeAmountB := new(big.Int).Mul(new(big.Int).SetUint64(10000000), ten18)

	sourceAddr1 := common.HexToAddress("0x111")
	holderAddr1 := common.HexToAddress("0x111")
	sourceAddr2 := common.HexToAddress("0x222")
	holderAddr2 := common.HexToAddress("0x222")
	sourceAddr3 := common.HexToAddress("0x333")
	holderAddr3 := common.HexToAddress("0x333")
	sourceAddr4 := common.HexToAddress("0x444")
	holderAddr4 := common.HexToAddress("0x444")
	sourceAddr5 := common.HexToAddress("0x555")
	holderAddr5 := common.HexToAddress("0x555")
	sourceAddr6 := common.HexToAddress("0x666")
	holderAddr6 := common.HexToAddress("0x666")

	vcp := &ValidatorCandidatePool{}
	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr3, stakeAmountA))
	assert.Nil(vcp.DepositStake(sourceAddr1, holderAddr1, stakeAmountA))
	assert.Nil(vcp.DepositStake(sourceAddr5, holderAddr5, stakeAmountB))
	assert.Nil(vcp.DepositStake(sourceAddr2, holderAddr2, stakeAmountA))
	assert.Nil(vcp.DepositStake(sourceAddr6, holderAddr6, stakeAmountB))
	assert.Nil(vcp.DepositStake(sourceAddr4, holderAddr4, stakeAmountA))

	vcp.sortCandidates()
	vcpJson1, _ := json.MarshalIndent(vcp, "", "  ")
	fmt.Printf("VCP after the 1st sort: %v\n\n", string(vcpJson1))

	vcp.sortCandidates()
	vcpJson2, _ := json.MarshalIndent(vcp, "", "  ")
	fmt.Printf("VCP after the 2nd sort: %v\n\n", string(vcpJson2))

	vcp.sortCandidates()
	vcpJson3, _ := json.MarshalIndent(vcp, "", "  ")
	fmt.Printf("VCP after the 3rd sort: %v\n\n", string(vcpJson3))

	vcp.sortCandidates()
	vcpJson4, _ := json.MarshalIndent(vcp, "", "  ")
	fmt.Printf("VCP after the 4th sort: %v\n\n", string(vcpJson4))

	assert.Equal(vcp.SortedCandidates[0].Holder, holderAddr4)
	assert.Equal(vcp.SortedCandidates[1].Holder, holderAddr3)
	assert.Equal(vcp.SortedCandidates[2].Holder, holderAddr2)
	assert.Equal(vcp.SortedCandidates[3].Holder, holderAddr1)
	assert.Equal(vcp.SortedCandidates[4].Holder, holderAddr6)
	assert.Equal(vcp.SortedCandidates[5].Holder, holderAddr5)

	assert.Equal(vcpJson1, vcpJson2)
	assert.Equal(vcpJson2, vcpJson3)
	assert.Equal(vcpJson3, vcpJson4)
}

// ------------------------- Utilities -------------------------

func checkAndPrintAllSortedCandidates(t *testing.T, assert *assert.Assertions, vcp *ValidatorCandidatePool) {
	log.Infof("------ Sorted Candidates ------")
	prevStake := new(big.Int).Mul(new(big.Int).SetUint64(99999999999), MinValidatorStakeDeposit) // some big number
	for _, sh := range vcp.SortedCandidates {
		holder := sh.Holder
		stake := sh.TotalStake()
		log.Infof("Holder: %v, TotalStake: %v, Stakes: %v", holder, stake, sh.Stakes)
		assert.True(prevStake.Cmp(stake) >= 0) // Should be sorted in the descending order
		prevStake = stake
	}
}

func checkAndPrintTopCandidates(t *testing.T, assert *assert.Assertions, vcp *ValidatorCandidatePool, numCandidates int) {
	log.Infof(fmt.Sprintf("------ Top %v Candidates ------", numCandidates))
	topCands := vcp.GetTopStakeHolders(3)
	prevStake := new(big.Int).Mul(new(big.Int).SetUint64(99999999999), MinValidatorStakeDeposit) // some big number
	for _, sh := range topCands {
		holder := sh.Holder
		stake := sh.TotalStake()
		log.Infof("Holder: %v, Stake: %v", holder, stake)
		assert.True(prevStake.Cmp(stake) > 0) // Should be sorted in the descending order
		prevStake = stake
	}
}
