package messenger

import (
	"context"
	"strconv"
	"sync"

	"github.com/spf13/viper"

	//nat "github.com/fd/go-nat"
	//nat "github.com/libp2p/go-nat"
	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
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
	natMgr        *NATManager
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

	var err error
	eport := port
	natMgr := CreateNATManager(port)
	if viper.GetBool(common.CfgP2PNatMapping) {
		natMgr.DiscoverGateway()
		if eport, err = natMgr.NatMapping(port); err != nil {
			logger.Warnf("Failed to perform NAT port mapping: %v", err)
		}
	}

	messenger := &Messenger{
		msgHandlerMap: make(map[common.ChannelIDEnum](p2p.MessageHandler)),
		peerTable:     pr.CreatePeerTable(),
		nodeInfo:      p2ptypes.CreateLocalNodeInfo(privKey, uint16(eport)),
		config:        msgrConfig,
		wg:            &sync.WaitGroup{},
	}

	localNetAddress := "0.0.0.0:" + strconv.Itoa(port)
	discMgrConfig := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(messenger, &(messenger.nodeInfo),
		msgrConfig.addrBookFilePath, msgrConfig.routabilityRestrict,
		seedPeerNetAddresses, msgrConfig.networkProtocol,
		localNetAddress, eport, msgrConfig.skipUPNP, &messenger.peerTable, discMgrConfig)
	if err != nil {
		logger.Errorf("Failed to create CreatePeerDiscoveryManager")
		return messenger, err
	}

	discMgr.SetMessenger(messenger)
	messenger.SetPeerDiscoveryManager(discMgr)
	messenger.RegisterMessageHandler(&discMgr.peerDiscMsgHandler)

	// should call SetNATManager/RegisterMessageHandler regardless of the CfgP2PNatMapping config since the node needs to handle the eport update messages
	natMgr.SetMessenger(messenger)
	messenger.SetNATManager(natMgr)
	messenger.RegisterMessageHandler(natMgr)

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

// SetPeerDiscoveryManager sets the PeerDiscoveryManager for the Messenger
func (msgr *Messenger) SetNATManager(natMgr *NATManager) {
	msgr.natMgr = natMgr
}

// Start is called when the Messenger starts
func (msgr *Messenger) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	msgr.ctx = c
	msgr.cancel = cancel

	err := msgr.discMgr.Start(c)
	if err != nil {
		return err
	}

	if msgr.natMgr != nil {
		err = msgr.natMgr.Start(c)
	}

	return err
}

// Stop is called when the Messenger stops
func (msgr *Messenger) Stop() {
	msgr.cancel()
}

// Wait suspends the caller goroutine
func (msgr *Messenger) Wait() {
	msgr.discMgr.Wait()
	if msgr.natMgr != nil {
		msgr.natMgr.Wait()
	}
	msgr.wg.Wait()
}

// Broadcast broadcasts the given message to all the connected peers
func (msgr *Messenger) Broadcast(message p2ptypes.Message, skipEdgeNode bool) (successes chan bool) {
	allPeers := msgr.peerTable.GetAllPeers(skipEdgeNode)
	successes = make(chan bool, len(*allPeers))
	logger.Debugf("Broadcasting message to %v peers on channel %v, skipEdgeNode: %v", len(*allPeers), message.ChannelID, skipEdgeNode)

	for _, peer := range *allPeers {
		//logger.Debugf("Broadcasting message with hash %v to %v, channelID: %v", hex.EncodeToString(crypto.Keccak256([]byte(fmt.Sprintf("%v", message.Content)))), peer.ID(), message.ChannelID)
		go func(peer *pr.Peer) {
			success := msgr.Send(peer.ID(), message)
			successes <- success
		}(peer)
	}
	return successes
}

// BroadcastToNeighbors broadcasts the given message to neighbors
func (msgr *Messenger) BroadcastToNeighbors(message p2ptypes.Message, maxNumPeersToBroadcast int, skipEdgeNode bool) (successes chan bool) {
	sampledPIDs := msgr.samplePeers(maxNumPeersToBroadcast, skipEdgeNode)
	logger.Debugf("Broadcasting message to %v neighbors on channel %v, skipEdgeNode: %v", len(sampledPIDs), message.ChannelID, skipEdgeNode)

	for _, pid := range sampledPIDs {
		//logger.Debugf("Broadcasting message with hash %v to neighbor %v, channelID: %v", hex.EncodeToString(crypto.Keccak256([]byte(fmt.Sprintf("%v", message.Content)))), pid, message.ChannelID)
		go func(pid string) {
			msgr.Send(pid, message)
		}(pid)
	}
	return make(chan bool)
}

// samplePeers randomly sample a subset of peers
func (msgr *Messenger) samplePeers(maxNumSampledPeers int, skipEdgeNode bool) []string {
	// Prioritize seed peers
	sampledPIDs, idx := []string{}, 0
	for seedPID := range msgr.discMgr.seedPeers {
		// Note: the order of map loop-through is undeterminstic, which effectively shuffles the seed peers
		sampledPIDs = append(sampledPIDs, seedPID)
		idx++
		if idx >= maxNumSampledPeers {
			return sampledPIDs
		}
	}

	// Randomly sample the remaining peers
	neighbors := *msgr.peerTable.GetAllPeers(skipEdgeNode)
	neighborPIDs := []string{}
	for _, peer := range neighbors {
		pid := peer.ID()
		if pid == msgr.ID() || msgr.discMgr.isSeedPeer(pid) {
			continue
		}
		neighborPIDs = append(neighborPIDs, pid)
	}

	numPeersToSample := maxNumSampledPeers - len(msgr.discMgr.seedPeers) // numPeersToSample is guaranteed > 0
	sampledNeighbors := util.Sample(neighborPIDs, numPeersToSample)
	if numPeersToSample >= len(sampledNeighbors) {
		numPeersToSample = len(sampledNeighbors)
	}

	for i := 0; i < numPeersToSample; i++ {
		sampledPIDs = append(sampledPIDs, sampledNeighbors[i])
	}

	return sampledPIDs
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
func (msgr *Messenger) Peers(skipEdgeNode bool) []string {
	allPeers := msgr.peerTable.GetAllPeers(skipEdgeNode)
	peerIDs := []string{}
	for _, peer := range *allPeers {
		peerIDs = append(peerIDs, peer.ID())
	}
	return peerIDs
}

// PeerURLs returns the URLs of all peers
func (msgr *Messenger) PeerURLs(skipEdgeNode bool) []string {
	allPeers := msgr.peerTable.GetAllPeers(skipEdgeNode)
	peerURLs := []string{}
	for _, peer := range *allPeers {
		peerURLs = append(peerURLs, peer.NetAddress().String())
	}
	return peerURLs
}

// PeerExists indicates if the given peerID is a neighboring peer
func (msgr *Messenger) PeerExists(peerID string) bool {
	return msgr.peerTable.PeerExists(peerID)
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
