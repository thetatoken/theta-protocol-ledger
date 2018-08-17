package messenger

import (
	pr "github.com/thetatoken/ukulele/p2p/peer"
)

//
// MessageHandler interface
//
type MessageHandler interface {
	AttachToPeer(peer *pr.Peer)
	DetachFromPeer(peer *pr.Peer)
	Receive(channelID byte, msgBytes []byte)
}
