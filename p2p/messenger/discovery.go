package messenger

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	cn "github.com/thetatoken/theta/p2p/connection"
	"github.com/thetatoken/theta/p2p/netutil"
	pr "github.com/thetatoken/theta/p2p/peer"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

//
// PeerDiscoveryManager manages the peer discovery process
//
type PeerDiscoveryManager struct {
	messenger *Messenger

	//addrBook  *AddrBook
	peerTable *pr.PeerTable
	nodeInfo  *p2ptypes.NodeInfo
	seedPeers map[string]*pr.Peer
	mutex     *sync.Mutex

	seedPeerOnly bool

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
	MaxNumPeers        int
	SufficientNumPeers uint
}

// CreatePeerDiscoveryManager creates an instance of the PeerDiscoveryManager
func CreatePeerDiscoveryManager(msgr *Messenger, nodeInfo *p2ptypes.NodeInfo, addrBookFilePath string,
	routabilityRestrict bool, seedPeerNetAddresses []string,
	networkProtocol string, localNetworkAddr string, externalPort int, skipUPNP bool, peerTable *pr.PeerTable,
	config PeerDiscoveryManagerConfig) (*PeerDiscoveryManager, error) {

	discMgr := &PeerDiscoveryManager{
		messenger:    msgr,
		nodeInfo:     nodeInfo,
		peerTable:    peerTable,
		seedPeers:    make(map[string]*pr.Peer),
		mutex:        &sync.Mutex{},
		seedPeerOnly: viper.GetBool(common.CfgP2PSeedPeerOnly),
		wg:           &sync.WaitGroup{},
	}

	//discMgr.addrBook = NewAddrBook(addrBookFilePath, routabilityRestrict)

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
	discMgr.inboundPeerListener, err = createInboundPeerListener(discMgr, networkProtocol, localNetworkAddr, externalPort, skipUPNP, inlConfig)
	if err != nil {
		return discMgr, err
	}
	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			logger.Infof("Inbound peer connected, ID: %v, from: %v", peer.ID(), peer.GetConnection().GetNetconn().RemoteAddr())
		} else {
			logger.Warnf("Inbound peer listener error: %v", err)
		}
	})

	return discMgr, nil
}

