package consensus

// Validator contains the public information of a validator.
type Validator string

// ID return the identifier of the validator.
func (v Validator) ID() string {
	return string(v)
}

// ValidatorSet represents a set of validators.
type ValidatorSet []Validator

// ValidatorManager is the component for managing validator related logic for consensus engine.
type ValidatorManager interface {
	GetProposerForHeight(height uint32) Validator
}

var _ ValidatorManager = &FixedValidatorManager{}

// FixedValidatorManager is an implementation of ValidatorManager interface that selects a fixed validator as the proposer.
type FixedValidatorManager struct {
	validators ValidatorSet
}

// NewFixedValidatorManager creates an instance of FixedValidatorManager.
func NewFixedValidatorManager(validators ValidatorSet) *FixedValidatorManager {
	m := &FixedValidatorManager{}
	m.validators = validators
	return m
}

// GetProposerForHeight implements ValidatorManager interface.
func (m *FixedValidatorManager) GetProposerForHeight(height uint32) Validator {
	if len(m.validators) == 0 {
		panic("No validators have been added")
	}
	return m.validators[0]
}
