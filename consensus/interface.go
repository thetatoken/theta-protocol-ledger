package consensus

import (
	"context"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/p2p"
)

// Engine is the interface of a consensus engine.
type Engine interface {
	ID() string
	Chain() *blockchain.Chain
	GetTip() *core.ExtendedBlock
	Network() p2p.Network
	GetEpoch() uint32
	GetValidatorManager() *ValidatorManager
	AddMessage(msg interface{})
	FinalizedBlocks() chan *core.Block

	Start(context.Context)
	Stop()
	Wait()
}
