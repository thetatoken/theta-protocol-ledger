package p2pl

import (
	"context"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p/types"
)

//
// MessageHandler interface
//
type MessageHandler interface {

	// GetChannelIDs returns the list channelIDs that the message handler needs to handle
	GetChannelIDs() []common.ChannelIDEnum

	// ParseMessage parses the raw message bytes
	ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error)

	// EncodeMessage encodes message to bytes
	EncodeMessage(message interface{}) (common.Bytes, error)

	// HandleMessage processes the received message
	HandleMessage(message types.Message) error
}

//
// Network is a handle to the P2P network
//
type Network interface {

	// Start is called when the network starts
	Start(ctx context.Context) error

	// Wait blocks until all goroutines have stopped
	Wait()

	// Stop is called when the network stops
	Stop()

	// Publish publishes the given message to all the subscribers
	Publish(message types.Message) error

	// Broadcast publishes the given message
	Broadcast(message types.Message, skipEdgeNode bool) chan bool

	// BroadcastToNeighbors broadcasts the given message to the neighboring peers
	BroadcastToNeighbors(message types.Message, maxNumPeersToBroadcast int, skipEdgeNode bool) chan bool

	// Send sends the given message to the peer specified by the peerID
	Send(peerID string, message types.Message) bool

	// Peers return the IDs of all peers
	Peers(skipEdgeNode bool) []string

	// PeerURLs return the URLs of all peers
	PeerURLs(skipEdgeNode bool) []string

	// PeerExists indicates if the given peerID is a neighboring peer
	PeerExists(peerID string) bool

	// RegisterMessageHandler registers message handler
	RegisterMessageHandler(messageHandler MessageHandler)

	// ID returns the ID of the network peer
	ID() string
}
