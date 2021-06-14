package messenger

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/spf13/viper"

	"github.com/thetatoken/theta/rlp"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p/netutil"
	pr "github.com/thetatoken/theta/p2p/peer"
	"github.com/thetatoken/theta/p2p/types"
)

// PeerDiscoveryMessageType defines the types of peer discovery message
type PeerDiscoveryMessageType byte

const (
	peerAddressesRequestType PeerDiscoveryMessageType = 0x01
	peerAddressesReplyType   PeerDiscoveryMessageType = 0x02
)

const (
	defaultPeerDiscoveryPulseInterval = 30 * time.Second
	minNumOutboundPeers               = 10
	maxPeerDiscoveryMessageSize       = 1048576 // 1MB
	requestPeersAddressesPercent      = 25      // 25%
	peersAddressesSubSamplingPercent  = 50      // 50%
	discoverInterval                  = 3000    // 3 sec
)

// PeerDiscoveryMessage defines the structure of the peer discovery message
type PeerDiscoveryMessage struct {
	Type         PeerDiscoveryMessageType
	SourcePeerID string
	Addresses    []pr.PeerIDAddress
}

//
// PeerDiscoveryMessageHandler implements the MessageHandler interface
//
type PeerDiscoveryMessageHandler struct {
	discMgr                    *PeerDiscoveryManager
	selfNetAddress             netutil.NetAddress
	peerDiscoveryPulse         *time.Ticker
	peerDiscoveryPulseInterval time.Duration
	discoveryCallback          InboundCallback

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// createPeerDiscoveryMessageHandler creates an instance of PeerDiscoveryMessageHandler
func createPeerDiscoveryMessageHandler(discMgr *PeerDiscoveryManager, selfNetAddressStr string) (PeerDiscoveryMessageHandler, error) {
	pdmh := PeerDiscoveryMessageHandler{
		discMgr:                    discMgr,
		peerDiscoveryPulseInterval: defaultPeerDiscoveryPulseInterval,
		wg:                         &sync.WaitGroup{},
	}
	selfNetAddress, err := netutil.NewNetAddressString(selfNetAddressStr)
	if err != nil {
		logger.Errorf("Failed to parse the self network address: %v", selfNetAddressStr)
		return pdmh, err
	}
	pdmh.selfNetAddress = *selfNetAddress
	return pdmh, nil
}

// Start is called when the message handler starts
func (pdmh *PeerDiscoveryMessageHandler) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	pdmh.ctx = c
	pdmh.cancel = cancel

	pdmh.wg.Add(1)
	go pdmh.maintainSufficientConnectivityRoutine()

	return nil
}

// Stop is called when the message handler stops
func (pdmh *PeerDiscoveryMessageHandler) Stop() {
	if pdmh.peerDiscoveryPulse != nil {
		pdmh.peerDiscoveryPulse.Stop()
	}
	pdmh.cancel()
}

// Wait suspends the caller goroutine
func (pdmh *PeerDiscoveryMessageHandler) Wait() {
	pdmh.wg.Wait()
}

// GetChannelIDs implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDPeerDiscovery,
	}
}

// EncodeMessage implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

// ParseMessage implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) ParseMessage(peerID string,
	channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error) {
	discMsg, err := decodePeerDiscoveryMessage(rawMessageBytes)
	message := types.Message{
		PeerID:    peerID,
		ChannelID: channelID,
		Content:   discMsg,
	}
	if err != nil {
		logger.Errorf("Error decoding PeerDiscoveryMessage: %v", err)
		return message, err
	}

	return message, nil
}

