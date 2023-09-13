// +build integration

package messenger

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	pr "github.com/thetatoken/theta/p2p/peer"
)

func TestPeerFailureHandling(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	peerANetAddr := "127.0.0.1:24511"
	peerBNetAddr := "127.0.0.1:24512"
	peerCNetAddr := "127.0.0.1:24513"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	peerADMChan := make(chan *PeerDiscoveryManager)
	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A] ID: %v", peerID)
		peerAIDChan <- peerID
		peerADMChan <- discMgr
	}()

	// Simulate PeerB
	peerBIDChan := make(chan string)
	peerBDMChan := make(chan *PeerDiscoveryManager)
	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // proactively connect to PeerC (i.e. us)
		localNetworkAddress := peerBNetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer B] ID: %v", peerID)
		peerBIDChan <- peerID
		peerBDMChan <- discMgr
	}()

	peerAID := <-peerAIDChan
	peerADM := <-peerADMChan
	peerBID := <-peerBIDChan
	peerBDM := <-peerBDMChan

	// Simulate PeerC (i.e. us)
	seedPeerNetAddressStrs := []string{peerANetAddr}
	localNetworkAddress := peerCNetAddr
	discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
	peerCID := discMgr.nodeInfo.PubKey.Address().Hex()

	inboundDetectedChan := make(chan bool)
	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			t.Logf("Inbound peer detected, ID: %v, from: %v", peer.ID(), peer.GetConnection().GetNetconn().RemoteAddr())
			inboundDetectedChan <- true
		} else {
			t.Logf("Inbound peer listener error: %v", err)
			inboundDetectedChan <- false
		}
	})

	discMgr.Start(ctx)

	numPeers := len(seedPeerNetAddressStrs)
	for i := 0; i < numPeers; i++ {
		connected := <-discMgr.seedPeerConnector.Connected
		assert.True(connected)
	}

	numExpectedInboundPeers := 1
	for i := 0; i < numExpectedInboundPeers; i++ {
		inboundDetected := <-inboundDetectedChan
		assert.True(inboundDetected)
	}

	allPeers := discMgr.peerTable.GetAllPeers(true)
	assert.Equal(2, len(*allPeers))

	t.Logf("---------------- All peers ----------------")
	for _, peer := range *allPeers {
		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
		assert.True(peer.ID() == peerAID || peer.ID() == peerBID)
		peer.SetPersistency(false)
		peer.GetConnection().SetPingTimer(1)
	}

	peerA := peerADM.peerTable.GetPeer(peerCID)
	peerA.GetConnection().SetPingTimer(1)

	peerB := peerBDM.peerTable.GetPeer(peerCID)
	peerB.GetConnection().SetPingTimer(1)

	peerA.CancelConnection()
	peerB.CancelConnection()

	time.Sleep(time.Second * 10)

	assert.Equal(uint(0), discMgr.peerTable.GetTotalNumPeers(true))
}
