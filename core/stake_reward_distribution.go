package core

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

//
// ------- RewardBeneficiary ------- //
//

type RewardDistribution struct {
	StakeHolder     common.Address // the stake delegator, i.e. a guardian node or an elite edge node address
	Beneficiary     common.Address // the beneficiary to split the reward
	SplitBasisPoint uint           // An integer between 0 and 10000, representing the fraction of the reward the beneficiary should get (in terms of 1/10000), https://en.wikipedia.org/wiki/Basis_point
}

func newRewardDistribution(stakeHolder common.Address, beneficiary common.Address, splitBasisPoint uint) (*RewardDistribution, error) {
	if splitBasisPoint > 10000 {
		return nil, fmt.Errorf("split basis point cannot exceed 10000")
	}

	return &RewardDistribution{
		StakeHolder:     stakeHolder,
		Beneficiary:     beneficiary,
		SplitBasisPoint: splitBasisPoint,
	}, nil
}

type StakeRewardDistributionRuleSet struct {
	SortedRewardDistribution []*RewardDistribution // reward distribution sorted by holder address.
}

// NewStakeRewardDistributionRuleSet creates a new instance of StakeRewardDistributionRuleSet.
func NewStakeRewardDistributionRuleSet() *StakeRewardDistributionRuleSet {
	return &StakeRewardDistributionRuleSet{
		SortedRewardDistribution: []*RewardDistribution{},
	}
}

// Upsert update or inserts a stake reward distribution to the rule set
func (srdr *StakeRewardDistributionRuleSet) Upsert(rd *RewardDistribution) bool {
	k := sort.Search(srdr.Len(), func(i int) bool {
		return bytes.Compare(srdr.SortedRewardDistribution[i].StakeHolder.Bytes(), rd.StakeHolder.Bytes()) >= 0
	})

	if k == srdr.Len() {
		srdr.SortedRewardDistribution = append(srdr.SortedRewardDistribution, rd)
		return true
	}

	// stake reward distribution is already added, update it
	if srdr.SortedRewardDistribution[k].StakeHolder == rd.StakeHolder {
		srdr.SortedRewardDistribution[k] = rd
		return true
	}

	srdr.SortedRewardDistribution = append(srdr.SortedRewardDistribution, nil)
	copy(srdr.SortedRewardDistribution[k+1:], srdr.SortedRewardDistribution[k:])
	srdr.SortedRewardDistribution[k] = rd
	return true
}

// Remove removes an elite edge node from the pool; returns false if guardian is not found.
func (srdr *StakeRewardDistributionRuleSet) Remove(stakeHolder common.Address) bool {
	k := sort.Search(srdr.Len(), func(i int) bool {
		return bytes.Compare(srdr.SortedRewardDistribution[i].StakeHolder.Bytes(), stakeHolder.Bytes()) >= 0
	})

	if k == srdr.Len() || bytes.Compare(srdr.SortedRewardDistribution[k].StakeHolder.Bytes(), stakeHolder.Bytes()) != 0 {
		return false
	}
	srdr.SortedRewardDistribution = append(srdr.SortedRewardDistribution[:k], srdr.SortedRewardDistribution[k+1:]...)
	return true
}

// Contains checks if given address is in the pool.
func (srdr *StakeRewardDistributionRuleSet) Contains(stakeHolder common.Address) bool {
	k := sort.Search(srdr.Len(), func(i int) bool {
		return bytes.Compare(srdr.SortedRewardDistribution[i].StakeHolder.Bytes(), stakeHolder.Bytes()) >= 0
	})

	if k == srdr.Len() || srdr.SortedRewardDistribution[k].StakeHolder != stakeHolder {
		return false
	}
	return true
}

// IndexWithStakeHolderAddress returns index of a stake holder address in the pool. Returns -1 if not found.
func (srdr *StakeRewardDistributionRuleSet) IndexWithStakeHolderAddress(addr common.Address) int {
	for i, rd := range srdr.SortedRewardDistribution {
		if rd.StakeHolder == addr {
			return i
		}
	}
	return -1
}

// GetWithStakeHolderAddress returns the reward distribution of a stake holder address in the pool. Returns nil if not found.
func (srdr *StakeRewardDistributionRuleSet) GetWithStakeHolderAddress(addr common.Address) *RewardDistribution {
	for _, rd := range srdr.SortedRewardDistribution {
		if rd.StakeHolder == addr {
			return rd
		}
	}
	return nil
}

// Implements sort.Interface for Guardians based on
// the Address field.
func (srdr *StakeRewardDistributionRuleSet) Len() int {
	return len(srdr.SortedRewardDistribution)
}
func (srdr *StakeRewardDistributionRuleSet) Swap(i, j int) {
	srdr.SortedRewardDistribution[i], srdr.SortedRewardDistribution[j] = srdr.SortedRewardDistribution[j], srdr.SortedRewardDistribution[i]
}
func (srdr *StakeRewardDistributionRuleSet) Less(i, j int) bool {
	return bytes.Compare(srdr.SortedRewardDistribution[i].StakeHolder.Bytes(), srdr.SortedRewardDistribution[j].StakeHolder.Bytes()) < 0
}

// Hash calculates the hash of elite edge node pool.
func (srdr *StakeRewardDistributionRuleSet) Hash() common.Hash {
	raw, err := rlp.EncodeToBytes(srdr)
	if err != nil {
		logger.Panic(err)
	}
	return crypto.Keccak256Hash(raw)
}
