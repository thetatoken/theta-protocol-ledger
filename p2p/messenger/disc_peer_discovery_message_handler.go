package messenger

import (
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p/netutil"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	"github.com/thetatoken/ukulele/p2p/types"
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
	Type         PeerDiscoveryMessageType
	SourcePeerID string
	Addresses    []*netutil.NetAddress
}

//
// PeerDiscoveryMessageHandler implements the MessageHandler interface
//
type PeerDiscoveryMessageHandler struct {
	discMgr *PeerDiscoveryManager
}

// createPeerDiscoveryMessageHandler creates an instance of PeerDiscoveryMessageHandler
func createPeerDiscoveryMessageHandler(discMgr *PeerDiscoveryManager) (PeerDiscoveryMessageHandler, error) {
	pmdh := PeerDiscoveryMessageHandler{
		discMgr: discMgr,
	}
	return pmdh, nil
}

// OnStart is called when the message handler starts
func (pdmh *PeerDiscoveryMessageHandler) OnStart() error {
	go pdmh.maintainSufficientConnectivityRoutine()
	return nil
}

// OnStop is called when the message handler stops
func (pdmh *PeerDiscoveryMessageHandler) OnStop() {
}

// GetChannelIDs implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDPeerDiscovery,
	}
}

// ParseMessage implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) ParseMessage(
	channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error) {
	discMsg, err := decodePeerDiscoveryMessage(rawMessageBytes)
	message := types.Message{
		ChannelID: channelID,
		Content:   discMsg,
	}
	if err != nil {
		log.Errorf("[p2p] Error decoding PeerDiscoveryMessage: %v", err)
		return message, err
	}

	return message, nil
}

// HandleMessage implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) HandleMessage(peerID string, msg types.Message) error {
	if msg.ChannelID != common.ChannelIDPeerDiscovery {
		errMsg := fmt.Sprintf("[p2p] Invalid channelID for the PeerDiscoveryMessageHandler: %v", msg.ChannelID)
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	peer := pdmh.discMgr.peerTable.GetPeer(peerID)
	if peer == nil {
		errMsg := fmt.Sprintf("[p2p] Cannot find peer %v in the peer table", peerID)
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	discMsg := (msg.Content).(PeerDiscoveryMessage)
	switch discMsg.Type {
	case peerAddressesRequestType:
		pdmh.handlePeerAddressRequest(peer, discMsg)
	case peerAddressesReplyType:
		pdmh.handlePeerAddressReply(peer, discMsg)
	default:
		errMsg := "[p2p] Invalid PeerDiscoveryMessageType"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressRequest(peer *pr.Peer, message PeerDiscoveryMessage) {
	addresses := pdmh.discMgr.addrBook.GetSelection()
	pdmh.sendAddresses(peer, addresses)
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressReply(peer *pr.Peer, message PeerDiscoveryMessage) {
	for _, addr := range message.Addresses {
		if addr.Valid() {
			srcAddr := netutil.NewNetAddress(peer.GetConnection().GetNetconn().RemoteAddr())
			pdmh.discMgr.addrBook.AddAddress(addr, srcAddr)
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
	peer.Send(common.ChannelIDPeerDiscovery, message)
}

func (pdmh *PeerDiscoveryMessageHandler) sendAddresses(peer *pr.Peer, addresses []*netutil.NetAddress) {
	message := PeerDiscoveryMessage{
		Type:      peerAddressesReplyType,
		Addresses: addresses,
	}
	peer.Send(common.ChannelIDPeerDiscovery, message)
}

func decodePeerDiscoveryMessage(msgBytes common.Bytes) (message PeerDiscoveryMessage, err error) {
	// TODO: implementation
	return PeerDiscoveryMessage{}, nil
}
