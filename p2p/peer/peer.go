package peer

import (
	"net"
	"time"

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
	DialTimeout time.Duration
}

func createPeer() *Peer {
	return nil
}

func dial(addr *netutil.NetAddress, config PeerConfig) (net.Conn, error) {
	conn, err := addr.DialTimeout(config.DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
