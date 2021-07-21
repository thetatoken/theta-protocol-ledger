package peer

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	cmn "github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	cn "github.com/thetatoken/theta/p2p/connection"
	nu "github.com/thetatoken/theta/p2p/netutil"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2p"})

const maxExtraHandshakeInfo = 4096

//
// Peer models a peer node in a network
//
type Peer struct {
	connection *cn.Connection

	isPersistent bool
	isOutbound   bool
	isSeed       bool
	netAddress   *nu.NetAddress

	nodeInfo p2ptypes.NodeInfo // information of the blockchain node of the peer
	nodeType cmn.NodeType
	config   PeerConfig

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

//
// PeerConfig specifies the configuration of a peer
//
type PeerConfig struct {
	HandshakeTimeout time.Duration
	DialTimeout      time.Duration
}

// CreateOutboundPeer creates an instance of an outbound peer
func CreateOutboundPeer(peerAddr *nu.NetAddress, peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	netconn, err := dial(peerAddr, peerConfig)
	if err != nil {
		logger.Debugf("Error dialing the peer: %v", peerAddr)
		return nil, err
	}
	peer := createPeer(netconn, true, peerConfig, connConfig)
	if peer == nil {
		return nil, errors.New("Failed to create outbound peer")
	}
	return peer, nil
}

// CreateInboundPeer creates an instance of an inbound peer
func CreateInboundPeer(netconn net.Conn, peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	peer := createPeer(netconn, false, peerConfig, connConfig)
	if peer == nil {
		return nil, errors.New("Failed to create inbound peer")
	}
	return peer, nil
}

// GetDefaultPeerConfig creates the default PeerConfig
func GetDefaultPeerConfig() PeerConfig {
	return PeerConfig{
		HandshakeTimeout: 10 * time.Second,
		DialTimeout:      10 * time.Second,
	}
}

// Start is called when the peer starts
// NOTE: need to call peer.Handshake() before peer.Start()
func (peer *Peer) Start(ctx context.Context) bool {
	c, cancel := context.WithCancel(ctx)
	peer.ctx = c
	peer.cancel = cancel

	success := peer.connection.Start(c)
	return success
}

// Wait suspends the caller goroutine
func (peer *Peer) Wait() {
	peer.wg.Wait()
}

// CancelConnection for testing purpose only
func (peer *Peer) CancelConnection() {
	peer.connection.CancelConnection()
}

// Stop is called when the peer stops
func (peer *Peer) Stop() {
	peer.connection.Stop()
}

// Handshake handles the initial signaling between two peers
// NOTE: need to call peer.Handshake() before peer.Start()
func (peer *Peer) Handshake(sourceNodeInfo *p2ptypes.NodeInfo) error {
	remoteAddr := peer.connection.GetNetconn().RemoteAddr()
	logger.Infof("Handshaking with %v...", remoteAddr)

	timeout := peer.config.HandshakeTimeout
	peer.connection.GetNetconn().SetDeadline(time.Now().Add(timeout))
	var s *rlp.Stream
	var sendError error
	var recvError error
	targetPeerNodeInfo := p2ptypes.NodeInfo{}
	cmn.Parallel(
		func() {
			sendError = rlp.Encode(peer.connection.GetBufNetconn(), sourceNodeInfo)
		},
		func() {
			s = rlp.NewStream(peer.connection.GetBufReader(), 1024)
			recvError = s.Decode(&targetPeerNodeInfo)
		},
	)
	if sendError != nil {
		logger.Warnf("Error during handshake/send: %v", sendError)
		return sendError
	}
	if recvError != nil {
		logger.Warnf("Error during handshake/recv: %v", recvError)
		return recvError
	}
	netconn := peer.connection.GetNetconn()
	targetNodePubKey, err := crypto.PublicKeyFromBytes(targetPeerNodeInfo.PubKeyBytes)
	if err != nil {
		logger.Warnf("Error during handshake/recv: %v", err)
		return err
	}
	targetPeerNodeInfo.PubKey = targetNodePubKey
	peer.nodeInfo = targetPeerNodeInfo

	// Forward compatibility.
	localChainID := viper.GetString(cmn.CfgGenesisChainID)
	selfNodeType := viper.GetInt(cmn.CfgNodeType)
	var peerType int
	cmn.Parallel(
		func() {
			sendError = rlp.Encode(peer.connection.GetBufNetconn(), localChainID)
			if sendError != nil {
				return
			}
			sendError = rlp.Encode(peer.connection.GetBufNetconn(), strconv.Itoa(selfNodeType))
			if sendError != nil {
				return
			}
			sendError = rlp.Encode(peer.connection.GetBufNetconn(), "EOH")
		},
		func() {
			var msg string
			recvError = s.Decode(&msg)
			if recvError != nil {
				return
			}
			if msg == "EOH" {
				return
			}
			if msg != localChainID {
				recvError = fmt.Errorf("ChainID mismatch: peer chainID: %v, local ChainID: %v", msg, localChainID)
				//return
			}
			logger.Infof("Peer ChainID: %v", msg)

			recvError = s.Decode(&msg)
			if recvError != nil {
				return
			}
			var convErr error
			peerType, convErr = strconv.Atoi(msg)
			if convErr != nil {
				//recvError = fmt.Errorf("Cannot parse the peer type: %v", msg)

				peerType = int(cmn.NodeTypeBlockchainNode)          // for backward compatibility, by default consider the peer as a blockchain node
				logger.Warnf("Cannot parse the peer type: %v", msg) // for backward compatibility, just print a warning instead of setting the recvError
				return
			}
			logger.Infof("Peer Type: %v", peerType)

			for {
				recvError = s.Decode(&msg)
				if recvError != nil {
					return
				}
				if msg == "EOH" {
					return
				}
			}
		},
	)
	if sendError != nil {
		logger.Warnf("Error during handshake/send extra info: %v", sendError)
		return sendError
	}
	if recvError != nil {
		logger.Warnf("Error during handshake/recv extra info: %v", recvError)
		return recvError
	}

	peer.nodeType = common.NodeType(peerType)

	remotePub, err := peer.connection.DoEncHandshake(
		crypto.PrivKeyToECDSA(sourceNodeInfo.PrivKey), crypto.PubKeyToECDSA(targetNodePubKey))
	if err != nil {
		logger.Warnf("Error during handshake/key exchange: %v", err)
		return err
	} else {
		if remotePub.Address() != targetNodePubKey.Address() {
			err = fmt.Errorf("expected remote address: %v, actual address: %v", targetNodePubKey.Address(), remotePub.Address())
			logger.Warnf("Error during handshake/key exchange: %v", err)
			return err
		}
	}
	logger.Infof("Using encrypted transport for peer: %v", targetNodePubKey.Address())

	if !peer.isOutbound {
		peer.SetNetAddress(nu.NewNetAddressWithEnforcedPort(netconn.RemoteAddr(), int(peer.nodeInfo.Port)))
	}

	netconn.SetDeadline(time.Time{})

	logger.Infof("Handshake completed, target address: %v, target public key: %v, address: %v",
		remoteAddr, hex.EncodeToString(targetNodePubKey.ToBytes()), targetNodePubKey.Address())

	return nil
}

// Send sends the given message through the specified channel to the target peer
func (peer *Peer) Send(channelID cmn.ChannelIDEnum, message interface{}) bool {
	success := peer.connection.EnqueueMessage(channelID, message)
	return success
}

// AttemptToSend attempts to send the given message through the specified channel to the target peer (non-blocking)
func (peer *Peer) AttemptToSend(channelID cmn.ChannelIDEnum, message interface{}) bool {
	success := peer.connection.AttemptToEnqueueMessage(channelID, message)
	return success
}

// CanSend indicates whether more messages can be sent through the specified channel
func (peer *Peer) CanSend(channelID cmn.ChannelIDEnum) bool {
	canSend := peer.connection.CanEnqueueMessage(channelID)
	return canSend
}

// GetConnection returns the connection object attached to the peer
func (peer *Peer) GetConnection() *cn.Connection {
	return peer.connection
}

// GetRemoteAddress returns the remote address of the peer
func (peer *Peer) GetRemoteAddress() net.Addr {
	return peer.connection.GetNetconn().RemoteAddr()
}

// SetPersistency sets the persistency for the given peer
func (peer *Peer) SetPersistency(persistent bool) {
	peer.isPersistent = persistent
}

// IsPersistent returns whether the peer is persistent
func (peer *Peer) IsPersistent() bool {
	return peer.isPersistent
}

// IsOutbound returns whether the peer is an outbound peer
func (peer *Peer) IsOutbound() bool {
	return peer.isOutbound
}

// NodeType returns the node type of the peer
func (peer *Peer) NodeType() cmn.NodeType {
	return peer.nodeType
}

// SetSeed sets the isSeed for the given peer
func (peer *Peer) SetSeed(isSeed bool) {
	peer.isSeed = isSeed
}

// IsSeed returns whether the peer is a seed peer
func (peer *Peer) IsSeed() bool {
	return peer.isSeed
}

// SetNetAddress sets the network address of the peer
func (peer *Peer) SetNetAddress(netAddr *nu.NetAddress) {
	peer.netAddress = netAddr
}

// NetAddress returns the network address of the peer
func (peer *Peer) NetAddress() *nu.NetAddress {
	return peer.netAddress
}

// SetPort sets the network port of the peer
func (peer *Peer) SetPort(port uint16) {
	peer.netAddress.Port = port
	peer.nodeInfo.Port = port
}

// ID returns the unique idenitifier of the peer in the P2P network
func (peer *Peer) ID() string {
	peerID := peer.nodeInfo.PubKey.Address() // use the blockchain address as the peer ID
	id := peerID.Hex()
	return id
}

func dial(addr *nu.NetAddress, config PeerConfig) (net.Conn, error) {
	netconn, err := addr.DialTimeout(config.DialTimeout)
	if err != nil {
		return nil, err
	}
	return netconn, nil
}

func createPeer(netconn net.Conn, isOutbound bool,
	peerConfig PeerConfig, connConfig cn.ConnectionConfig) *Peer {
	connection := cn.CreateConnection(netconn, connConfig)
	if connection == nil {
		logger.Warnf("Failed to create connection")
		if netconn != nil {
			netconn.Close()
		}
		return nil
	}
	var netAddress *nu.NetAddress
	if isOutbound {
		netAddress = nu.NewNetAddress(netconn.RemoteAddr())
	}
	peer := &Peer{
		connection: connection,
		isOutbound: isOutbound,
		netAddress: netAddress,
		config:     peerConfig,
		wg:         &sync.WaitGroup{},
	}
	return peer
}