// HandleMessage implements the p2p.MessageHandler interface
func (pdmh *PeerDiscoveryMessageHandler) HandleMessage(msg types.Message) error {
	if msg.ChannelID != common.ChannelIDPeerDiscovery {
		errMsg := fmt.Sprintf("Invalid channelID for the PeerDiscoveryMessageHandler: %v", msg.ChannelID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	peerID := msg.PeerID
	peer := pdmh.discMgr.peerTable.GetPeer(peerID)
	if peer == nil {
		errMsg := fmt.Sprintf("Cannot find peer %v in the peer table", peerID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	discMsg := (msg.Content).(PeerDiscoveryMessage)
	switch discMsg.Type {
	case peerAddressesRequestType:
		pdmh.handlePeerAddressRequest(peer, discMsg)
	case peerAddressesReplyType:
		pdmh.handlePeerAddressReply(peer, discMsg)
	default:
		errMsg := "Invalid PeerDiscoveryMessageType"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressRequest(peer *pr.Peer, message PeerDiscoveryMessage) {
	skipEdgeNode := (peer.NodeType() == common.NodeTypeBlockchainNode)
	peerIDAddrs := pdmh.discMgr.peerTable.GetSelection(skipEdgeNode)
	pdmh.sendAddresses(peer, peerIDAddrs)
}

func (pdmh *PeerDiscoveryMessageHandler) handlePeerAddressReply(peer *pr.Peer, message PeerDiscoveryMessage) {
	logger.Infof("Received peer discovery reply from %v with %v peer addresses", peer.ID(), len(message.Addresses))
	validAddressMap := make(map[*netutil.NetAddress]bool)
	for _, idAddr := range message.Addresses {
		isNotASeedPeer := !pdmh.discMgr.seedPeerConnector.isASeedPeer(idAddr.Addr)
		if seedPeerOnlyOutbound() && isNotASeedPeer {
			// Sometimes we want to run some nodes behind firewalls. We only allow these nodes to proactively
			// connect to the seed peers (i.e. only the seed peers can be outbound peers). Such nodes can
			// still accept inbound connections from non-seed peers based on the firewall rules.
			continue
		}

		logger.Debugf("Discovered peerID: %v, peerAddress: %v, isValid: %v", idAddr.ID, idAddr.Addr, idAddr.Addr.Valid())

		if idAddr.Addr.Valid() && pdmh.discMgr.messenger.ID() != idAddr.ID && !pdmh.discMgr.peerTable.PeerExists(idAddr.ID) {
			validAddressMap[idAddr.Addr] = true
		}
	}

	logger.Infof("%v out of %v peer addresses are valid", len(validAddressMap), len(message.Addresses))

	if len(validAddressMap) > 0 {
		var validAddresses []*netutil.NetAddress
		for addr := range validAddressMap {
			validAddresses = append(validAddresses, addr)
		}
		pdmh.connectToOutboundPeers(validAddresses)
	}
}

// SetDiscoveryCallback sets the discovery callback function
func (pdmh *PeerDiscoveryMessageHandler) SetDiscoveryCallback(disccb InboundCallback) {
	pdmh.discoveryCallback = disccb
}

func (pdmh *PeerDiscoveryMessageHandler) connectToOutboundPeers(addresses []*netutil.NetAddress) {
	selfNodeType := viper.GetInt(common.CfgNodeType)
	skipEdgeNode := (selfNodeType == int(common.NodeTypeBlockchainNode)) // a blockchain node only asks other blockchain nodes for peers
	numPeers := int(pdmh.discMgr.peerTable.GetTotalNumPeers(skipEdgeNode))
	sufficientNumPeers := int(GetDefaultPeerDiscoveryManagerConfig().SufficientNumPeers)
	numNeeded := sufficientNumPeers - numPeers
	if numNeeded > 0 {
		numToAdd := len(addresses) * peersAddressesSubSamplingPercent / 100
		if numToAdd < 1 {
			numToAdd = 1
		} else if numToAdd > numNeeded {
			numToAdd = numNeeded
		}
		logger.Infof("Already has %v peers, attempt to connect to %v discovered peers", numPeers, numToAdd)

		perm := rand.Perm(len(addresses))
		for i := 0; i < numToAdd; i++ {
			go func(i int) {
				time.Sleep(time.Duration(rand.Int63n(discoverInterval)) * time.Millisecond)
				j := perm[i]
				peerNetAddress := addresses[j]
				peer, err := pdmh.discMgr.connectToOutboundPeer(peerNetAddress, true)
				if err != nil {
					logger.Debugf("Failed to connect to discovery peer %v: %v", peerNetAddress.String(), err)
				} else {
					logger.Infof("Successfully connected to discovery peer %v", peerNetAddress.String())
				}
				if pdmh.discoveryCallback != nil {
					pdmh.discoveryCallback(peer, err)
				}
			}(i)
		}
	} else {
		logger.Infof("No need to proactively connect to more peers, numPeers: %v, sufficientNumPeers: %v",
			numPeers, sufficientNumPeers)
	}
}

func (pdmh *PeerDiscoveryMessageHandler) maintainSufficientConnectivityRoutine() {
	defer pdmh.wg.Done()

	peerDiscoveryPulse := time.NewTicker(pdmh.peerDiscoveryPulseInterval)
	for {
		select {
		case <-peerDiscoveryPulse.C:
			pdmh.maintainSufficientConnectivity()
		}
	}
}

// maintainSufficientConnectivity tries to maintain sufficient number
// of connections by dialing peers when the number of connected peers are lower than the
// required threshold
func (pdmh *PeerDiscoveryMessageHandler) maintainSufficientConnectivity() {
	selfNodeType := viper.GetInt(common.CfgNodeType)
	skipEdgeNode := (selfNodeType == int(common.NodeTypeBlockchainNode)) // a blockchain node only asks other blockchain nodes for peers
	peers := *(pdmh.discMgr.peerTable.GetAllPeers(skipEdgeNode))
	numPeers := uint(len(peers))

	sufficientNumPeers := GetDefaultPeerDiscoveryManagerConfig().SufficientNumPeers
	if numPeers > 0 {
		if numPeers < sufficientNumPeers {
			logger.Infof("Attempt to maintain sufficient connectivity...")

			// recover persisted peers
			var peerNetAddresses []*netutil.NetAddress
			prevPeerAddrs, err := pdmh.discMgr.peerTable.RetrievePreviousPeers()
			if err == nil {
				for _, addr := range prevPeerAddrs {
					if !pdmh.discMgr.peerTable.PeerAddrExists(addr) {
						peerNetAddresses = append(peerNetAddresses, addr)
					}
				}
				if len(peerNetAddresses) > 0 {
					pdmh.connectToOutboundPeers(peerNetAddresses)
				}
			}

			// discovery
			numPeersToSendRequest := numPeers * requestPeersAddressesPercent / 100
			if numPeersToSendRequest < 1 {
				numPeersToSendRequest = 1
			}
			perm := rand.Perm(int(numPeers))
			for i := uint(0); i < numPeersToSendRequest; i++ {
				time.Sleep(time.Duration(rand.Int63n(discoverInterval)) * time.Millisecond)
				peer := peers[perm[i]]
				pdmh.requestAddresses(peer)
			}

			logger.Infof("Sent peer discovery requests to %v peers", numPeersToSendRequest)
		} else {
			logger.Infof("Already has sufficient number of peers, numPeers: %v, sufficientNumPeers: %v", numPeers, sufficientNumPeers)
		}
	} else { // no peer left in the peer table, try to reconnect to seed peers
		pdmh.discMgr.seedPeerConnector.connectToSeedPeers()
	}
}

func (pdmh *PeerDiscoveryMessageHandler) requestAddresses(peer *pr.Peer) {
	message := PeerDiscoveryMessage{
		Type: peerAddressesRequestType,
	}
	peer.Send(common.ChannelIDPeerDiscovery, message)
}

func (pdmh *PeerDiscoveryMessageHandler) sendAddresses(peer *pr.Peer, peerIDAddrs []pr.PeerIDAddress) {
	message := PeerDiscoveryMessage{
		Type:      peerAddressesReplyType,
		Addresses: peerIDAddrs,
	}
	peer.Send(common.ChannelIDPeerDiscovery, message)
}

func Fuzz(data []byte) int {
	if _, err := decodePeerDiscoveryMessage(data); err != nil {
		return 1
	}
	return 0
}

func decodePeerDiscoveryMessage(msgBytes common.Bytes) (message PeerDiscoveryMessage, err error) {
	err = rlp.DecodeBytes(msgBytes, &message)
	return
}

func seedPeerOnlyOutbound() bool {
	seedOnlyOutbound := viper.GetBool(common.CfgP2PSeedPeerOnlyOutbound)
	return seedOnlyOutbound
}
