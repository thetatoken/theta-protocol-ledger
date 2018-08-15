package peer

import (
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/p2p"
	cn "github.com/thetatoken/ukulele/p2p/connection"
	nu "github.com/thetatoken/ukulele/p2p/netutil"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

//
// Peer models a peer node in a network
//
type Peer struct {
	connection *cn.Connection

	isPersistent bool
	isOutbound   bool
	key          string

	nodeInfo *p2p.NodeInfo

	config PeerConfig
}

//
// PeerConfig specifies the configuration of a peer
//
type PeerConfig struct {
	DialTimeout time.Duration
}

// CreateOutboundPeer creates an instance of an outbound peer
func CreateOutboundPeer(nodeInfo *p2p.NodeInfo, peerAddr *nu.NetAddress, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
	peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	netconn, err := dial(peerAddr, peerConfig)
	if err != nil {
		log.Errorf("[p2p] Error dialing the peer: %v", peerAddr)
		return nil, err
	}
	peer := createPeer(nodeInfo, netconn, true, onReceive, onError, peerConfig, connConfig)
	return peer, nil
}

// CreateInboundPeer creates an instance of an inbound peer
func CreateInboundPeer(nodeInfo *p2p.NodeInfo, netconn net.Conn, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
	peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	peer := createPeer(nodeInfo, netconn, true, onReceive, onError, peerConfig, connConfig)
	return peer, nil
}

func createPeer(nodeInfo *p2p.NodeInfo, netconn net.Conn, isOutbound bool, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
	peerConfig PeerConfig, connConfig cn.ConnectionConfig) *Peer {
	connection := cn.CreateConnection(netconn, onReceive, onError, connConfig)
	peer := &Peer{
		connection: connection,
		isOutbound: isOutbound,
		nodeInfo:   nodeInfo,
		config:     peerConfig,
	}
	return peer
}

// OnStart is called when the peer starts
func (peer *Peer) OnStart() bool {
	success := peer.connection.OnStart()
	return success
}

// OnStop is called when the peer stops
func (peer *Peer) OnStop() {
	peer.connection.OnStop()
}

// Handshake handles the initial signaling between two peers
func (peer *Peer) Handshake(timeout time.Duration) error {
	peer.connection.GetNetconn().SetDeadline(time.Now().Add(timeout))
	var sendError error
	var recvError error
	targetPeerNodeInfo := p2p.NodeInfo{}
	cmn.Parallel(
		func() { sendError = rlp.Encode(peer.connection.GetNetconn(), peer.nodeInfo) },
		func() { recvError = rlp.Decode(peer.connection.GetNetconn(), targetPeerNodeInfo) },
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

	return nil
}

// Send sends the given message through the specified channel to the target peer
func (peer *Peer) Send(channelID byte, message interface{}) bool {
	success := peer.connection.EnqueueMessage(channelID, message)
	return success
}

// AttemptToSend attempts to send the given message through the specified channel to the target peer (non-blocking)
func (peer *Peer) AttemptToSend(channelID byte, message interface{}) bool {
	success := peer.connection.AttemptToEnqueueMessage(channelID, message)
	return success
}

// CanSend indicates whether more messages can be sent through the specified channel
func (peer *Peer) CanSend(channelID byte) bool {
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

func (peer *Peer) Key() string {
	if len(peer.key) == 0 {
		keyBytes := crypto.PubkeyToAddress(peer.nodeInfo.PubKey)
		peer.key = string(keyBytes[:])
	}
	return peer.key
}

func dial(addr *nu.NetAddress, config PeerConfig) (net.Conn, error) {
	conn, err := addr.DialTimeout(config.DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
