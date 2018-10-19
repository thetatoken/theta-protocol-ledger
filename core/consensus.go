package core

import (
	"github.com/thetatoken/ukulele/crypto"
)

// ConsensusEngine is the interface of a consensus engine.
type ConsensusEngine interface {
	ID() string
	PrivateKey() *crypto.PrivateKey
	GetTip() *ExtendedBlock
	GetEpoch() uint64
	AddMessage(msg interface{})
	FinalizedBlocks() chan *Block
}

// ValidatorManager is the component for managing validator related logic for consensus engine.
type ValidatorManager interface {
	GetProposerForEpoch(epoch uint64) Validator
	GetValidatorSetForEpoch(epoch uint64) *ValidatorSet
}
