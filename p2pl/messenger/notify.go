package messenger

import (
	"github.com/libp2p/go-libp2p-core/network"
	ma "github.com/multiformats/go-multiaddr"
)

// var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2pl"})

var _ network.Notifiee = (*PeerNotif)(nil)

type PeerNotif Messenger

func (p *PeerNotif) OpenedStream(n network.Network, s network.Stream) {
	// peerID := s.Conn().RemotePeer()
	// logger.Infof("OpenedStream %v", peerID)
}

func (p *PeerNotif) ClosedStream(n network.Network, s network.Stream) {
	// peerID := s.Conn().RemotePeer()
	// logger.Infof("ClosedStream %v", peerID)
}

func (p *PeerNotif) Connected(n network.Network, c network.Conn) {
	go func() {
		select {
		case p.newPeers <- c.RemotePeer():
		// case <-p.ctx.Done():
		}
	}()
}

func (p *PeerNotif) Disconnected(n network.Network, c network.Conn) {
	go func() {
		select {
		case p.peerDead <- c.RemotePeer():
		// case <-p.ctx.Done():
		}
	}()
}

func (p *PeerNotif) Listen(n network.Network, _ ma.Multiaddr) {
}

func (p *PeerNotif) ListenClose(n network.Network, _ ma.Multiaddr) {
}
