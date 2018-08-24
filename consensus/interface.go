package consensus

import (
	"context"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

// Engine is the interface of a consensus engine.
type Engine interface {
	ID() string
	Chain() *blockchain.Chain
	Network() p2p.Network
	HandleMessage(peerID string, msg p2ptypes.Message)
	FinalizedBlocks() chan *blockchain.Block

	Start(context.Context)
	Stop()
	Wait()
}
