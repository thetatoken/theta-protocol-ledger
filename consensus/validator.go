package consensus

import (
	"math/big"
	"math/rand"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
)

const MaxValidatorCount int = 31

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
	return m.getProposerFromValidators(m.GetValidatorSet(blockHash))
}

// GetNextProposer implements ValidatorManager interface.
func (m *FixedValidatorManager) GetNextProposer(blockHash common.Hash, _ uint64) core.Validator {
	return m.getProposerFromValidators(m.GetNextValidatorSet(blockHash))
}

func (m *FixedValidatorManager) getProposerFromValidators(valSet *core.ValidatorSet) core.Validator {
	if valSet.Size() == 0 {
		log.Panic("No validators have been added")
	}

	return valSet.Validators()[0]
}

// GetValidatorSet returns the validator set for given block hash.
func (m *FixedValidatorManager) GetValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidatorsForBlock(m.consensus, blockHash, false)
	return valSet
}

// GetNextValidatorSet returns the validator set for given block hash's next block.
func (m *FixedValidatorManager) GetNextValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidatorsForBlock(m.consensus, blockHash, true)
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
	return m.getProposerFromValidators(m.GetValidatorSet(blockHash), epoch)
}

// GetNextProposer implements ValidatorManager interface.
func (m *RotatingValidatorManager) GetNextProposer(blockHash common.Hash, epoch uint64) core.Validator {
	return m.getProposerFromValidators(m.GetNextValidatorSet(blockHash), epoch)
}

func (m *RotatingValidatorManager) getProposerFromValidators(valSet *core.ValidatorSet, epoch uint64) core.Validator {
	if valSet.Size() == 0 {
		log.Panic("No validators have been added")
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
		curr += scaleDown(v.Stake, scalingFactor)
		if r < curr {
			return v
		}
	}

	// Should not reach here.
	log.Panic("Failed to randomly select a validator")
	panic("Should not reach here")
}

// GetValidatorSet returns the validator set for given block.
func (m *RotatingValidatorManager) GetValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidatorsForBlock(m.consensus, blockHash, false)
	return valSet
}

// GetNextValidatorSet returns the validator set for given block's next block.
func (m *RotatingValidatorManager) GetNextValidatorSet(blockHash common.Hash) *core.ValidatorSet {
	valSet := selectTopStakeHoldersAsValidatorsForBlock(m.consensus, blockHash, true)
	return valSet
}

//
// -------------------------------- Utilities ----------------------------------
//

func SelectTopStakeHoldersAsValidators(vcp *core.ValidatorCandidatePool) *core.ValidatorSet {
	maxNumValidators := MaxValidatorCount
	topStakeHolders := vcp.GetTopStakeHolders(maxNumValidators)

	valSet := core.NewValidatorSet()
	for _, stakeHolder := range topStakeHolders {
		valAddr := stakeHolder.Holder.Hex()
		valStake := stakeHolder.TotalStake()
		if valStake.Cmp(core.Zero) == 0 {
			continue
		}
		validator := core.NewValidator(valAddr, valStake)
		valSet.AddValidator(validator)
	}

	return valSet
}

func selectTopStakeHoldersAsValidatorsForBlock(consensus core.ConsensusEngine, blockHash common.Hash, isNext bool) *core.ValidatorSet {
	vcp, err := consensus.GetLedger().GetFinalizedValidatorCandidatePool(blockHash, isNext)
	if err != nil {
		log.Panicf("Failed to get the validator candidate pool, blockHash: %v, isNext: %v, err: %v", blockHash.Hex(), isNext, err)
	}
	if vcp == nil {
		log.Panic("Failed to retrieve the validator candidate pool")
	}

	return SelectTopStakeHoldersAsValidators(vcp)
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
		log.Panic("scalingFactor is zero")
	}
	scaledX := new(big.Int).Div(x, scalingFactor)
	scaledXUint64 := scaledX.Uint64()
	return scaledXUint64
}
