package peer

import (
	"fmt"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/crypto"
	cn "github.com/thetatoken/theta/p2p/connection"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

func TestDefaultPeerTableAddPeer(t *testing.T) {
	assert := assert.New(t)

	port := 37856
	netconn := newIncomingNetconn(port)

	pt := newTestEmptyPeerTable()
	assert.Equal(uint(0), pt.GetTotalNumPeers(true))

	randPubKey1 := p2ptypes.GetTestRandPubKey()
	randPubKey2 := p2ptypes.GetTestRandPubKey()

	peer1 := newSimulatedInboundPeer(netconn, randPubKey1)
	success := pt.AddPeer(peer1)
	assert.True(success)
	assert.Equal(uint(1), pt.GetTotalNumPeers(true))
	assert.True(pt.PeerExists(peer1.ID()))
	assert.Equal(peer1, pt.GetPeer(peer1.ID()))

	peer1a := newSimulatedInboundPeer(netconn, randPubKey1)
	success = pt.AddPeer(peer1a)
	assert.True(success) // should overwrite the entry with same peer ID
	assert.Equal(uint(1), pt.GetTotalNumPeers(true))
	assert.Equal(peer1a, pt.GetPeer(peer1.ID()))
	assert.Equal(peer1a, pt.peers[0])

	peer2 := newSimulatedInboundPeer(netconn, randPubKey2)
	success = pt.AddPeer(peer2)
	assert.True(success)
	assert.Equal(uint(2), pt.GetTotalNumPeers(true))
	assert.True(pt.PeerExists(peer2.ID()))
	assert.Equal(peer2, pt.GetPeer(peer2.ID()))
}

func TestDefaultPeerTableDeletePeer(t *testing.T) {
	assert := assert.New(t)

	pt := newTestEmptyPeerTable()
	assert.Equal(uint(0), pt.GetTotalNumPeers(true))

	port := 37857
	netconn := newIncomingNetconn(port)

	peer1 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer2 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer3 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())

	assert.True(pt.AddPeer(peer1))
	assert.Equal(uint(1), pt.GetTotalNumPeers(true))
	assert.True(pt.AddPeer(peer2))
	assert.Equal(uint(2), pt.GetTotalNumPeers(true))
	assert.True(pt.AddPeer(peer3))
	assert.Equal(uint(3), pt.GetTotalNumPeers(true))

	assert.True(pt.PeerExists(peer1.ID()))
	assert.True(pt.PeerExists(peer2.ID()))
	assert.True(pt.PeerExists(peer3.ID()))

	pt.DeletePeer(peer1.ID())
	assert.False(pt.PeerExists(peer1.ID()))
	assert.Equal(uint(2), pt.GetTotalNumPeers(true))
	pt.DeletePeer(peer2.ID())
	assert.False(pt.PeerExists(peer2.ID()))
	assert.Equal(uint(1), pt.GetTotalNumPeers(true))
	pt.DeletePeer(peer3.ID())
	assert.False(pt.PeerExists(peer3.ID()))
	assert.Equal(uint(0), pt.GetTotalNumPeers(true))
}

func TestDefaultPeerIterationOrder(t *testing.T) {
	assert := assert.New(t)

	pt := newTestEmptyPeerTable()

	port := 37858
	netconn := newIncomingNetconn(port)

	peer1 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer2 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer3 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer4 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer5 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())
	peer6 := newSimulatedInboundPeer(netconn, p2ptypes.GetTestRandPubKey())

	assert.True(pt.AddPeer(peer1))
	assert.True(pt.AddPeer(peer2))
	assert.True(pt.AddPeer(peer3))
	assert.True(pt.AddPeer(peer4))
	assert.True(pt.AddPeer(peer5))
	assert.True(pt.AddPeer(peer6))

	allPeers := pt.GetAllPeers(true)
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer3)
	assert.Equal((*allPeers)[3], peer4)
	assert.Equal((*allPeers)[4], peer5)
	assert.Equal((*allPeers)[5], peer6)

	// Delete two peers

	pt.DeletePeer(peer3.ID())
	pt.DeletePeer(peer5.ID())

	allPeers = pt.GetAllPeers(true)
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer4)
	assert.Equal((*allPeers)[3], peer6)

	// Add the two peers back

	assert.True(pt.AddPeer(peer5))
	assert.True(pt.AddPeer(peer3))

	allPeers = pt.GetAllPeers(true)
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer4)
	assert.Equal((*allPeers)[3], peer6)
	assert.Equal((*allPeers)[4], peer5)
	assert.Equal((*allPeers)[5], peer3)
}

// --------------- Test Utilities --------------- //

func newTestEmptyPeerTable() PeerTable {
	pt := CreatePeerTable()
	return pt
}

func newSimulatedInboundPeer(netconn net.Conn, pubKey *crypto.PublicKey) *Peer {
	peerConfig := GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	inboundPeer, err := CreateInboundPeer(netconn, peerConfig, connConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create outbound peer: %v", err))
	}

	_, portStr, _ := net.SplitHostPort(netconn.LocalAddr().String())
	port, _ := strconv.ParseUint(portStr, 16, 16)
	inboundPeer.nodeInfo = p2ptypes.CreateNodeInfo(pubKey, uint16(port))
	return inboundPeer
}

func newIncomingNetconn(port int) net.Conn {
	go func() {
		netconn := p2ptypes.GetTestNetconn(port)
		defer netconn.Close()
	}()

	listener := p2ptypes.GetTestListener(port)
	netconn, err := listener.Accept()
	if err != nil {
		panic(fmt.Sprintf("Failed to listen to the netconn: %v", err))
	}
	defer netconn.Close()

	return netconn
}
