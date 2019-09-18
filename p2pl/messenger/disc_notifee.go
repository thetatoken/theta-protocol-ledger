package messenger

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	ps "github.com/libp2p/go-libp2p-core/peerstore"
)

type discoveryNotifee struct {
	ctx  context.Context
	host host.Host
}

func (d *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL) // may not need
	err := d.host.Connect(d.ctx, pi)
	if err != nil {
		logger.Errorf("Failed to connect to peer: %v", err)
		return
	}

	logger.Infof("Connected to: %v", pi)
}
