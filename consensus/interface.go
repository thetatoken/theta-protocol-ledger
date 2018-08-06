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
	Start(ctx context.Context)
	HandleMessage(network p2p.Network, msg interface{})
	FinalizedBlocks() chan *blockchain.Block
}
