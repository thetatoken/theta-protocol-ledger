package discovery

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

func TestSeedPeerConnector(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr := "127.0.0.1:24561"
	peerBNetAddr := "127.0.0.1:24562"
	peerCNetAddr := "127.0.0.1:24563"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		//seedPeerNetAddressStrs := []string{peerBNetAddr, peerCNetAddr}
		seedPeerNetAddressStrs := []string{}
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
		//seedPeerNetAddressStrs := []string{peerANetAddr, peerCNetAddr}
		seedPeerNetAddressStrs := []string{}
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
		peerID := peer.ID()
		t.Logf("ID: %v", peer.ID())
		assert.True(peerID == peerAID || peerID == peerBID)
	}
}

// --------------- Test Utilities --------------- //

func newTestPeerDiscoveryManager(seedPeerNetAddressStrs []string, localNetworkAddress string) *PeerDiscoveryManager {
	peerPubKey := p2ptypes.GetTestRandPubKey()
	peerNodeInfo := p2ptypes.CreateNodeInfo(peerPubKey)
	addrbookPath := "./.addrbooks/addrbook_" + localNetworkAddress + ".json"
	routabilityRestrict := false
	selfNetAddressStr := "104.105.23.91:8888" // not useful for the test
	networkProtocol := "tcp"
	skipUPNP := true
	peerTable := peer.CreatePeerTable()
	config := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(&peerNodeInfo, addrbookPath, routabilityRestrict,
		selfNetAddressStr, seedPeerNetAddressStrs, networkProtocol, localNetworkAddress,
		skipUPNP, &peerTable, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	return discMgr
}
