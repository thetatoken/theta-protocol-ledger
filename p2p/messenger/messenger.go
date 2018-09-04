package messenger

import (
	"crypto/ecdsa"
	"strconv"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

//
// Messenger implements the Network interface
//
var _ p2p.Network = (*Messenger)(nil)

type Messenger struct {
	discMgr       *PeerDiscoveryManager
	msgHandlerMap map[common.ChannelIDEnum](p2p.MessageHandler)
	peerTable     pr.PeerTable
	nodeInfo      p2ptypes.NodeInfo // information of our blockchain node

	config MessengerConfig
}

//
// MessengerConfig specifies the configuration for Messenger
//
type MessengerConfig struct {
	addrBookFilePath    string
	routabilityRestrict bool
	skipUPNP            bool
	networkProtocol     string
}

// CreateMessenger creates an instance of Messenger
func CreateMessenger(pubKey ecdsa.PublicKey, seedPeerNetAddresses []string,
	port int, msgrConfig MessengerConfig) (*Messenger, error) {

	messenger := &Messenger{
		msgHandlerMap: make(map[common.ChannelIDEnum](p2p.MessageHandler)),
		peerTable:     pr.CreatePeerTable(),
		nodeInfo:      p2ptypes.CreateNodeInfo(pubKey),
		config:        msgrConfig,
	}

	localNetAddress := "127.0.0.1:" + strconv.Itoa(port)
	discMgrConfig := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(messenger, &(messenger.nodeInfo),
		msgrConfig.addrBookFilePath, msgrConfig.routabilityRestrict,
		seedPeerNetAddresses, msgrConfig.networkProtocol,
		localNetAddress, msgrConfig.skipUPNP, &messenger.peerTable, discMgrConfig)
	if err != nil {
		log.Errorf("[p2p] Failed to create CreatePeerDiscoveryManager")
		return messenger, err
	}

	discMgr.SetMessenger(messenger)
	messenger.SetPeerDiscoveryManager(discMgr)

	return messenger, nil
}

// GetDefaultMessengerConfig returns the default config for messenger
func GetDefaultMessengerConfig() MessengerConfig {
	return MessengerConfig{
		addrBookFilePath:    "./.addrbook/addrbook.json",
		routabilityRestrict: false,
		skipUPNP:            false,
		networkProtocol:     "tcp",
	}
}

// SetPeerDiscoveryManager sets the PeerDiscoveryManager for the Messenger
func (msgr *Messenger) SetPeerDiscoveryManager(discMgr *PeerDiscoveryManager) {
	msgr.discMgr = discMgr
}

// OnStart is called when the Messenger starts
func (msgr *Messenger) OnStart() error {
	err := msgr.discMgr.OnStart()
	return err
}

// OnStop is called when the Messenger stops
func (msgr *Messenger) OnStop() {
	msgr.discMgr.OnStop()
}

// Broadcast broadcasts the given message to all the connected peers
func (msgr *Messenger) Broadcast(message p2ptypes.Message) (successes chan bool) {
	log.Debugf("[p2p] Broadcasting messages...")
	allPeers := msgr.peerTable.GetAllPeers()
	successes = make(chan bool, len(*allPeers))
	for _, peer := range *allPeers {
		log.Debugf("[p2p] Broadcasting \"%v\" to %v", message.Content, peer.ID())
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

// RegisterMessageHandler registers the message handler
func (msgr *Messenger) RegisterMessageHandler(msgHandler p2p.MessageHandler) {
	channelIDs := msgHandler.GetChannelIDs()
	for _, channelID := range channelIDs {
		if msgr.msgHandlerMap[channelID] != nil {
			log.Errorf("[p2p] Message handlered is already added for channelID: %v", channelID)
			return
		}
		msgr.msgHandlerMap[channelID] = msgHandler
	}
}

// ID returns the ID of the current node
func (msgr *Messenger) ID() string {
	return msgr.nodeInfo.Address
}

// AttachMessageHandlersToPeer attaches the registerred message handlers to the given peer
func (msgr *Messenger) AttachMessageHandlersToPeer(peer *pr.Peer) {
	messageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			log.Errorf("[p2p] Failed to setup message parser for channelID %v", channelID)
		}
		message, err := msgHandler.ParseMessage(channelID, rawMessageBytes)
		return message, err
	}
	peer.GetConnection().SetMessageParser(messageParser)

	receiveHandler := func(message p2ptypes.Message) error {
		channelID := message.ChannelID
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			log.Errorf("[p2p] Failed to setup message handler for channelID %v", channelID)
		}
		peerID := peer.ID()
		err := msgHandler.HandleMessage(peerID, message)
		return err
	}
	peer.GetConnection().SetReceiveHandler(receiveHandler)

	// TODO: error handling..
	// errorHandler := func(interface{}) {
	// 	msgr.discMgr.HandlePeerWithErrors(peer)
	// }
	// peer.GetConnection().SetErrorHandler(errorHandler)
}

// SetAddressBookFilePath sets the address book file path
func (msgrConfig *MessengerConfig) SetAddressBookFilePath(filePath string) {
	msgrConfig.addrBookFilePath = filePath
}
