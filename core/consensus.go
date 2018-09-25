package core

import (
	"context"
)

// ConsensusEngine is the interface of a consensus engine.
type ConsensusEngine interface {
	ID() string
	GetTip() *ExtendedBlock
	GetEpoch() uint32
	GetValidatorManager() ValidatorManager
	AddMessage(msg interface{})
	FinalizedBlocks() chan *Block

	Start(context.Context)
	Stop()
	Wait()
}

// ValidatorManager is the component for managing validator related logic for consensus engine.
type ValidatorManager interface {
	GetProposerForEpoch(epoch uint32) Validator
	GetValidatorSetForEpoch(epoch uint32) *ValidatorSet
}
