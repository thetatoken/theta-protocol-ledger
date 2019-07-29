package messenger

import (
	// "bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	cn "github.com/thetatoken/theta/p2p/connection"
	"github.com/thetatoken/theta/p2p/netutil"
	pr "github.com/thetatoken/theta/p2p/peer"
	p2ptypes "github.com/thetatoken/theta/p2p/types"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	// "github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/protocol"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	// dht "github.com/libp2p/go-libp2p-kad-dht"
	ma "github.com/multiformats/go-multiaddr"
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
	// seedPeerConnector   SeedPeerConnector           // pro-actively connect to seed peers
	// peerDiscMsgHandler  PeerDiscoveryMessageHandler // pro-actively connect to peer candidates obtained from connected peers
	// inboundPeerListener InboundPeerListener         // listen to incoming peering requests

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	// libp2p
	host host.Host

	// seedPeerAddresses []ma.Multiaddr
}

//
// PeerDiscoveryManagerConfig specifies the configuration for PeerDiscoveryManager
//
type PeerDiscoveryManagerConfig struct {
	MaxNumPeers        uint
	SufficientNumPeers uint
}

func createP2pAddr(netAddr string) (ma.Multiaddr, error) {
	ip, port, err := net.SplitHostPort(netAddr)
	if err != nil {
		return nil, err
	}
	multiAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%v/tcp/%v", ip, port))
	if err != nil {
		return nil, err
	}
	return multiAddr, nil
}

func createP2pAddrIpfs(netAddr, ipfs string) (ma.Multiaddr, error) {
	ip, port, err := net.SplitHostPort(netAddr)
	if err != nil {
		return nil, err
	}
	multiAddr, err := ma.NewMultiaddr(fmt.Sprintf("/ip4/%v/tcp/%v/p2p/%v", ip, port, ipfs))
	if err != nil {
		return nil, err
	}
	return multiAddr, nil
}

func handleStream(stream network.Stream) {
	logger.Info("################# Got a new stream!")

	// rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

	// go readData(rw)
	// go writeData(rw)
}

// CreatePeerDiscoveryManager creates an instance of the PeerDiscoveryManager
func CreatePeerDiscoveryManager(msgr *Messenger, nodeInfo *p2ptypes.NodeInfo, addrBookFilePath string,
	routabilityRestrict bool, seedPeerNetAddresses []string,
	networkProtocol string, localNetworkAddr string, skipUPNP bool, peerTable *pr.PeerTable,
	config PeerDiscoveryManagerConfig) (*PeerDiscoveryManager, error) {

	discMgr := &PeerDiscoveryManager{
		messenger:             msgr,
		nodeInfo:              nodeInfo,
		peerTable:             peerTable,
		wg:                    &sync.WaitGroup{},
	}

	discMgr.addrBook = NewAddrBook(addrBookFilePath, routabilityRestrict)

	// var err error
	// discMgr.seedPeerConnector, err = createSeedPeerConnector(discMgr, localNetworkAddr, seedPeerNetAddresses)
	// if err != nil {
	// 	return discMgr, err
	// }

	// discMgr.peerDiscMsgHandler, err = createPeerDiscoveryMessageHandler(discMgr, localNetworkAddr)
	// if err != nil {
	// 	return discMgr, err
	// }

	// inlConfig := GetDefaultInboundPeerListenerConfig()
	// discMgr.inboundPeerListener, err = createInboundPeerListener(discMgr, networkProtocol, localNetworkAddr, skipUPNP, inlConfig)
	// if err != nil {
	// 	return discMgr, err
	// }
	// discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
	// 	if err == nil {
	// 		logger.Infof("Inbound peer connected, ID: %v, from: %v", peer.ID(), peer.GetConnection().GetNetconn().RemoteAddr())
	// 	} else {
	// 		logger.Errorf("Inbound peer listener error: %v", err)
	// 	}
	// })

	logger.Warnf("=-=-=-=-=-=-=-=0 %v", localNetworkAddr)
	logger.Warnf("=-=-=-=-=-=-=-=1 %v", nodeInfo.PubKey.Address().Hex())

	hostId, _, err := crypto.GenerateEd25519Key(strings.NewReader(nodeInfo.PubKey.Address().Hex()))
	if err != nil {
		return discMgr, err
	}
	localP2pAddr, err := createP2pAddr(localNetworkAddr)
	if err != nil {
		return discMgr, err
	}
	discMgr.host, err = libp2p.New(
		context.Background(),
		libp2p.Identity(hostId),
		libp2p.ListenAddrs([]ma.Multiaddr{localP2pAddr}...),
	)
	if err != nil {
		return discMgr, err
	}

	logger.Warnf("=-=-=-=-=-=-=-= %v, %v", discMgr.host.ID(), discMgr.host.Addrs())

	// for _, seedAddr := range seedPeerNetAddresses {
	// 	p2pAddr, err := createP2pAddrIpfs(seedAddr, discMgr.host.ID().String())
	// 	if err != nil {
	// 		logger.Warnf("Can't convert seed %v to p2p address", seedAddr)
	// 	} else {
	// 		discMgr.seedPeerAddresses = append(discMgr.seedPeerAddresses, p2pAddr)
	// 	}
	// }

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

	localChainID := viper.GetString(common.CfgGenesisChainID)
	discMgr.host.SetStreamHandler(protocol.ID(localChainID), func(stream network.Stream) {
		logger.Warnf("<<<<< Received new stream: %v", stream.Conn().RemotePeer())

		peerConfig := pr.GetDefaultPeerConfig()
		connConfig := cn.GetDefaultConnectionConfig()
		peer := pr.CreatePeer(stream, peerConfig, connConfig)
		if peer == nil {
			logger.Errorf("Failed to create peer")
			return
		}

		if !peer.Start(ctx) {
			errMsg := "Failed to start peer"
			logger.Errorf(errMsg)
			return
		}
	})
	
	mdnsService, err := discovery.NewMdnsService(ctx, discMgr.host, time.Second * 10, "Theta2damoon") //temp: 3 Minute
	if err != nil {
		return err
	}

	mdnsService.RegisterNotifee(&discoveryNotifee{ctx, discMgr.host})

	// ///////
	// localChainID := viper.GetString(common.CfgGenesisChainID)
	// stream, err := d.host.NewStream(ctx, pi.ID, protocol.ID(localChainID))
	// if err != nil {
	// 	logger.Errorf("Failed to create stream for peer: %v", err)
	// 	return
	// }

	return nil
}

