package consensus

import (
	"context"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/p2p"
)

// Engine is the interface of a consensus engine.
type Engine interface {
	ID() string
	Chain() *blockchain.Chain
	Network() p2p.Network
	AddMessage(msg interface{})
	FinalizedBlocks() chan *blockchain.Block

	Start(context.Context)
	Stop()
	Wait()
}
