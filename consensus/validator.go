package consensus

import (
	"errors"

	"github.com/thetatoken/ukulele/blockchain"
)

var (
	// ErrValidatorNotFound for ID is not found in validator set.
	ErrValidatorNotFound = errors.New("ValidatorNotFound")
)

// Validator contains the public information of a validator.
type Validator struct {
	id    string
	stake uint64
}

// NewValidator creates a new validator instance.
func NewValidator(id string, stake uint64) Validator {
	return Validator{id, stake}
}

// ID return the identifier of the validator.
func (v Validator) ID() string {
	return v.id
}

// Stake returns the stake of the validator.
func (v Validator) Stake() uint64 {
	return v.stake
}

// ValidatorSet represents a set of validators.
type ValidatorSet struct {
	validators []Validator
}

// NewValidatorSet returns a new instance of ValidatorSet.
func NewValidatorSet() *ValidatorSet {
	return &ValidatorSet{
		validators: []Validator{},
	}
}

// Copy creates a copy of this validator set.
func (s *ValidatorSet) Copy() *ValidatorSet {
	ret := NewValidatorSet()
	for _, v := range s.Validators() {
		ret.AddValidator(v)
	}
	return ret
}

// Size returns the number of the validators in the validator set.
func (s *ValidatorSet) Size() int {
	return len(s.validators)
}

// GetValidator returns a validator if a matching ID is found.
func (s *ValidatorSet) GetValidator(id string) (Validator, error) {
	for _, v := range s.validators {
		if v.ID() == id {
			return v, nil
		}
	}
	return Validator{}, ErrValidatorNotFound
}

// AddValidator adds a validator to the validator set.
func (s *ValidatorSet) AddValidator(validator Validator) {
	s.validators = append(s.validators, validator)
}

// TotalStake returns the total stake of the validators in the set.
func (s *ValidatorSet) TotalStake() uint64 {
	ret := uint64(0)
	for _, v := range s.validators {
		ret += v.Stake()
	}
	return ret
}

// HasMajority checks whether a vote set has reach majority.
func (s *ValidatorSet) HasMajority(votes *blockchain.VoteSet) bool {
	quorum := s.TotalStake()*2/3 + 1
	votedStake := uint64(0)
	for _, vote := range votes.Votes() {
		validator, err := s.GetValidator(vote.ID)
		if err == nil {
			votedStake += validator.Stake()
		}
	}
	return votedStake >= quorum
}

// Validators returns a slice of validators.
func (s *ValidatorSet) Validators() []Validator {
	return s.validators
}

// ValidatorManager is the component for managing validator related logic for consensus engine.
type ValidatorManager interface {
	GetProposerForHeight(height uint32) Validator
	GetValidatorSetForHeight(height uint32) *ValidatorSet
}

var _ ValidatorManager = &FixedValidatorManager{}

// FixedValidatorManager is an implementation of ValidatorManager interface that selects a fixed validator as the proposer.
type FixedValidatorManager struct {
	validators *ValidatorSet
}

// NewFixedValidatorManager creates an instance of FixedValidatorManager.
func NewFixedValidatorManager(validators *ValidatorSet) *FixedValidatorManager {
	m := &FixedValidatorManager{}
	m.validators = validators.Copy()
	return m
}

// GetProposerForHeight implements ValidatorManager interface.
func (m *FixedValidatorManager) GetProposerForHeight(height uint32) Validator {
	if m.validators.Size() == 0 {
		panic("No validators have been added")
	}
	return m.validators.validators[0]
}

// GetValidatorSetForHeight returns the validator set for given height.
func (m *FixedValidatorManager) GetValidatorSetForHeight(_ uint32) *ValidatorSet {
	return m.validators
}
