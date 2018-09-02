package peer

import (
	"errors"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/ukulele/common"
	cn "github.com/thetatoken/ukulele/p2p/connection"
	nu "github.com/thetatoken/ukulele/p2p/netutil"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

//
// Peer models a peer node in a network
//
type Peer struct {
	connection *cn.Connection

	isPersistent bool
	isOutbound   bool
	netAddress   *nu.NetAddress

	nodeInfo p2ptypes.NodeInfo // information of the blockchain node of the peer

	config PeerConfig
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
		log.Errorf("[p2p] Error dialing the peer: %v", peerAddr)
		return nil, err
	}
	peer := createPeer(netconn, true, peerConfig, connConfig)
	if peer == nil {
		return nil, errors.New("[p2p] Failed to create outbound peer")
	}
	return peer, nil
}

// CreateInboundPeer creates an instance of an inbound peer
func CreateInboundPeer(netconn net.Conn, peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	peer := createPeer(netconn, false, peerConfig, connConfig)
	if peer == nil {
		return nil, errors.New("[p2p] Failed to create inbound peer")
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

// OnStart is called when the peer starts
// NOTE: need to call peer.Handshake() before peer.OnStart()
func (peer *Peer) OnStart() bool {
	success := peer.connection.OnStart()
	return success
}

// OnStop is called when the peer stops
func (peer *Peer) OnStop() {
	peer.connection.OnStop()
}

// Handshake handles the initial signaling between two peers
// NOTE: need to call peer.Handshake() before peer.OnStart()
func (peer *Peer) Handshake(sourceNodeInfo *p2ptypes.NodeInfo) error {
	timeout := peer.config.HandshakeTimeout
	peer.connection.GetNetconn().SetDeadline(time.Now().Add(timeout))
	var sendError error
	var recvError error
	targetPeerNodeInfo := p2ptypes.NodeInfo{}
	cmn.Parallel(
		func() { sendError = rlp.Encode(peer.connection.GetNetconn(), sourceNodeInfo) },
		func() { recvError = rlp.Decode(peer.connection.GetNetconn(), &targetPeerNodeInfo) },
	)
	if sendError != nil {
		log.Errorf("[p2p] error during handshake/send: %v", sendError)
		return sendError
	}
	if recvError != nil {
		log.Errorf("[p2p] error during handshake/recv: %v", recvError)
		return recvError
	}
	peer.connection.GetNetconn().SetDeadline(time.Time{})
	peer.nodeInfo = targetPeerNodeInfo

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

// NetAddress returns the network address of the peer
func (peer *Peer) NetAddress() *nu.NetAddress {
	return peer.netAddress
}

// ID returns the unique idenitifier of the peer in the P2P network
func (peer *Peer) ID() string {
	peerID := peer.nodeInfo.Address // use the blockchain address as the peer ID
	return peerID
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
		log.Errorf("[p2p] Failed to create connection")
		return nil
	}
	peer := &Peer{
		connection: connection,
		isOutbound: isOutbound,
		netAddress: nu.NewNetAddress(netconn.RemoteAddr()),
		config:     peerConfig,
	}
	return peer
}