// Stop is called when the PeerDiscoveryManager stops
func (discMgr *PeerDiscoveryManager) Stop() {
	discMgr.cancel()
}

// Wait suspends the caller goroutine
func (discMgr *PeerDiscoveryManager) Wait() {
	// discMgr.seedPeerConnector.wg.Wait()
	// discMgr.inboundPeerListener.wg.Wait()
	// discMgr.peerDiscMsgHandler.wg.Wait()

	discMgr.wg.Wait()
}

// HandlePeerWithErrors handles peers that are in the error state.
// If the peer is persistent, it will attempt to reconnect to the
// peer. Otherwise, it disconnects from that peer
func (discMgr *PeerDiscoveryManager) HandlePeerWithErrors(peer *pr.Peer) {
	peerRemoteAddress := peer.GetConnection().GetNetconn().RemoteAddr().String()
	lookedUpPeer := discMgr.peerTable.GetPeer(peer.ID())
	if lookedUpPeer == nil {
		logger.Errorf("HandlePeerWithErrors cannot find the peer: %v", peer.ID())
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

	if peer.IsPersistent() {
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
		logger.Errorf("Failed to re-connect to peer %v: %v", peer.NetAddress().String(), err)
	}
}

func (discMgr *PeerDiscoveryManager) connectToOutboundPeer(peerNetAddress *netutil.NetAddress, persistent bool) (*pr.Peer, error) {
	logger.Infof("Connecting to outbound peer: %v...", peerNetAddress)
	peerConfig := pr.GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	peer, err := pr.CreateOutboundPeer(peerNetAddress, peerConfig, connConfig)
	if err != nil {
		logger.Warnf("Failed to create outbound peer: %v", peerNetAddress)
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
		logger.Errorf("Failed to create inbound peer: %v", netconn.RemoteAddr())
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
		logger.Errorf("Failed to handshake with peer, error: %v", err)
		return err
	}

	if discMgr.messenger != nil {
		discMgr.messenger.AttachMessageHandlersToPeer(peer)
	} else {
		logger.Warnf("discMgr.messenger not set, cannot attach message handlers")
	}

	if !peer.Start(discMgr.ctx) {
		errMsg := "Failed to start peer"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if !discMgr.peerTable.AddPeer(peer) {
		errMsg := "Failed to add peer to the peerTable"
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	discMgr.addrBook.AddAddress(peer.NetAddress(), peer.NetAddress())
	discMgr.addrBook.Save()

	return nil
}
