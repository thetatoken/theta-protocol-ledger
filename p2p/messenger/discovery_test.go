package messenger

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/crypto"
	pr "github.com/thetatoken/theta/p2p/peer"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

func TestSeedPeerConnector(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	peerANetAddr := "127.0.0.1:24511"
	peerBNetAddr := "127.0.0.1:24512"
	peerCNetAddr := "127.0.0.1:24513"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
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
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
		t.Logf("[Peer B] ID: %v", peerID)
		peerBIDChan <- peerID
	}()
	peerBID := <-peerBIDChan

	// Simulate PeerC (i.e. us), proactively reaches out to the two seed peers
	seedPeerNetAddressStrs := []string{peerANetAddr, peerBNetAddr}
	localNetworkAddress := peerCNetAddr
	discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
	discMgr.Start(ctx)

	numPeers := len(seedPeerNetAddressStrs)
	for i := 0; i < numPeers; i++ {
		connected := <-discMgr.seedPeerConnector.Connected
		assert.True(connected)
	}
	allPeers := discMgr.peerTable.GetAllPeers(true)
	assert.Equal(2, len(*allPeers))
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
	ctx := context.Background()

	peerANetAddr := "127.0.0.1:24521"
	peerBNetAddr := "127.0.0.1:24522"
	peerCNetAddr := "127.0.0.1:24523"

	// Simulate PeerA
	peerAIDChan := make(chan string)
	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // proactively connect to PeerC (i.e. us)
		localNetworkAddress := peerANetAddr
		discMgr := newTestPeerDiscoveryManager(seedPeerNetAddressStrs, localNetworkAddress)
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
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
		discMgr.Start(ctx)

		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
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
			t.Logf("Inbound peer listener error: %v", err)
			inboundDetectedChan <- false
		}
	})
	discMgr.Start(ctx)

	numExpectedInboundPeers := 2
	for i := 0; i < numExpectedInboundPeers; i++ {
		inboundDetected := <-inboundDetectedChan
		assert.True(inboundDetected)
	}

	allPeers := discMgr.peerTable.GetAllPeers(true)
	t.Logf("---------------- All peers ----------------")
	for _, peer := range *allPeers {
		assert.False(peer.IsOutbound())
		peerID := peer.ID()
		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
		assert.True(peerID == peerAID || peerID == peerBID)
	}
}

// // FIXME: peer discovery is disabled while transitioning to new handshake protocol.
// func TestPeerDiscoveryMessageHandler(t *testing.T) {
// 	assert := assert.New(t)
// 	ctx := context.Background()

// 	peerANetAddr := "127.0.0.1:24531"
// 	peerBNetAddr := "127.0.0.1:24532"
// 	peerCNetAddr := "127.0.0.1:24533"
// 	peerA1NetAddr := "127.0.0.1:24534"
// 	peerA2NetAddr := "127.0.0.1:24535"
// 	peerB1NetAddr := "127.0.0.1:24536"
// 	peerB2NetAddr := "127.0.0.1:24537"
// 	peerA11NetAddr := "127.0.0.1:24538"
// 	peerA111NetAddr := "127.0.0.1:24539"
// 	peerA1111NetAddr := "127.0.0.1:24540"

// 	peerIds := make(map[string]bool)

// 	// Simulate PeerA
// 	peerAIDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{peerA1NetAddr, peerA2NetAddr}
// 		localNetworkAddress := peerANetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A] ID: %v", peerID)
// 		peerAIDChan <- peerID
// 	}()
// 	peerIds[<-peerAIDChan] = true

// 	// Simulate PeerA1
// 	peerA1IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{peerA11NetAddr}
// 		localNetworkAddress := peerA1NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A1] ID: %v", peerID)
// 		peerA1IDChan <- peerID
// 	}()
// 	peerIds[<-peerA1IDChan] = true

// 	// Simulate PeerA2
// 	peerA2IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{}
// 		localNetworkAddress := peerA2NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A2] ID: %v", peerID)
// 		peerA2IDChan <- peerID
// 	}()
// 	peerIds[<-peerA2IDChan] = true

// 	peerA11IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{peerA111NetAddr}
// 		localNetworkAddress := peerA11NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A11] ID: %v", peerID)
// 		peerA11IDChan <- peerID
// 	}()
// 	peerIds[<-peerA11IDChan] = true

// 	peerA111IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{peerA1111NetAddr}
// 		localNetworkAddress := peerA111NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A111] ID: %v", peerID)
// 		peerA111IDChan <- peerID
// 	}()
// 	peerIds[<-peerA111IDChan] = true

// 	peerA1111IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{}
// 		localNetworkAddress := peerA1111NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer A1111] ID: %v", peerID)
// 		peerA1111IDChan <- peerID
// 	}()
// 	peerIds[<-peerA1111IDChan] = true

// 	// Simulate PeerB
// 	peerBIDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{peerB1NetAddr, peerB2NetAddr}
// 		localNetworkAddress := peerBNetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer B] ID: %v", peerID)
// 		peerBIDChan <- peerID
// 	}()
// 	peerIds[<-peerBIDChan] = true

