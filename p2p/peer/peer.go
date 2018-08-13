package peer

import (
	"crypto/ecdsa"
	"net"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/ukulele/common"
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

	nodeInfo NodeInfo

	config PeerConfig
}

//
// PeerConfig specifies the configuration of a peer
//
type PeerConfig struct {
	DialTimeout time.Duration
}

//
// NodeInfo provides the information of the current node
//
type NodeInfo struct {
	PubKey ecdsa.PublicKey
}

// CreateOutboundPeer creates an instance of an outbound peer
func CreateOutboundPeer(nodeInfo NodeInfo, peerAddr *nu.NetAddress, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
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
func CreateInboundPeer(nodeInfo NodeInfo, netconn net.Conn, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
	peerConfig PeerConfig, connConfig cn.ConnectionConfig) (*Peer, error) {
	peer := createPeer(nodeInfo, netconn, true, onReceive, onError, peerConfig, connConfig)
	return peer, nil
}

func createPeer(nodeInfo NodeInfo, netconn net.Conn, isOutbound bool, onReceive cn.ReceiveHandler, onError cn.ErrorHandler,
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

func (peer *Peer) OnStart() bool {
	success := peer.connection.OnStart()
	return success
}

func (peer *Peer) OnStop() {
	peer.connection.OnStop()
}

func (peer *Peer) Handshake(timeout time.Duration) error {
	peer.connection.GetNetconn().SetDeadline(time.Now().Add(timeout))
	var sendError error
	var recvError error
	targetPeerNodeInfo := NodeInfo{}
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

func (peer *Peer) Send(channelID byte, message interface{}) bool {
	success := peer.connection.EnqueueMessage(channelID, message)
	return success
}

func (peer *Peer) AttemptToSend(channelID byte, message interface{}) bool {
	success := peer.connection.AttemptToEnqueueMessage(channelID, message)
	return success
}

func (peer *Peer) CanSend(channelID byte) bool {
	canSend := peer.connection.CanEnqueueMessage(channelID)
	return canSend
}

func (peer *Peer) GetConnection() *cn.Connection {
	return peer.connection
}

func (peer *Peer) GetRemoteAddress() net.Addr {
	return peer.connection.GetNetconn().RemoteAddr()
}

func (peer *Peer) SetPersistency(persistent bool) {
	peer.isPersistent = persistent
}

func (peer *Peer) IsPersistent() bool {
	return peer.isPersistent
}

func (peer *Peer) IsOutbound() bool {
	return peer.isOutbound
}

func dial(addr *nu.NetAddress, config PeerConfig) (net.Conn, error) {
	conn, err := addr.DialTimeout(config.DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
