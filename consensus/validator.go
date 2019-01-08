package consensus

import (
	"fmt"
	"math/big"
	"math/rand"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
)

//
// -------------------------------- FixedValidatorManager ----------------------------------
//
var _ core.ValidatorManager = &FixedValidatorManager{}

// FixedValidatorManager is an implementation of ValidatorManager interface that selects a fixed validator as the proposer.
type FixedValidatorManager struct {
	consensus core.ConsensusEngine
}

// NewFixedValidatorManager creates an instance of FixedValidatorManager.
func NewFixedValidatorManager() *FixedValidatorManager {
	m := &FixedValidatorManager{
		consensus: nil,
	}
	return m
}

// SetConsensusEngine mplements ValidatorManager interface.
func (m *FixedValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {
	m.consensus = consensus
}

// GetProposer implements ValidatorManager interface.
func (m *FixedValidatorManager) GetProposer(blockHash common.Hash, _ uint64) core.Validator {
	valSet := m.GetValidatorSet(blockHash)
	if valSet.Size() == 0 {
		panic("No validators have been added")
	}

	return valSet.Validators()[0]
}

// GetValidatorSet returns the validator set for given block hash.
func (m *FixedValidatorManager) GetValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidators(m.consensus, blockHash)
	return valSet
}

//
// -------------------------------- RotatingValidatorManager ----------------------------------
//
var _ core.ValidatorManager = &RotatingValidatorManager{}

// RotatingValidatorManager is an implementation of ValidatorManager interface that selects a random validator as
// the proposer using validator's stake as weight.
type RotatingValidatorManager struct {
	consensus core.ConsensusEngine
}

// NewRotatingValidatorManager creates an instance of RotatingValidatorManager.
func NewRotatingValidatorManager() *RotatingValidatorManager {
	m := &RotatingValidatorManager{}
	return m
}

// SetConsensusEngine mplements ValidatorManager interface.
func (m *RotatingValidatorManager) SetConsensusEngine(consensus core.ConsensusEngine) {
	m.consensus = consensus
}

// GetProposer implements ValidatorManager interface.
func (m *RotatingValidatorManager) GetProposer(blockHash common.Hash, epoch uint64) core.Validator {
	valSet := m.GetValidatorSet(blockHash)
	if valSet.Size() == 0 {
		panic("No validators have been added")
	}

	totalStake := valSet.TotalStake()
	scalingFactor := new(big.Int).Div(totalStake, common.BigMaxUint32)
	scalingFactor = new(big.Int).Add(scalingFactor, common.Big1)
	scaledTotalStake := scaleDown(totalStake, scalingFactor)

	// TODO: replace with more secure randomness.
	rnd := rand.New(rand.NewSource(int64(epoch)))
	r := randUint64(rnd, scaledTotalStake)
	curr := uint64(0)
	validators := valSet.Validators()
	for _, v := range validators {
		curr += scaleDown(v.Stake(), scalingFactor)
		if r < curr {
			return v
		}
	}

	// Should not reach here.
	panic("Failed to randomly select a validator")
}

// GetValidatorSet returns the validator set for given epoch.
func (m *RotatingValidatorManager) GetValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidators(m.consensus, blockHash)
	return valSet
}

//
// -------------------------------- Utilities ----------------------------------
//

func selectTopStakeHoldersAsValidators(consensus core.ConsensusEngine, blockHash common.Hash) *core.ValidatorSet {
	vcp, err := consensus.GetLedger().GetValidatorCandidatePool(blockHash)
	if err != nil {
		panic(fmt.Sprintf("Failed to get the validator candiate pool: %v", err))
	}

	maxNumValidators := viper.GetInt(common.CfgConsensusMaxNumValidators)
	topStakeHolders := vcp.GetTopStakeHolders(maxNumValidators)

	valSet := core.NewValidatorSet()
	for _, stakeHolder := range topStakeHolders {
		valAddr := stakeHolder.Holder.Hex()
		valStake := stakeHolder.TotalStake()
		validator := core.NewValidator(valAddr, valStake)
		valSet.AddValidator(validator)
	}

	return valSet
}

// Generate a random uint64 in [0, max)
func randUint64(rnd *rand.Rand, max uint64) uint64 {
	const maxInt64 uint64 = 1<<63 - 1
	if max <= maxInt64 {
		return uint64(rnd.Int63n(int64(max)))
	}
	for {
		r := rnd.Uint64()
		if r < max {
			return r
		}
	}
}

func scaleDown(x *big.Int, scalingFactor *big.Int) uint64 {
	if scalingFactor.Cmp(common.Big0) == 0 {
		panic("scalingFactor is zero")
	}
	scaledX := new(big.Int).Div(x, scalingFactor)
	scaledXUint64 := scaledX.Uint64()
	return scaledXUint64
}