// GetDefaultPeerDiscoveryManagerConfig returns the default config for the PeerDiscoveryManager
func GetDefaultPeerDiscoveryManagerConfig() PeerDiscoveryManagerConfig {
	return PeerDiscoveryManagerConfig{
		MaxNumPeers:        viper.GetInt(common.CfgP2PMaxNumPeers),
		SufficientNumPeers: uint(viper.GetInt(common.CfgP2PMinNumPeers)),
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

	if discMgr.seedPeerOnly {
		return nil // if seed peer only, we don't need to start the peer discovery manager
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
	peerRemoteAddress := peer.GetConnection().GetNetconn().RemoteAddr().String()
	lookedUpPeer := discMgr.peerTable.GetPeer(peer.ID())
	if lookedUpPeer == nil {
		logger.Warnf("HandlePeerWithErrors cannot find the peer: %v", peer.ID())
		return // Should not happen
	}
	lookedUpPeerRemoteAddress := lookedUpPeer.GetConnection().GetNetconn().RemoteAddr().String()

	logger.Infof("HandlePeerWithErrors, peerRemoteAddress: %v", peerRemoteAddress)
	logger.Infof("HandlePeerWithErrors, lookedUpPeerRemoteAddress: %v", lookedUpPeerRemoteAddress)
	if peerRemoteAddress != lookedUpPeerRemoteAddress {
		logger.Warnf("Will not reconnect, since peerRemoteAddress and lookedUpPeerRemoteAddress are not the same")
		return
		// lookedUpPeer might be created by the inbound connection. A senario is that
		// the peer restarted and established a new connection with us. In this case,
		// we should not proceed to reconnect
	}

	discMgr.peerTable.DeletePeer(peer.ID())
	peer.Stop() // TODO: may need to stop peer regardless of the remote address comparison

	seedPeerOnly := viper.GetBool(common.CfgP2PSeedPeerOnly)

	//shouldRetry := seedPeerOnly && peer.IsPersistent()
	shouldRetry := (seedPeerOnly && peer.IsSeed()) || (!seedPeerOnly && !peer.IsSeed()) // avoid bombarding the seed nodes
	if shouldRetry {
		logger.Infof("Lost connection to peer %v with IP address %v, trying to re-connect", peer.ID(), peer.NetAddress().String())

		var err error
		for i := 0; i < 3; i++ { // retry up to 3 times
			if peer.IsOutbound() {
				_, err = discMgr.connectToOutboundPeer(peer.NetAddress(), true)
			} else {
				// For now not to retry connecting to the inbound peer, since that peer will
				// retry to etablish the connection
				//_, err = discMgr.connectWithInboundPeer(peer.GetConnection().GetNetconn(), true)
			}
			if err == nil {
				logger.Infof("Successfully re-connected to peer %v", peer.NetAddress().String())
				return
			}
			time.Sleep(time.Second * 3)
		}
		logger.Warnf("Failed to re-connect to peer %v with IP address %v: %v", peer.ID(), peer.NetAddress().String(), err)
	}
}

func (discMgr *PeerDiscoveryManager) connectToOutboundPeer(peerNetAddress *netutil.NetAddress, persistent bool) (*pr.Peer, error) {
	logger.Debugf("Connecting to outbound peer: %v...", peerNetAddress)
	peerConfig := pr.GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	peer, err := pr.CreateOutboundPeer(peerNetAddress, peerConfig, connConfig)
	if err != nil {
		logger.Debugf("Failed to create outbound peer: %v", peerNetAddress)
		return nil, err
	}
	peer.SetPersistency(persistent)
	err = discMgr.handshakeAndAddPeer(peer)
	return peer, err
}

func (discMgr *PeerDiscoveryManager) connectWithInboundPeer(netconn net.Conn, persistent bool) (*pr.Peer, error) {
	logger.Infof("Connecting with inbound peer: %v...", netconn.RemoteAddr())
	peerConfig := pr.GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	peer, err := pr.CreateInboundPeer(netconn, peerConfig, connConfig)
	if err != nil {
		logger.Warnf("Failed to create inbound peer: %v", netconn.RemoteAddr())
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
		logger.Warnf("Failed to handshake with peer, error: %v", err)
		return err
	}

	isSeed := discMgr.seedPeerConnector.isASeedPeer(peer.NetAddress())
	peer.SetSeed(isSeed)
	if isSeed {
		logger.Infof("Handshaked with a seed peer: %v, isOutbound: %v", peer.NetAddress(), peer.IsOutbound())
	}

	if discMgr.messenger != nil {
		discMgr.messenger.AttachMessageHandlersToPeer(peer)
	} else {
		logger.Warnf("discMgr.messenger not set, cannot attach message handlers")
	}

	if !peer.Start(discMgr.ctx) {
		errMsg := "Failed to start peer"
		logger.Warnf(errMsg)
		return errors.New(errMsg)
	}

	if !discMgr.peerTable.AddPeer(peer) {
		errMsg := "Failed to add peer to the peerTable"
		logger.Warnf(errMsg)
		return errors.New(errMsg)
	}

	//discMgr.addrBook.AddAddress(peer.NetAddress(), peer.NetAddress())
	//discMgr.addrBook.Save()

	if peer.IsSeed() {
		discMgr.mutex.Lock()
		defer discMgr.mutex.Unlock()

		discMgr.seedPeers[peer.ID()] = peer
	}

	return nil
}

func (discMgr *PeerDiscoveryManager) isSeedPeer(pid string) bool {
	discMgr.mutex.Lock()
	defer discMgr.mutex.Unlock()

	_, isSeed := discMgr.seedPeers[pid]
	return isSeed
}
