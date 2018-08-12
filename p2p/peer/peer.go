package peer

import (
	"github.com/thetatoken/ukulele/p2p/connection"
	"github.com/thetatoken/ukulele/p2p/netutil"
)

//
// Peer models a peer node in a network
//
type Peer struct {
	connection *connection.Connection
	address    *netutil.NetAddress

	config PeerConfig
}

//
// PeerConfig specifies the configuration of a peer node
//
type PeerConfig struct {
}

func createPeer() *Peer {
	return nil
}
