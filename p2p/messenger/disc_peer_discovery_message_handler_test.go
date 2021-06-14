// +build integration

package messenger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	pr "github.com/thetatoken/theta/p2p/peer"
)

func TestPeerDiscoveryMessageHandler(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr := "127.0.0.1:24531"
	peerBNetAddr := "127.0.0.1:24532"
	peerCNetAddr := "127.0.0.1:24533"
	peerA1NetAddr := "127.0.0.1:24534"
	peerA2NetAddr := "127.0.0.1:24535"
	peerB1NetAddr := "127.0.0.1:24536"
	peerB2NetAddr := "127.0.0.1:24537"
	peerA11NetAddr := "127.0.0.1:24538"
	peerA21NetAddr := "127.0.0.1:24539"
	peerA111NetAddr := "127.0.0.1:24540"
	peerA112NetAddr := "127.0.0.1:24541"
	peerA113NetAddr := "127.0.0.1:24542"
	peerA114NetAddr := "127.0.0.1:24543"
	peerA115NetAddr := "127.0.0.1:24544"
	peerA116NetAddr := "127.0.0.1:24545"
	peerA117NetAddr := "127.0.0.1:24546"
	peerA118NetAddr := "127.0.0.1:24547"
	peerA119NetAddr := "127.0.0.1:24548"
	peerA120NetAddr := "127.0.0.1:24549"
	peerA1111NetAddr := "127.0.0.1:24550"

	peerIds := make(map[string]bool)

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerA1NetAddr, peerA2NetAddr}
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A] ID: %v", peerID)
		peerAIDChan <- peerID
	}()
	peerIds[<-peerAIDChan] = true

	// Simulate PeerA1
	peerA1IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerA11NetAddr}
		localNetworkAddress := peerA1NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A1] ID: %v", peerID)
		peerA1IDChan <- peerID
	}()
	peerIds[<-peerA1IDChan] = true

	// Simulate PeerA2
	peerA2IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerA21NetAddr}
		localNetworkAddress := peerA2NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A2] ID: %v", peerID)
		peerA2IDChan <- peerID
	}()
	peerIds[<-peerA2IDChan] = true

	peerA11IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerA111NetAddr, peerA112NetAddr, peerA113NetAddr, peerA114NetAddr, peerA115NetAddr, peerA116NetAddr, peerA117NetAddr, peerA118NetAddr, peerA119NetAddr, peerA120NetAddr}
		// seedPeerNetAddressStrs := []string{peerA111NetAddr, peerA112NetAddr, peerA113NetAddr, peerA114NetAddr, peerA115NetAddr}
		localNetworkAddress := peerA11NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A11] ID: %v", peerID)
		peerA11IDChan <- peerID
	}()
	peerIds[<-peerA11IDChan] = true

	peerA21IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA21NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A21] ID: %v", peerID)
		peerA21IDChan <- peerID
	}()
	peerIds[<-peerA21IDChan] = true

	// Simulate PeerB
	peerBIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerB1NetAddr, peerB2NetAddr}
		localNetworkAddress := peerBNetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer B] ID: %v", peerID)
		peerBIDChan <- peerID
	}()
	peerIds[<-peerBIDChan] = true

	// Simulate PeerB1
	peerB1IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerB1NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer B1] ID: %v", peerID)
		peerB1IDChan <- peerID
	}()
	peerIds[<-peerB1IDChan] = true

	// Simulate PeerB2
	peerB2IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerB2NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer B2] ID: %v", peerID)
		peerB2IDChan <- peerID
	}()
	peerIds[<-peerB2IDChan] = true

	peerA111IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA111NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A111] ID: %v", peerID)
		peerA111IDChan <- peerID
	}()
	peerIds[<-peerA111IDChan] = true

	peerA112IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA112NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A112] ID: %v", peerID)
		peerA112IDChan <- peerID
	}()
	peerIds[<-peerA112IDChan] = true

	peerA113IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA113NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A113] ID: %v", peerID)
		peerA113IDChan <- peerID
	}()
	peerIds[<-peerA113IDChan] = true

	peerA114IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA114NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A114] ID: %v", peerID)
		peerA114IDChan <- peerID
	}()
	peerIds[<-peerA114IDChan] = true

	peerA115IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA115NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A115] ID: %v", peerID)
		peerA115IDChan <- peerID
	}()
	peerIds[<-peerA115IDChan] = true

	peerA116IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA116NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A116] ID: %v", peerID)
		peerA116IDChan <- peerID
	}()
	peerIds[<-peerA116IDChan] = true

	peerA117IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA117NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A117] ID: %v", peerID)
		peerA117IDChan <- peerID
	}()
	peerIds[<-peerA117IDChan] = true

	peerA118IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA118NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A118] ID: %v", peerID)
		peerA118IDChan <- peerID
	}()
	peerIds[<-peerA118IDChan] = true

	peerA119IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA119NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A119] ID: %v", peerID)
		peerA119IDChan <- peerID
	}()
	peerIds[<-peerA119IDChan] = true

	peerA120IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA120NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A120] ID: %v", peerID)
		peerA120IDChan <- peerID
	}()
	peerIds[<-peerA120IDChan] = true

	peerA1111IDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{}
		localNetworkAddress := peerA1111NetAddr
		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start()

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer A1111] ID: %v", peerID)
		peerA1111IDChan <- peerID
	}()
	peerIds[<-peerA1111IDChan] = true

	// Simulate PeerC (i.e. us), proactively reaches out to the two seed peers
	seedPeerNetAddressStrs := []string{peerANetAddr, peerBNetAddr}
	localNetworkAddress := peerCNetAddr
	discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)

	discDetectedChan := make(chan bool)
	discAddresses := map[string]bool{
		peerA1NetAddr:    false,
		peerA2NetAddr:    false,
		peerB1NetAddr:    false,
		peerB2NetAddr:    false,
		peerA11NetAddr:   false,
		peerA21NetAddr:   false,
		peerA111NetAddr:  false,
		peerA112NetAddr:  false,
		peerA113NetAddr:  false,
		peerA114NetAddr:  false,
		peerA115NetAddr:  false,
		peerA116NetAddr:  false,
		peerA117NetAddr:  false,
		peerA118NetAddr:  false,
		peerA119NetAddr:  false,
		peerA120NetAddr:  false,
		peerA1111NetAddr: false,
	}
	numDiscAddresses := len(discAddresses)

	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			_, ok := discAddresses[peer.NetAddress().String()]
			if ok {
				t.Logf("Inbound peer added, ID: %v, from: %v", peer.ID(), peer.NetAddress())
				delete(discAddresses, peer.NetAddress().String())
				discDetectedChan <- true
			}
		} else {
			t.Logf("Inbound peer listener error: %v", err)
		}
	})
	discMgr.peerDiscMsgHandler.SetDiscoveryCallback(func(peer *pr.Peer, err error) {
		if err == nil {
			_, ok := discAddresses[peer.NetAddress().String()]
			if ok {
				t.Logf("Discovery peer added, ID: %v, from: %v", peer.ID(), peer.NetAddress())
				delete(discAddresses, peer.NetAddress().String())
				discDetectedChan <- true
			}
		} else {
			t.Logf("failed to Discovery peer added, ID: %v, from: %v", peer.ID(), peer.NetAddress())
		}
	})
	discMgr.Start()

	numPeers := len(seedPeerNetAddressStrs)
	for i := 0; i < numPeers; i++ {
		connected := <-discMgr.seedPeerConnector.Connected
		assert.True(connected)
	}

	for i := 0; i < numDiscAddresses; i++ {
		discDetected := <-discDetectedChan
		assert.True(discDetected)
	}

	allPeers := discMgr.peerTable.GetAllPeers(true)
	assert.Equal(numDiscAddresses+2, len(*allPeers))
	t.Logf("---------------- All peers ----------------")
	for _, peer := range *allPeers {
		assert.True(peer.IsOutbound())
		peerID := peer.ID()
		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
		_, ok := peerIds[peerID]
		if ok {
			delete(peerIds, peerID)
		}
	}
	assert.Empty(peerIds)
}
