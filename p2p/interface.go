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

	// ParseMessage parses the raw message bytes
	ParseMessage(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error)

	// HandleMessage handles the message received from the peer with peerID
	HandleMessage(peerID string, message types.Message) error
}

//
// Network is a handle to the P2P network
//
type Network interface {

	// OnStart is called when the network starts
	OnStart() error

	// OnStop is called when the network stops
	OnStop()

	// Broadcast broadcasts the given message to all the neighboring peers
	Broadcast(message types.Message) chan bool

	// Send sends the given message to the peer specified by the peerID
	Send(peerID string, message types.Message) bool

	// RegisterMessageHandler registers message handler for the specified channel
	RegisterMessageHandler(messageHandler MessageHandler)

	// ID returns the ID of the network peer
	ID() string
}
