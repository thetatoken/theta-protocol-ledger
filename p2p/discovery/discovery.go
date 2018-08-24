package discovery

import (
	"errors"
	"net"

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
	addrBook  *AddrBook
	peerTable *pr.PeerTable
	nodeInfo  *p2ptypes.NodeInfo

	// Three mechanisms for peer discovery
	seedPeerConnector   SeedPeerConnector           // pro-actively connect to seed peers
	peerDiscMsgHandler  PeerDiscoveryMessageHandler // pro-actively connect to peer candidates obtained from connected peers
	inboundPeerListener InboundPeerListener         // listen to incoming peering requests
}

//
// PeerDiscoveryManagerConfig specifies the configuration for PeerDiscoveryManager
//
type PeerDiscoveryManagerConfig struct {
	MaxNumPeers uint
}

// CreatePeerDiscoveryManager creates an instance of the PeerDiscoveryManager
func CreatePeerDiscoveryManager(nodeInfo *p2ptypes.NodeInfo, addrBookFilePath string,
	routabilityRestrict bool, selfNetAddressStr string, seedPeerNetAddressStrs []string,
	networkProtocol string, localNetworkAddr string, skipUPNP bool, peerTable *pr.PeerTable,
	config PeerDiscoveryManagerConfig) (*PeerDiscoveryManager, error) {

	discMgr := &PeerDiscoveryManager{
		nodeInfo:  nodeInfo,
		peerTable: peerTable,
	}

	discMgr.addrBook = NewAddrBook(addrBookFilePath, routabilityRestrict)

	var err error
	discMgr.seedPeerConnector, err = createSeedPeerConnector(discMgr, selfNetAddressStr, seedPeerNetAddressStrs)
	if err != nil {
		return discMgr, err
	}

	discMgr.peerDiscMsgHandler, err = createPeerDiscoveryMessageHandler(discMgr)
	if err != nil {
		return discMgr, err
	}

	inlConfig := GetDefaultInboundPeerListenerConfig()
	discMgr.inboundPeerListener, err = createInboundPeerListener(discMgr, networkProtocol, localNetworkAddr, skipUPNP, inlConfig)
	if err != nil {
		return discMgr, err
	}
	return discMgr, nil
}

// GetDefaultPeerDiscoveryManagerConfig returns the default config for the PeerDiscoveryManager
func GetDefaultPeerDiscoveryManagerConfig() PeerDiscoveryManagerConfig {
	return PeerDiscoveryManagerConfig{
		MaxNumPeers: 128,
	}
}

// OnStart is called when the PeerDiscoveryManager starts
func (discMgr *PeerDiscoveryManager) OnStart() error {
	var err error
	err = discMgr.seedPeerConnector.OnStart()
	if err != nil {
		return err
	}

	err = discMgr.inboundPeerListener.OnStart()
	if err != nil {
		return err
	}

	err = discMgr.peerDiscMsgHandler.OnStart()
	if err != nil {
		return err
	}

	return nil
}

// OnStop is called when the PeerDiscoveryManager stops
func (discMgr *PeerDiscoveryManager) OnStop() {
	discMgr.seedPeerConnector.OnStop()
	discMgr.inboundPeerListener.OnStop()
	discMgr.peerDiscMsgHandler.OnStop()
}

// HandlePeerWithErrors handles peers that are in the error state.
// If the peer is persistent, it will attempt to reconnect to the
// peer. Otherwise, it disconnects from that peer
func (discMgr *PeerDiscoveryManager) HandlePeerWithErrors(peer *pr.Peer) {
	// TODO: implementation
}

func (discMgr *PeerDiscoveryManager) connectToOutboundPeer(peerNetAddress *netutil.NetAddress, persistent bool) (*pr.Peer, error) {
	log.Infof("[p2p] Connectiong to outbound peer: %v...", peerNetAddress)
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
	log.Infof("[p2p] Connectiong with inbound peer: %v...", netconn.RemoteAddr())
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

	if !peer.OnStart() {
		errMsg := "[p2p] Failed to start peer"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if !discMgr.peerTable.AddPeer(peer) {
		errMsg := "[p2p] Failed to add peer to the peerTable"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	return nil
}
