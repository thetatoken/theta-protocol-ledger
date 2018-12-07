package messenger

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	cn "github.com/thetatoken/ukulele/p2p/connection"
	"github.com/thetatoken/ukulele/p2p/netutil"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

//
// PeerDiscoveryManager manages the peer discovery process
//
type PeerDiscoveryManager struct {
	messenger *Messenger

	addrBook  *AddrBook
	peerTable *pr.PeerTable
	nodeInfo  *p2ptypes.NodeInfo

	// Three mechanisms for peer discovery
	seedPeerConnector   SeedPeerConnector           // pro-actively connect to seed peers
	peerDiscMsgHandler  PeerDiscoveryMessageHandler // pro-actively connect to peer candidates obtained from connected peers
	inboundPeerListener InboundPeerListener         // listen to incoming peering requests

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

//
// PeerDiscoveryManagerConfig specifies the configuration for PeerDiscoveryManager
//
type PeerDiscoveryManagerConfig struct {
	MaxNumPeers        uint
	SufficientNumPeers uint
}

// CreatePeerDiscoveryManager creates an instance of the PeerDiscoveryManager
func CreatePeerDiscoveryManager(msgr *Messenger, nodeInfo *p2ptypes.NodeInfo, addrBookFilePath string,
	routabilityRestrict bool, seedPeerNetAddresses []string,
	networkProtocol string, localNetworkAddr string, skipUPNP bool, peerTable *pr.PeerTable,
	config PeerDiscoveryManagerConfig) (*PeerDiscoveryManager, error) {

	discMgr := &PeerDiscoveryManager{
		messenger: msgr,
		nodeInfo:  nodeInfo,
		peerTable: peerTable,
		wg:        &sync.WaitGroup{},
	}

	discMgr.addrBook = NewAddrBook(addrBookFilePath, routabilityRestrict)

	var err error
	discMgr.seedPeerConnector, err = createSeedPeerConnector(discMgr, localNetworkAddr, seedPeerNetAddresses)
	if err != nil {
		return discMgr, err
	}

	discMgr.peerDiscMsgHandler, err = createPeerDiscoveryMessageHandler(discMgr, localNetworkAddr)
	if err != nil {
		return discMgr, err
	}

	inlConfig := GetDefaultInboundPeerListenerConfig()
	discMgr.inboundPeerListener, err = createInboundPeerListener(discMgr, networkProtocol, localNetworkAddr, skipUPNP, inlConfig)
	if err != nil {
		return discMgr, err
	}
	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			log.Infof("Inbound peer connected, ID: %v, from: %v", peer.ID(), peer.GetConnection().GetNetconn().RemoteAddr())
		} else {
			log.Errorf("Inbound peer listener error: %v", err)
		}
	})

	return discMgr, nil
}

// GetDefaultPeerDiscoveryManagerConfig returns the default config for the PeerDiscoveryManager
func GetDefaultPeerDiscoveryManagerConfig() PeerDiscoveryManagerConfig {
	return PeerDiscoveryManagerConfig{
		MaxNumPeers:        128,
		SufficientNumPeers: 32,
	}
}

// SetMessenger sets the Messenger for the PeerDiscoveryManager
func (discMgr *PeerDiscoveryManager) SetMessenger(msgr *Messenger) {
	discMgr.messenger = msgr
}

// Start is called when the PeerDiscoveryManager starts
func (discMgr *PeerDiscoveryManager) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	discMgr.ctx = c
	discMgr.cancel = cancel

	var err error
	err = discMgr.seedPeerConnector.Start(c)
	if err != nil {
		return err
	}

	err = discMgr.inboundPeerListener.Start(c)
	if err != nil {
		return err
	}

	err = discMgr.peerDiscMsgHandler.Start(c)
	if err != nil {
		return err
	}

	return nil
}

// Stop is called when the PeerDiscoveryManager stops
func (discMgr *PeerDiscoveryManager) Stop() {
	discMgr.cancel()
}

// Wait suspends the caller goroutine
func (discMgr *PeerDiscoveryManager) Wait() {
	discMgr.seedPeerConnector.wg.Wait()
	discMgr.inboundPeerListener.wg.Wait()
	discMgr.peerDiscMsgHandler.wg.Wait()
	discMgr.wg.Wait()
}

// HandlePeerWithErrors handles peers that are in the error state.
// If the peer is persistent, it will attempt to reconnect to the
// peer. Otherwise, it disconnects from that peer
func (discMgr *PeerDiscoveryManager) HandlePeerWithErrors(peer *pr.Peer) {
	discMgr.peerTable.DeletePeer(peer.ID())
	peer.Stop()

	if peer.IsPersistent() {
		var err error
		for i := 0; i < 3; i++ { // retry up to 3 times
			if peer.IsOutbound() {
				_, err = discMgr.connectToOutboundPeer(peer.NetAddress(), true)
			} else {
				_, err = discMgr.connectWithInboundPeer(peer.GetConnection().GetNetconn(), true)
			}
			if err == nil {
				log.Infof("[p2p] Successfully re-connected to peer %v", peer.NetAddress().String())
				return
			}
			time.Sleep(time.Second * 3)
		}
		log.Errorf("[p2p] Failed to re-connect to peer %v: %v", peer.NetAddress().String(), err)
	}
}

func (discMgr *PeerDiscoveryManager) connectToOutboundPeer(peerNetAddress *netutil.NetAddress, persistent bool) (*pr.Peer, error) {
	log.Infof("[p2p] Connecting to outbound peer: %v...", peerNetAddress)
	peerConfig := pr.GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	peer, err := pr.CreateOutboundPeer(peerNetAddress, peerConfig, connConfig)
	if err != nil {
		log.Errorf("[p2p] Failed to create outbound peer: %v", peerNetAddress)
		return nil, err
	}
	peer.SetPersistency(persistent)
	err = discMgr.handshakeAndAddPeer(peer)
	return peer, err
}

func (discMgr *PeerDiscoveryManager) connectWithInboundPeer(netconn net.Conn, persistent bool) (*pr.Peer, error) {
	log.Infof("[p2p] Connecting with inbound peer: %v...", netconn.RemoteAddr())
	peerConfig := pr.GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	peer, err := pr.CreateInboundPeer(netconn, peerConfig, connConfig)
	if err != nil {
		log.Errorf("[p2p] Failed to create inbound peer: %v", netconn.RemoteAddr())
		return nil, err
	}
	peer.SetPersistency(persistent)
	err = discMgr.handshakeAndAddPeer(peer)
	return peer, err
}

// handshakeAndAddPeer performs handshake with a peer. Upon successful handshake,
// it save the peer to the peer table
func (discMgr *PeerDiscoveryManager) handshakeAndAddPeer(peer *pr.Peer) error {
	if err := peer.Handshake(discMgr.nodeInfo); err != nil {
		log.Errorf("[p2p] Failed to handshake with peer, error: %v", err)
		return err
	}

	if discMgr.messenger != nil {
		discMgr.messenger.AttachMessageHandlersToPeer(peer)
	} else {
		log.Warnf("[p2p] discMgr.messenger not set, cannot attach message handlers")
	}

	if !peer.Start(discMgr.ctx) {
		errMsg := "[p2p] Failed to start peer"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if !discMgr.peerTable.AddPeer(peer) {
		errMsg := "[p2p] Failed to add peer to the peerTable"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	netAddr := pr.GetPeerNetAddress(peer.GetRemoteAddress(), peer.IsOutbound())
	discMgr.addrBook.AddAddress(netAddr, netAddr)

	return nil
}