// 	// Simulate PeerB1
// 	peerB1IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{}
// 		localNetworkAddress := peerB1NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer B1] ID: %v", peerID)
// 		peerB1IDChan <- peerID
// 	}()
// 	peerIds[<-peerB1IDChan] = true

// 	// Simulate PeerB2
// 	peerB2IDChan := make(chan string)
// 	go func() {
// 		seedPeerNetAddressStrs := []string{}
// 		localNetworkAddress := peerB2NetAddr
// 		discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)
// 		discMgr.Start(ctx)

// 		peerID := discMgr.nodeInfo.PubKey.Address().Hex()
// 		t.Logf("[Peer B2] ID: %v", peerID)
// 		peerB2IDChan <- peerID
// 	}()
// 	peerIds[<-peerB2IDChan] = true

// 	// Simulate PeerC (i.e. us), proactively reaches out to the two seed peers
// 	seedPeerNetAddressStrs := []string{peerANetAddr, peerBNetAddr}
// 	localNetworkAddress := peerCNetAddr
// 	discMgr := newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs, localNetworkAddress)

// 	discDetectedChan := make(chan bool)
// 	discAddresses := map[string]bool{
// 		peerA1NetAddr:    false,
// 		peerA2NetAddr:    false,
// 		peerB1NetAddr:    false,
// 		peerB2NetAddr:    false,
// 		peerA11NetAddr:   false,
// 		peerA111NetAddr:  false,
// 		peerA1111NetAddr: false,
// 	}
// 	numDiscAddresses := len(discAddresses)

// 	discMgr.inboundPeerListener.SetInboundCallback(func(peer *pr.Peer, err error) {
// 		if err == nil {
// 			_, ok := discAddresses[peer.NetAddress().String()]
// 			if ok {
// 				t.Logf("Inbound peer added, ID: %v, from: %v", peer.ID(), peer.NetAddress())
// 				delete(discAddresses, peer.NetAddress().String())
// 				discDetectedChan <- true
// 			}
// 		} else {
// 			t.Logf("Inbound peer listener error: %v", err)
// 		}
// 	})
// 	discMgr.peerDiscMsgHandler.SetDiscoveryCallback(func(peer *pr.Peer, err error) {
// 		if err == nil {
// 			_, ok := discAddresses[peer.NetAddress().String()]
// 			if ok {
// 				t.Logf("Discovery peer added, ID: %v, from: %v", peer.ID(), peer.NetAddress())
// 				delete(discAddresses, peer.NetAddress().String())
// 				discDetectedChan <- true
// 			}
// 		} else {
// 			t.Logf("Failed to connect to discovery peer, error: %v", err)
// 		}
// 	})
// 	discMgr.Start(ctx)

// 	numPeers := len(seedPeerNetAddressStrs)
// 	for i := 0; i < numPeers; i++ {
// 		connected := <-discMgr.seedPeerConnector.Connected
// 		assert.True(connected)
// 	}

// 	for i := 0; i < numDiscAddresses; i++ {
// 		discDetected := <-discDetectedChan
// 		assert.True(discDetected)
// 	}

// 	allPeers := discMgr.peerTable.GetAllPeers(true)
// 	assert.Equal(numDiscAddresses+2, len(*allPeers))
// 	t.Logf("---------------- All peers ----------------")
// 	for _, peer := range *allPeers {
// 		peerID := peer.ID()
// 		t.Logf("ID: %v, isOutbound: %v", peer.ID(), peer.IsOutbound())
// 		_, ok := peerIds[peerID]
// 		if ok {
// 			delete(peerIds, peerID)
// 		}
// 	}
// 	assert.Empty(peerIds)
// }

// --------------- Test Utilities --------------- //

func newTestPeerDiscoveryManager(seedPeerNetAddressStrs []string, localNetworkAddress string) *PeerDiscoveryManager {
	messenger := (*Messenger)(nil) // not important for the test
	randPeerPrivKey, _, _ := crypto.GenerateKeyPair()
	_, portStr, _ := net.SplitHostPort(localNetworkAddress)
	port, _ := strconv.ParseUint(portStr, 16, 16)
	peerNodeInfo := p2ptypes.CreateLocalNodeInfo(randPeerPrivKey, uint16(port))
	addrbookPath := "./.addrbooks/addrbook_" + localNetworkAddress + ".json"
	routabilityRestrict := false
	networkProtocol := "tcp"
	skipUPNP := true
	peerTable := pr.CreatePeerTable()
	config := GetDefaultPeerDiscoveryManagerConfig()
	discMgr, err := CreatePeerDiscoveryManager(messenger, &peerNodeInfo, addrbookPath, routabilityRestrict,
		seedPeerNetAddressStrs, networkProtocol, localNetworkAddress, int(port),
		skipUPNP, &peerTable, config)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	return discMgr
}

func newTestPeerDiscoveryManagerAndMessenger(seedPeerNetAddressStrs []string, localNetworkAddress string) *PeerDiscoveryManager {
	_, portStr, err := net.SplitHostPort(localNetworkAddress)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	messenger := newTestMessenger(seedPeerNetAddressStrs, port)
	peerDiscoveryManager := messenger.discMgr
	peerDiscoveryManager.peerDiscMsgHandler.peerDiscoveryPulseInterval = 1 * time.Second

	return peerDiscoveryManager
}
