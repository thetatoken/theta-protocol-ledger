package core

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "core"})

var (
	// ErrValidatorNotFound for ID is not found in validator set.
	ErrValidatorNotFound = errors.New("ValidatorNotFound")
)

// Validator contains the public information of a validator.
type Validator struct {
	Address common.Address
	Stake   *big.Int
}

// NewValidator creates a new validator instance.
func NewValidator(addressStr string, stake *big.Int) Validator {
	address := common.HexToAddress(addressStr)
	return Validator{address, stake}
}

// ID returns the ID of the validator, which is the string representation of its address.
func (v Validator) ID() common.Address {
	return v.Address
}

// Equals checks whether the validator is the same as another validator
func (v Validator) Equals(x Validator) bool {
	if v.Address != x.Address {
		return false
	}
	if v.Stake.Cmp(x.Stake) != 0 {
		return false
	}
	return true
}

// String represents the string representation of the validator
func (v Validator) String() string {
	return fmt.Sprintf("{ID: %v, Stake: %v}", v.ID(), v.Stake)
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

// SetValidators sets validators
func (s *ValidatorSet) SetValidators(validators []Validator) {
	s.validators = validators
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

// Equals checks whether the validator set is the same as another validator set
func (s *ValidatorSet) Equals(t *ValidatorSet) bool {
	numVals := len(s.validators)
	if numVals != len(t.validators) {
		return false
	}
	for i := 0; i < numVals; i++ {
		if !s.validators[i].Equals(t.validators[i]) {
			return false
		}
	}
	return true
}

// String represents the string representation of the validator set
func (s *ValidatorSet) String() string {
	return fmt.Sprintf("{Validators: %v}", s.validators)
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
		ret = new(big.Int).Add(ret, v.Stake)
	}
	return ret
}

// HasMajorityVotes checks whether a vote set has reach majority.
func (s *ValidatorSet) HasMajorityVotes(votes []Vote) bool {
	votedStake := new(big.Int).SetUint64(0)
	for _, vote := range votes {
		validator, err := s.GetValidator(vote.ID)
		if err == nil {
			votedStake = new(big.Int).Add(votedStake, validator.Stake)
		}
	}

	three := new(big.Int).SetUint64(3)
	two := new(big.Int).SetUint64(2)
	lhs := new(big.Int)
	rhs := new(big.Int)

	//return votedStake*3 > s.TotalStake()*2
	return lhs.Mul(votedStake, three).Cmp(rhs.Mul(s.TotalStake(), two)) > 0
}

// HasMajority checks whether a vote set has reach majority.
func (s *ValidatorSet) HasMajority(votes *VoteSet) bool {
	return s.HasMajorityVotes(votes.Votes())
}

// Validators returns a slice of validators.
func (s *ValidatorSet) Validators() []Validator {
	return s.validators
}

//
// ------- ValidatorCandidatePool ------- //
//

var (
	MinValidatorStakeDeposit     *big.Int
	MinValidatorStakeDeposit200K *big.Int
)

func init() {
	// Each stake deposit needs to be at least 2,000,000 Theta
	MinValidatorStakeDeposit = new(big.Int).Mul(new(big.Int).SetUint64(2000000), new(big.Int).SetUint64(1000000000000000000))

	// Minimum Validator stake deposit reduced to 200,000 Theta
	MinValidatorStakeDeposit200K = new(big.Int).Mul(new(big.Int).SetUint64(200000), new(big.Int).SetUint64(1000000000000000000))
}

type ValidatorCandidatePool struct {
	SortedCandidates []*StakeHolder
}

func (vcp *ValidatorCandidatePool) FindStakeDelegate(delegateAddr common.Address) *StakeHolder {
	for _, candidate := range vcp.SortedCandidates {
		if candidate.Holder == delegateAddr {
			return candidate
		}
	}
	return nil
}

func (vcp *ValidatorCandidatePool) GetTopStakeHolders(maxNumStakeHolders int) []*StakeHolder {
	n := len(vcp.SortedCandidates)
	if n > maxNumStakeHolders {
		n = maxNumStakeHolders
	}
	return vcp.SortedCandidates[:n]
}

func (vcp *ValidatorCandidatePool) DepositStake(source common.Address, holder common.Address, amount *big.Int, blockHeight uint64) (err error) {
	minValidatorStake := MinValidatorStakeDeposit
	if blockHeight >= common.HeightValidatorStakeChangedTo200K {
		minValidatorStake = MinValidatorStakeDeposit200K
	}
	if amount.Cmp(minValidatorStake) < 0 {
		return fmt.Errorf("insufficient stake: %v", amount)
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
		newCandidate := NewStakeHolder(holder, []*Stake{NewStake(source, amount)})
		vcp.SortedCandidates = append(vcp.SortedCandidates, newCandidate)
	}

	vcp.sortCandidates()

	return nil
}

func (vcp *ValidatorCandidatePool) WithdrawStake(source common.Address, holder common.Address, currentHeight uint64) error {
	matchedHolderFound := false
	for _, candidate := range vcp.SortedCandidates {
		if candidate.Holder == holder {
			matchedHolderFound = true
			_, err := candidate.withdrawStake(source, currentHeight)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		return fmt.Errorf("No matched stake holder address found: %v", holder)
	}

	vcp.sortCandidates()

	return nil
}

func (vcp *ValidatorCandidatePool) ReturnStakes(currentHeight uint64) []*Stake {
	returnedStakes := []*Stake{}

	// need to iterate in the reverse order, since we may delete elemements
	// from the slice while iterating through it
	for cidx := len(vcp.SortedCandidates) - 1; cidx >= 0; cidx-- {
		candidate := vcp.SortedCandidates[cidx]
		numStakeSources := len(candidate.Stakes)
		for sidx := numStakeSources - 1; sidx >= 0; sidx-- { // similar to the outer loop, need to iterate in the reversed order
			stake := candidate.Stakes[sidx]
			if (stake.Withdrawn) && (currentHeight >= stake.ReturnHeight) {
				logger.Printf("Stake to be returned: source = %v, amount = %v", stake.Source, stake.Amount)
				source := stake.Source
				returnedStake, err := candidate.returnStake(source, currentHeight)
				if err != nil {
					logger.Errorf("Failed to return stake: %v, error: %v", source, err)
					continue
				}
				returnedStakes = append(returnedStakes, returnedStake)
			}
		}

		if len(candidate.Stakes) == 0 { // the candidate's stake becomes zero, no need to keep track of the candidate anymore
			vcp.SortedCandidates = append(vcp.SortedCandidates[:cidx], vcp.SortedCandidates[cidx+1:]...)
		}
	}

	vcp.sortCandidates()

	return returnedStakes
}

func (vcp *ValidatorCandidatePool) sortCandidates() {
	sort.Slice(vcp.SortedCandidates[:], func(i, j int) bool { // descending order in (totalStake, holderAddress)
		stakeCmp := vcp.SortedCandidates[i].TotalStake().Cmp(vcp.SortedCandidates[j].TotalStake())
		if stakeCmp == 0 {
			return strings.Compare(vcp.SortedCandidates[i].Holder.Hex(), vcp.SortedCandidates[j].Holder.Hex()) >= 0
		}
		return stakeCmp > 0
	})
}

// func (vcp *ValidatorCandidatePool) sortCandidates() {
// 	sort.Slice(vcp.SortedCandidates[:], func(i, j int) bool { // descending order in totalStake
// 		stakeCmp := vcp.SortedCandidates[i].TotalStake().Cmp(vcp.SortedCandidates[j].TotalStake())
// 		return stakeCmp >= 0
// 	})
// }
