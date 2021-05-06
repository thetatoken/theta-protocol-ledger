package core

import (
	"fmt"

	"github.com/thetatoken/theta/common"
)

//
// ------- RewardBeneficiary ------- //
//

type RewardDistribution struct {
	StakeHolder     common.Address // the stake delegate, i.e. a guardian node or an elite edge node address
	Beneficiary     common.Address // the beneficiary to split the reward
	SplitBasisPoint uint           // An integer between 0 and 10000, representing the fraction of the reward the beneficiary should get (in terms of 1/10000), https://en.wikipedia.org/wiki/Basis_point
}

func NewRewardDistribution(stakeHolder common.Address, beneficiary common.Address, splitBasisPoint uint) (*RewardDistribution, error) {
	if splitBasisPoint > 10000 {
		return nil, fmt.Errorf("split basis point cannot exceed 10000")
	}

	return &RewardDistribution{
		StakeHolder:     stakeHolder,
		Beneficiary:     beneficiary,
		SplitBasisPoint: splitBasisPoint,
	}, nil
}
