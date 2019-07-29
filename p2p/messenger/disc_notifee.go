package messenger

import (
	"bufio"
	"context"
	"github.com/thetatoken/theta/common"

	// pr "github.com/thetatoken/theta/p2p/peer"
	// cn "github.com/thetatoken/theta/p2p/connection"

	"github.com/spf13/viper"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	ps "github.com/libp2p/go-libp2p-core/peerstore"
)

type discoveryNotifee struct {
	ctx  context.Context
	host host.Host
}

//interface to be called when new  peer is found
func (d *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	d.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, ps.PermanentAddrTTL)
	err := d.host.Connect(d.ctx, pi)
	if err != nil {
		logger.Errorf("Failed to connect to peer: %v", err)
		return
	}
	

	localChainID := viper.GetString(common.CfgGenesisChainID)
	stream, err := d.host.NewStream(d.ctx, pi.ID, protocol.ID(localChainID))
	if err != nil {
		logger.Errorf("Failed to create stream for peer: %v", err)
		return
	}

	w := bufio.NewWriter(stream)
	err = w.Flush()
	if err != nil {
		logger.Errorf("Error flushing buffer %v", err)
		return
	}

	logger.Infof(">>>>>>>>>> Connected to: %v", pi)

	// peerConfig := pr.GetDefaultPeerConfig()
	// connConfig := cn.GetDefaultConnectionConfig()
	// peer := pr.CreatePeer(stream, peerConfig, connConfig)
	// if peer == nil {
	// 	logger.Errorf("Failed to create peer")
	// 	return
	// }

	// if !peer.Start(d.ctx) {
	// 	errMsg := "Failed to start peer"
	// 	logger.Errorf(errMsg)
	// 	return
	// }

}

// //Initialize the MDNS service
// func initMDNS(ctx context.Context, peerhost host.Host, rendezvous string) chan peer.AddrInfo {
// 	ser, err := discovery.NewMdnsService(ctx, peerhost, time.Second * 10, rendezvous)
// 	if err != nil {
// 		panic(err)
// 	}

// 	//register with service so that we get notified about peer discovery
// 	n := &discoveryNotifee{}
// 	n.PeerChan = make(chan peer.AddrInfo)

// 	ser.RegisterNotifee(n)
// 	return n.PeerChan
// }
