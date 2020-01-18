package messenger

import (
	"context"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/p2p"
	pr "github.com/thetatoken/theta/p2p/peer"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2p"})

//
// Messenger implements the Network interface
//
var _ p2p.Network = (*Messenger)(nil)

type Messenger struct {
	discMgr       *PeerDiscoveryManager
	msgHandlerMap map[common.ChannelIDEnum](p2p.MessageHandler)

	peerTable pr.PeerTable
	nodeInfo  p2ptypes.NodeInfo // information of our blockchain node

	config MessengerConfig

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
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
func CreateMessenger(privKey *crypto.PrivateKey, seedPeerNetAddresses []string,
	port int, msgrConfig MessengerConfig) (*Messenger, error) {

	messenger := &Messenger{
		msgHandlerMap: make(map[common.ChannelIDEnum](p2p.MessageHandler)),
		peerTable:     pr.CreatePeerTable(),
		nodeInfo:      p2ptypes.CreateLocalNodeInfo(privKey, uint16(port)),
		config:        msgrConfig,
		wg:            &sync.WaitGroup{},
	}

	localNetAddress := "0.0.0.0:" + strconv.Itoa(port)
	discMgrConfig := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(messenger, &(messenger.nodeInfo),
		msgrConfig.addrBookFilePath, msgrConfig.routabilityRestrict,
		seedPeerNetAddresses, msgrConfig.networkProtocol,
		localNetAddress, msgrConfig.skipUPNP, &messenger.peerTable, discMgrConfig)
	if err != nil {
		logger.Errorf("Failed to create CreatePeerDiscoveryManager")
		return messenger, err
	}

	discMgr.SetMessenger(messenger)
	messenger.SetPeerDiscoveryManager(discMgr)
	messenger.RegisterMessageHandler(&discMgr.peerDiscMsgHandler)

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

// Start is called when the Messenger starts
func (msgr *Messenger) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	msgr.ctx = c
	msgr.cancel = cancel

	err := msgr.discMgr.Start(c)
	return err
}

// Stop is called when the Messenger stops
func (msgr *Messenger) Stop() {
	msgr.cancel()
}

// Wait suspends the caller goroutine
func (msgr *Messenger) Wait() {
	msgr.discMgr.Wait()
	msgr.wg.Wait()
}

// Broadcast broadcasts the given message to all the connected peers
func (msgr *Messenger) Broadcast(message p2ptypes.Message) (successes chan bool) {
	logger.Debugf("Broadcasting messages...")
	allPeers := msgr.peerTable.GetAllPeers()
	successes = make(chan bool, len(*allPeers))
	for _, peer := range *allPeers {
		logger.Debugf("Broadcasting \"%v\" to %v", message.Content, peer.ID())
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

// Peers returns the IDs of all peers
func (msgr *Messenger) Peers() []string {
	allPeers := msgr.peerTable.GetAllPeers()
	peerIDs := []string{}
	for _, peer := range *allPeers {
		peerIDs = append(peerIDs, peer.ID())
	}
	return peerIDs
}

// RegisterMessageHandler registers the message handler
func (msgr *Messenger) RegisterMessageHandler(msgHandler p2p.MessageHandler) {
	channelIDs := msgHandler.GetChannelIDs()
	for _, channelID := range channelIDs {
		if msgr.msgHandlerMap[channelID] != nil {
			logger.Errorf("Message handler is already added for channelID: %v", channelID)
			return
		}
		msgr.msgHandlerMap[channelID] = msgHandler
	}
}

// ID returns the ID of the current node
func (msgr *Messenger) ID() string {
	return msgr.nodeInfo.PubKey.Address().Hex()
}

// AttachMessageHandlersToPeer attaches the registered message handlers to the given peer
func (msgr *Messenger) AttachMessageHandlersToPeer(peer *pr.Peer) {
	messageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		peerID := peer.ID()
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			logger.Errorf("Failed to setup message parser for channelID %v", channelID)
		}
		message, err := msgHandler.ParseMessage(peerID, channelID, rawMessageBytes)
		return message, err
	}
	peer.GetConnection().SetMessageParser(messageParser)

	messageEncoder := func(channelID common.ChannelIDEnum, message interface{}) (common.Bytes, error) {
		msgHandler := msgr.msgHandlerMap[channelID]
		return msgHandler.EncodeMessage(message)
	}
	peer.GetConnection().SetMessageEncoder(messageEncoder)

	receiveHandler := func(message p2ptypes.Message) error {
		channelID := message.ChannelID
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			logger.Errorf("Failed to setup message handler for peer %v on channelID %v", message.PeerID, channelID)
		}
		err := msgHandler.HandleMessage(message)
		return err
	}
	peer.GetConnection().SetReceiveHandler(receiveHandler)

	errorHandler := func(interface{}) {
		msgr.discMgr.HandlePeerWithErrors(peer)
	}
	peer.GetConnection().SetErrorHandler(errorHandler)
}

// SetAddressBookFilePath sets the address book file path
func (msgrConfig *MessengerConfig) SetAddressBookFilePath(filePath string) {
	msgrConfig.addrBookFilePath = filePath
}
