package p2p

import (
	"github.com/thetatoken/ukulele/common"
)

//
// Message models the message sent/received through the P2P network
//
type Message struct {
	ChannelID byte
	Content   interface{}
}

//
// MessageHandler interface
//
type MessageHandler interface {

	// GetChannelID returns the ID of the channel that the message handler needs to handle
	GetChannelID() common.ChannelIDEnum

	// HandleMessage handled the message received from the corresponding channel
	HandleMessage(peerID string, rawMsgBytes common.Bytes)
}

//
// Network is a handle to the P2P network
//
type Network interface {

	// Broadcast broadcasts the given message to all the neighboring peers
	Broadcast(msg Message) error

	// Send sends the given message to the peer specified by the peerID
	Send(peerID string, msg Message) error

	// AddMessageHandler adds message handler for the specified channel
	AddMessageHandler(msgHandler MessageHandler)

	// ID returns the ID of the network peer
	ID() string
}
