package p2p

import (
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	disc "github.com/thetatoken/ukulele/p2p/discovery"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

//
// Messenger implements the Network interface
//
type Messenger struct {
	discMgr       *disc.PeerDiscoveryManager
	msgHandlerMap map[common.ChannelIDEnum](*MessageHandler)
	peerTable     pr.PeerTable
	nodeInfo      p2ptypes.NodeInfo // information of our blockchain node
}

//
// MessengerConfig specifies the configuration for Messenger
//
type MessengerConfig struct {
}

// CreateMessenger creates an instance of Messenger
func CreateMessenger(addrBookFilePath string, routabilityRestrict bool, selfNetAddressStr string,
	seedPeerNetAddressStrs []string, networkProtocol string, localNetworkAddr string,
	skipUPNP bool, nodeInfo p2ptypes.NodeInfo) (*Messenger, error) {

	messenger := &Messenger{
		peerTable: pr.PeerTable{},
		nodeInfo:  nodeInfo,
	}

	var err error
	discMgrConfig := disc.CreateDefaultPeerDiscoveryManagerConfig()
	messenger.discMgr, err = disc.CreatePeerDiscoveryManager(&messenger.nodeInfo, addrBookFilePath,
		routabilityRestrict, selfNetAddressStr, seedPeerNetAddressStrs, networkProtocol,
		localNetworkAddr, skipUPNP, &messenger.peerTable, discMgrConfig)
	if err != nil {
		log.Errorf("[p2p] Failed to create CreatePeerDiscoveryManager")
		return messenger, err
	}

	return messenger, nil
}

// OnStart is called when the Messenger starts
func (msgr *Messenger) OnStart() {
	msgr.discMgr.OnStart()
}

// OnStop is called when the Messenger stops
func (msgr *Messenger) OnStop() {
	msgr.discMgr.OnStop()
}

// Broadcast broadcasts the given message to all the connected peers
func (msgr *Messenger) Broadcast(message p2ptypes.Message) (successes chan bool) {
	allPeers := msgr.peerTable.GetAllPeers()
	successes = make(chan bool, len(*allPeers))
	for _, peer := range *allPeers {
		go func(peer *pr.Peer) {
			success := msgr.Send(peer.ID(), message)
			successes <- success
		}(peer)
	}
	return successes
}

// Send sends the given message to the specified peer
func (msgr *Messenger) Send(peerID string, message p2ptypes.Message) bool {
	peer := msgr.peerTable.GetPeer(peerID)
	if peer == nil {
		return false
	}

	success := peer.Send(message.ChannelID, message.Content)

	return success
}

// AddMessageHandler adds the message handler
func (msgr *Messenger) AddMessageHandler(msgHandler *MessageHandler) bool {
	channelIDs := (*msgHandler).GetChannelIDs()
	for _, channelID := range channelIDs {
		if msgr.msgHandlerMap[channelID] != nil {
			log.Errorf("[p2p] Message handlered is already added for channelID: %v", channelID)
			return false
		}
		msgr.msgHandlerMap[channelID] = msgHandler
	}
	return true
}

// ID returns the ID of the current node
func (msgr *Messenger) ID() string {
	return msgr.nodeInfo.GetAddress()
}

// AttachMessageHandlerToPeer attaches the approporiate message handler to the given peer
func (msgr *Messenger) AttachMessageHandlerToPeer(peer *pr.Peer) {
	receiveHandler := func(channelID common.ChannelIDEnum, msgBytes common.Bytes) {
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			log.Errorf("[p2p] Failed to setup message handler for ")
		}
		peerID := peer.ID()
		message := p2ptypes.Message{
			ChannelID: channelID,
			Content:   msgBytes,
		}
		(*msgHandler).HandleMessage(peerID, message)
	}
	peer.GetConnection().SetReceiveHandler(receiveHandler)

	errorHandler := func(interface{}) {
		msgr.discMgr.HandlePeerWithErrors(peer)
	}
	peer.GetConnection().SetErrorHandler(errorHandler)
}
