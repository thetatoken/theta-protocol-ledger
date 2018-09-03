package messenger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

func TestSeedPeerConnector(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr := "127.0.0.1:24511"
	peerBNetAddr := "127.0.0.1:24512"
	peerCNetAddr := "127.0.0.1:24513"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.OnStart()

		peerID := discMgr.nodeInfo.Address
		t.Logf("[Peer A] ID: %v", peerID)
		peerAIDChan <- peerID
	}()
	peerAID := <-peerAIDChan

	// Simulate PeerB
	peerBIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		localNetworkAddress := peerBNetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.OnStart()

		peerID := discMgr.nodeInfo.Address
		t.Logf("[Peer B] ID: %v", peerID)
		peerBIDChan <- peerID
	}()
	peerBID := <-peerBIDChan

	// Simulate PeerC (i.e. us), proactively reaches out to the two seed peers
	seedPeerNetAddressStrs := []string{peerANetAddr, peerBNetAddr}
	localNetworkAddress := peerCNetAddr
	discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
	discMgr.OnStart()

	numPeers := len(seedPeerNetAddressStrs)
	for i := 0; i < numPeers; i++ {
		connected := <-discMgr.seedPeerConnector.Connected
		assert.True(connected)
	}
	allPeers := discMgr.peerTable.GetAllPeers()
	t.Logf("---------------- All peers ----------------")
	for _, peer := range *allPeers {
		assert.True(peer.IsOutbound())
		peerID := peer.ID()
		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
		assert.True(peerID == peerAID || peerID == peerBID)
	}
}

func TestInboundPeerListener(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr := "127.0.0.1:24521"
	peerBNetAddr := "127.0.0.1:24522"
	peerCNetAddr := "127.0.0.1:24523"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // proactively connect to PeerC (i.e. us)
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.OnStart()

		peerID := discMgr.nodeInfo.Address
		t.Logf("[Peer A] ID: %v", peerID)
		peerAIDChan <- peerID
	}()
	peerAID := <-peerAIDChan

	// Simulate PeerB
	peerBIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // proactively connect to PeerC (i.e. us)
		localNetworkAddress := peerBNetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.OnStart()

		peerID := discMgr.nodeInfo.Address
		t.Logf("[Peer B] ID: %v", peerID)
		peerBIDChan <- peerID
	}()
	peerBID := <-peerBIDChan

	// Simulate PeerC (i.e. us)
	seedPeerNetAddressStrs := []string{} // passively listening to incoming connections
	localNetworkAddress := peerCNetAddr
	discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)

	inboundDetectedChan := make(chan bool)
	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			t.Logf("Inbound peer detected, ID: %v, from: %v", peer.ID(), peer.GetConnection().GetNetconn().RemoteAddr())
			inboundDetectedChan <- true
		} else {
			inboundDetectedChan <- false
		}
	})
	discMgr.OnStart()

	numExpectedInboundPeers := 2
	for i := 0; i < numExpectedInboundPeers; i++ {
		inboundDetected := <-inboundDetectedChan
		assert.True(inboundDetected)
	}

	allPeers := discMgr.peerTable.GetAllPeers()
	t.Logf("---------------- All peers ----------------")
	for _, peer := range *allPeers {
		assert.False(peer.IsOutbound())
		peerID := peer.ID()
		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
		assert.True(peerID == peerAID || peerID == peerBID)
	}
}

// --------------- Test Utilities --------------- //

func newTestPeerDiscoveryManager(seedPeerNetAddressStrs []string, localNetworkAddress string) *PeerDiscoveryManager {
	messenger := (*Messenger)(nil) // not important for the test
	peerPubKey := p2ptypes.GetTestRandPubKey()
	peerNodeInfo := p2ptypes.CreateNodeInfo(peerPubKey)
	addrbookPath := "./.addrbooks/addrbook_" + localNetworkAddress + ".json"
	routabilityRestrict := false
	selfNetAddressStr := "104.105.23.91:8888" // not important for the test
	networkProtocol := "tcp"
	skipUPNP := true
	peerTable := pr.CreatePeerTable()
	config := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(messenger, &peerNodeInfo, addrbookPath, routabilityRestrict,
		selfNetAddressStr, seedPeerNetAddressStrs, networkProtocol, localNetworkAddress,
		skipUPNP, &peerTable, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	return discMgr
}
