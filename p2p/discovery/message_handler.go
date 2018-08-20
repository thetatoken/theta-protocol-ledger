package discovery

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p/netutil"
	pr "github.com/thetatoken/ukulele/p2p/peer"
)

// PeerDiscoveryMessageType defines the types of peer discovery message
type PeerDiscoveryMessageType byte

const (
	peerAddressesRequestType PeerDiscoveryMessageType = 0x01
	peerAddressesReplyType   PeerDiscoveryMessageType = 0x02
)

const (
	connectivityPulsePeriod     = 30 * time.Second
	minNumOutboundPeers         = 10
	maxPeerDiscoveryMessageSize = 1048576 // 1MB
)

// PeerDiscoveryMessage defines the structure of the peer discovery message
type PeerDiscoveryMessage struct {
	Type      PeerDiscoveryMessageType
	Addresses []*netutil.NetAddress
}

//
// PeerDiscoveryMessageHandler implements the MessageHandler interface
//
type PeerDiscoveryMessageHandler struct {
	addrBook *AddrBook
}

// OnStart is called when the message handler starts
func (pdmh *PeerDiscoveryMessageHandler) OnStart() {
	pdmh.addrBook.OnStart()
	go pdmh.maintainSufficientConnectivityRoutine()
}

// OnStop is called when the message handler stops
func (pdmh *PeerDiscoveryMessageHandler) OnStop() {
	pdmh.addrBook.OnStop()
}

// AttachToPeer attaches the message handler to the given peer
func (pdmh *PeerDiscoveryMessageHandler) AttachToPeer(peer *pr.Peer) {
	if peer.IsOutbound() {
		if pdmh.addrBook.NeedMoreAddrs() {
			pdmh.requestAddresses(peer)
		}
	} else {
		addr := netutil.NewNetAddress(peer.GetConnection().GetNetconn().RemoteAddr())
		pdmh.addrBook.addAddress(addr, addr)
		log.Infof("[p2p] Peer discovery - added inbound peer %v to the address book", addr)
	}
}

// DetachFromPeer detaches the message handler from the given peer
func (pdmh *PeerDiscoveryMessageHandler) DetachFromPeer(peer *pr.Peer) {
	return
}

// GetChannelIDs returns the list of channels the message handler needs to handle
func (pdmh *PeerDiscoveryMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDPeerDiscovery,
	}
}

// Receive is called when a message is received on the specified channel
func (pdmh *PeerDiscoveryMessageHandler) Receive(peer *pr.Peer, channelID byte, msgBytes common.Bytes) {
	message, err := decodePeerDiscoveryMessage(msgBytes)
	if err != nil {
		log.Errorf("[p2p] Error decoding PeerDiscoveryMessage: %v", err)
		return
	}

	switch message.Type {
	case peerAddressesRequestType:
		pdmh.handlePeerAddressRequest(peer, message)
	case peerAddressesReplyType:
		pdmh.handlePeerAddressReply(peer, message)
	default:
		log.Errorf("[p2p] Invalid PeerDiscoveryMessageType")
	}
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressRequest(peer *pr.Peer, message PeerDiscoveryMessage) {
	addresses := pdmh.addrBook.GetSelection()
	pdmh.sendAddresses(peer, addresses)
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressReply(peer *pr.Peer, message PeerDiscoveryMessage) {
	for _, addr := range message.Addresses {
		if addr.Valid() {
			srcAddr := netutil.NewNetAddress(peer.GetConnection().GetNetconn().RemoteAddr())
			pdmh.addrBook.AddAddress(addr, srcAddr)
		}
	}
}

func (pdmh *PeerDiscoveryMessageHandler) maintainSufficientConnectivityRoutine() {
	pulse := time.NewTicker(connectivityPulsePeriod)
	for {
		select {
		case <-pulse.C:
			pdmh.maintainSufficientConnectivity()
		}
	}
}

// maintainConnecmaintainSufficientConnectivitytivity tries to maintain sufficient number
// of connections by dialing peers when the number of connected peers are lower than the
// required threshold
func (pdmh *PeerDiscoveryMessageHandler) maintainSufficientConnectivity() {
	// TODO: implementation
}

func (pdmh *PeerDiscoveryMessageHandler) requestAddresses(peer *pr.Peer) {
	message := PeerDiscoveryMessage{
		Type: peerAddressesRequestType,
	}
	peer.Send(byte(common.ChannelIDPeerDiscovery), message)
}

func (pdmh *PeerDiscoveryMessageHandler) sendAddresses(peer *pr.Peer, addresses []*netutil.NetAddress) {
	message := PeerDiscoveryMessage{
		Type:      peerAddressesReplyType,
		Addresses: addresses,
	}
	peer.Send(byte(common.ChannelIDPeerDiscovery), message)
}

func decodePeerDiscoveryMessage(msgBytes common.Bytes) (message PeerDiscoveryMessage, err error) {
	// TODO: implementation
	return PeerDiscoveryMessage{}, nil
}
