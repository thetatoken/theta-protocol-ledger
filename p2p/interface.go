package p2p

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p/types"
)

//
// MessageHandler interface
//
type MessageHandler interface {

	// GetChannelIDs returns the list channelIDs that the message handler needs to handle
	GetChannelIDs() []common.ChannelIDEnum

	// HandleMessage handles the message received from the peer with peerID
	HandleMessage(peerID string, message types.Message)
}

//
// Network is a handle to the P2P network
//
type Network interface {

	// Broadcast broadcasts the given message to all the neighboring peers
	Broadcast(message types.Message) error

	// Send sends the given message to the peer specified by the peerID
	Send(peerID string, message types.Message) error

	// AddMessageHandler adds message handler for the specified channel
	AddMessageHandler(messageHandler MessageHandler)

	// ID returns the ID of the network peer
	ID() string
}
