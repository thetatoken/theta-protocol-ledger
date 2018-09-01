package peer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func newTestEmptyPeerTable() PeerTable {
	pt := CreatePeerTable()
	return pt
}

func TestDefaultPeerTableAddPeer(t *testing.T) {
	assert := assert.New(t)

	port := 37856
	netconn := newIncomingNetconn(port)

	pt := newTestEmptyPeerTable()
	assert.Equal(uint(0), pt.GetTotalNumPeers())

	randPubKey1 := getRandPubKey()
	randPubKey2 := getRandPubKey()

	peer1 := newInboundPeer(netconn, randPubKey1)
	success := pt.AddPeer(peer1)
	assert.True(success)
	assert.Equal(uint(1), pt.GetTotalNumPeers())
	assert.True(pt.PeerExists(peer1.ID()))
	assert.Equal(peer1, pt.GetPeer(peer1.ID()))

	peer1a := newInboundPeer(netconn, randPubKey1)
	success = pt.AddPeer(peer1a)
	assert.False(success) // cannot add two peers with the same PeerID
	assert.Equal(uint(1), pt.GetTotalNumPeers())

	peer2 := newInboundPeer(netconn, randPubKey2)
	success = pt.AddPeer(peer2)
	assert.True(success)
	assert.Equal(uint(2), pt.GetTotalNumPeers())
	assert.True(pt.PeerExists(peer2.ID()))
	assert.Equal(peer2, pt.GetPeer(peer2.ID()))
}

func TestDefaultPeerTableDeletePeer(t *testing.T) {
	assert := assert.New(t)

	pt := newTestEmptyPeerTable()
	assert.Equal(uint(0), pt.GetTotalNumPeers())

	port := 37857
	netconn := newIncomingNetconn(port)

	peer1 := newInboundPeer(netconn, getRandPubKey())
	peer2 := newInboundPeer(netconn, getRandPubKey())
	peer3 := newInboundPeer(netconn, getRandPubKey())

	assert.True(pt.AddPeer(peer1))
	assert.Equal(uint(1), pt.GetTotalNumPeers())
	assert.True(pt.AddPeer(peer2))
	assert.Equal(uint(2), pt.GetTotalNumPeers())
	assert.True(pt.AddPeer(peer3))
	assert.Equal(uint(3), pt.GetTotalNumPeers())

	assert.True(pt.PeerExists(peer1.ID()))
	assert.True(pt.PeerExists(peer2.ID()))
	assert.True(pt.PeerExists(peer3.ID()))

	pt.DeletePeer(peer1.ID())
	assert.False(pt.PeerExists(peer1.ID()))
	assert.Equal(uint(2), pt.GetTotalNumPeers())
	pt.DeletePeer(peer2.ID())
	assert.False(pt.PeerExists(peer2.ID()))
	assert.Equal(uint(1), pt.GetTotalNumPeers())
	pt.DeletePeer(peer3.ID())
	assert.False(pt.PeerExists(peer3.ID()))
	assert.Equal(uint(0), pt.GetTotalNumPeers())
}

func TestDefaultPeerIterationOrder(t *testing.T) {
	assert := assert.New(t)

	pt := newTestEmptyPeerTable()

	port := 37858
	netconn := newIncomingNetconn(port)

	peer1 := newInboundPeer(netconn, getRandPubKey())
	peer2 := newInboundPeer(netconn, getRandPubKey())
	peer3 := newInboundPeer(netconn, getRandPubKey())
	peer4 := newInboundPeer(netconn, getRandPubKey())
	peer5 := newInboundPeer(netconn, getRandPubKey())
	peer6 := newInboundPeer(netconn, getRandPubKey())

	assert.True(pt.AddPeer(peer1))
	assert.True(pt.AddPeer(peer2))
	assert.True(pt.AddPeer(peer3))
	assert.True(pt.AddPeer(peer4))
	assert.True(pt.AddPeer(peer5))
	assert.True(pt.AddPeer(peer6))

	allPeers := pt.GetAllPeers()
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer3)
	assert.Equal((*allPeers)[3], peer4)
	assert.Equal((*allPeers)[4], peer5)
	assert.Equal((*allPeers)[5], peer6)

	// Delete two peers

	pt.DeletePeer(peer3.ID())
	pt.DeletePeer(peer5.ID())

	allPeers = pt.GetAllPeers()
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer4)
	assert.Equal((*allPeers)[3], peer6)

	// Add the two peers back

	assert.True(pt.AddPeer(peer5))
	assert.True(pt.AddPeer(peer3))

	allPeers = pt.GetAllPeers()
	assert.Equal((*allPeers)[0], peer1)
	assert.Equal((*allPeers)[1], peer2)
	assert.Equal((*allPeers)[2], peer4)
	assert.Equal((*allPeers)[3], peer6)
	assert.Equal((*allPeers)[4], peer5)
	assert.Equal((*allPeers)[5], peer3)
}
