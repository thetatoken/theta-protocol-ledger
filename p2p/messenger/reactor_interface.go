package messenger

import (
	pr "github.com/thetatoken/ukulele/p2p/peer"
)

//
// Reactor interface
//
type Reactor interface {
	OnStart() error
	OnStop()
	AddPeer(peer *pr.Peer)
	RemovePeer(peer *pr.Peer)
	Receive(channelID byte, peer *pr.Peer, msgBytes []byte)
}
