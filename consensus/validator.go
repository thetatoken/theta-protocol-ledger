package consensus

import (
	"math/rand"

	"github.com/thetatoken/ukulele/core"
)

//
// -------------------------------- FixedValidatorManager ----------------------------------
//
var _ core.ValidatorManager = &FixedValidatorManager{}

// FixedValidatorManager is an implementation of ValidatorManager interface that selects a fixed validator as the proposer.
type FixedValidatorManager struct {
	validators *core.ValidatorSet
}

// NewFixedValidatorManager creates an instance of FixedValidatorManager.
func NewFixedValidatorManager(validators *core.ValidatorSet) *FixedValidatorManager {
	m := &FixedValidatorManager{}
	m.validators = validators.Copy()
	return m
}

// GetProposerForEpoch implements ValidatorManager interface.
func (m *FixedValidatorManager) GetProposerForEpoch(epoch uint32) core.Validator {
	if m.validators.Size() == 0 {
		panic("No validators have been added")
	}
	return m.validators.Validators()[0]
}

// GetValidatorSetForEpoch returns the validator set for given epoch.
func (m *FixedValidatorManager) GetValidatorSetForEpoch(_ uint32) *core.ValidatorSet {
	return m.validators
}

//
// -------------------------------- RotatingValidatorManager ----------------------------------
//
var _ core.ValidatorManager = &RotatingValidatorManager{}

// RotatingValidatorManager is an implementation of ValidatorManager interface that selects a random validator as
// the proposer using validator's stake as weight.
type RotatingValidatorManager struct {
	validators *core.ValidatorSet
}

// NewRotatingValidatorManager creates an instance of RotatingValidatorManager.
func NewRotatingValidatorManager(validators *core.ValidatorSet) *RotatingValidatorManager {
	m := &RotatingValidatorManager{}
	m.validators = validators.Copy()
	return m
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

// GetProposerForEpoch implements ValidatorManager interface.
func (m *RotatingValidatorManager) GetProposerForEpoch(epoch uint32) core.Validator {
	if m.validators.Size() == 0 {
		panic("No validators have been added")
	}
	// TODO: replace with more secure randomness.
	rnd := rand.New(rand.NewSource(int64(epoch)))
	totalStake := m.validators.TotalStake()
	r := randUint64(rnd, totalStake)
	curr := uint64(0)
	validators := m.validators.Validators()
	for _, v := range validators {
		curr += v.Stake()
		if r < curr {
			return v
		}
	}
	// Should not reach here.
	panic("Failed to randomly select a validator")
}

// GetValidatorSetForEpoch returns the validator set for given epoch.
func (m *RotatingValidatorManager) GetValidatorSetForEpoch(_ uint32) *core.ValidatorSet {
	return m.validators
}
