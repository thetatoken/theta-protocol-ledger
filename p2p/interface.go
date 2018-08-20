package p2p

import (
	"github.com/thetatoken/ukulele/common"
	pr "github.com/thetatoken/ukulele/p2p/peer"
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
	AttachToPeer(peer *pr.Peer)
	DetachFromPeer(peer *pr.Peer)
	GetChannelIDs() []common.ChannelIDEnum
	Receive(peer *pr.Peer, channelID byte, msgBytes common.Bytes)
}

//
// Network is a handle to the P2P network.
//
type Network interface {
	Broadcast(msg Message) error
	Send(ID string, msg Message) error

	AddMessageHandler(handler MessageHandler)

	ID() string
}
