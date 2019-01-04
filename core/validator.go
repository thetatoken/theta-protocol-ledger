package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/thetatoken/ukulele/common"
)

var (
	// ErrValidatorNotFound for ID is not found in validator set.
	ErrValidatorNotFound = errors.New("ValidatorNotFound")
)

// Validator contains the public information of a validator.
type Validator struct {
	address common.Address
	stake   *big.Int
}

// NewValidator creates a new validator instance.
func NewValidator(addressStr string, stake *big.Int) Validator {
	address := common.HexToAddress(addressStr)
	return Validator{address, stake}
}

// Address returns the address of the validator.
func (v Validator) Address() common.Address {
	return v.address
}

// ID returns the ID of the validator, which is the string representation of its address.
func (v Validator) ID() common.Address {
	return v.address
}

// Stake returns the stake of the validator.
func (v Validator) Stake() *big.Int {
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

// ByID implements sort.Interface for ValidatorSet based on ID.
type ByID []Validator

func (b ByID) Len() int           { return len(b) }
func (b ByID) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b ByID) Less(i, j int) bool { return bytes.Compare(b[i].ID().Bytes(), b[j].ID().Bytes()) < 0 }

// GetValidator returns a validator if a matching ID is found.
func (s *ValidatorSet) GetValidator(id common.Address) (Validator, error) {
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
	sort.Sort(ByID(s.validators))
}

// TotalStake returns the total stake of the validators in the set.
func (s *ValidatorSet) TotalStake() *big.Int {
	ret := new(big.Int).SetUint64(0)
	for _, v := range s.validators {
		ret = new(big.Int).Add(ret, v.Stake())
	}
	return ret
}

// HasMajority checks whether a vote set has reach majority.
func (s *ValidatorSet) HasMajority(votes *VoteSet) bool {
	votedStake := new(big.Int).SetUint64(0)
	for _, vote := range votes.Votes() {
		validator, err := s.GetValidator(vote.ID)
		if err == nil {
			votedStake = new(big.Int).Add(votedStake, validator.Stake())
		}
	}

	three := new(big.Int).SetUint64(3)
	two := new(big.Int).SetUint64(2)
	lhs := new(big.Int)
	rhs := new(big.Int)

	//return votedStake*3 > s.TotalStake()*2
	return lhs.Mul(votedStake, three).Cmp(rhs.Mul(s.TotalStake(), two)) > 0
}

// Validators returns a slice of validators.
func (s *ValidatorSet) Validators() []Validator {
	return s.validators
}

//
// ------- ValidatorCandidatePool ------- //
//

var (
	MinValidatorStakeDeposit *big.Int
)

func init() {
	// Each stake deposit needs to be at least 10,000,000 Theta
	MinValidatorStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(10000000), new(big.Int).SetUint64(1000000000000000000))
}

type ValidatorCandidatePool struct {
	SortedCandidates []*StakeHolder
}

func (vcp *ValidatorCandidatePool) DepositStake(source common.Address, holder common.Address, amount *big.Int) (err error) {
	if amount.Cmp(MinValidatorStakeDeposit) < 0 {
		return fmt.Errorf("Insufficient stake: %v", amount)
	}

	matchedHolderFound := false
	for _, candidate := range vcp.SortedCandidates {
		if candidate.Holder == holder {
			matchedHolderFound = true
			err = candidate.depositStake(source, amount)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		newCandidate := newStakeHolder(holder, []*Stake{newStake(source, amount)})
		vcp.SortedCandidates = append(vcp.SortedCandidates, newCandidate)
	}

	sort.Slice(vcp.SortedCandidates[:], func(i, j int) bool { // descending order
		return vcp.SortedCandidates[i].totalStake().Cmp(vcp.SortedCandidates[j].totalStake()) >= 0
	})

	return nil
}

func (vcp *ValidatorCandidatePool) WithdrawStake(source common.Address, holder common.Address) (withdrawnAmount *big.Int, err error) {
	withdrawnAmount = new(big.Int).SetUint64(0)

	matchedHolderFound := false
	for _, candidate := range vcp.SortedCandidates {
		if candidate.Holder == holder {
			matchedHolderFound = true
			withdrawnAmount, err = candidate.withdrawStake(source)
			if err != nil {
				return withdrawnAmount, err
			}
			break
		}
	}

	if !matchedHolderFound {
		return withdrawnAmount, fmt.Errorf("No matched stake holder address found: %v", holder)
	}

	sort.Slice(vcp.SortedCandidates[:], func(i, j int) bool { // descending order
		return vcp.SortedCandidates[i].totalStake().Cmp(vcp.SortedCandidates[j].totalStake()) >= 0
	})

	return withdrawnAmount, nil
}
