package messenger

import (
	"bytes"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/thetatoken/theta/p2p/netutil"
)

const (
	connectInterval               = 3000 // 3 sec
	lowConnectivityCheckInterval  = 1800
	highConnectivityCheckInterval = 6
)

//
// SeedPeerConnector proactively connects to seed peers
//
type SeedPeerConnector struct {
	discMgr *PeerDiscoveryManager

	selfNetAddress       netutil.NetAddress
	seedPeerNetAddresses []netutil.NetAddress

	Connected chan bool

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// createSeedPeerConnector creates an instance of the SeedPeerConnector
func createSeedPeerConnector(discMgr *PeerDiscoveryManager,
	selfNetAddressStr string, seedPeerNetAddressStrs []string) (SeedPeerConnector, error) {
	numSeedPeers := len(seedPeerNetAddressStrs)
	spc := SeedPeerConnector{
		discMgr:   discMgr,
		Connected: make(chan bool, numSeedPeers),
		wg:        &sync.WaitGroup{},
	}

	selfNetAddress, err := netutil.NewNetAddressString(selfNetAddressStr)
	if err != nil {
		logger.Errorf("Failed to parse the self network address: %v", selfNetAddressStr)
		return spc, err
	}
	spc.selfNetAddress = *selfNetAddress

	for _, seedPeerNetAddressStr := range seedPeerNetAddressStrs {
		seedNetAddress, err := netutil.NewNetAddressString(seedPeerNetAddressStr)
		if err != nil {
			logger.Errorf("Failed to parse the seed network address: %v", seedPeerNetAddressStr)
			return spc, err
		}
		if seedNetAddress.Equals(selfNetAddress) {
			continue
		}
		spc.seedPeerNetAddresses = append(spc.seedPeerNetAddresses, *seedNetAddress)
	}

	return spc, nil
}

// Start is called when the SeedPeerConnector starts
func (spc *SeedPeerConnector) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	spc.ctx = c
	spc.cancel = cancel

	spc.connectToSeedPeers()
	go spc.maintainConnectivityRoutine()
	return nil
}

// Stop is called when the SeedPeerConnector stops
func (spc *SeedPeerConnector) Stop() {
	spc.cancel()
}

// Wait suspends the caller goroutine
func (spc *SeedPeerConnector) Wait() {
	spc.wg.Wait()
}

func (spc *SeedPeerConnector) isASeedPeerIgnoringPort(netAddr *netutil.NetAddress) bool {
	for _, seedAddr := range spc.seedPeerNetAddresses {
		if bytes.Compare(netAddr.IP, seedAddr.IP) == 0 {
			return true
		}
	}
	return false
}

func (spc *SeedPeerConnector) isASeedPeer(netAddr *netutil.NetAddress) bool {
	for _, seedAddr := range spc.seedPeerNetAddresses {
		if netAddr.Equals(&seedAddr) {
			return true
		}
	}
	return false
}

func (spc *SeedPeerConnector) connectToSeedPeers() {
	logger.Infof("Connecting to seed and persisted peers...")

	var peerNetAddresses []netutil.NetAddress
	// add seed peers first
	peerNetAddresses = append(peerNetAddresses, spc.seedPeerNetAddresses...)
	// add persisted peers
	persistedPeerAddrs, err := spc.discMgr.peerTable.RetrievePreviousPeers()
	if err == nil {
		for _, addr := range persistedPeerAddrs {
			if !spc.isASeedPeer(addr) {
				peerNetAddresses = append(peerNetAddresses, *addr)
			}
		}
	}

	perm := rand.Perm(len(peerNetAddresses))
	for i := 0; i < len(perm); i++ { // create outbound peers in a random order
		spc.wg.Add(1)
		go func(i int) {
			defer spc.wg.Done()

			time.Sleep(time.Duration(rand.Int63n(connectInterval)) * time.Millisecond)
			j := perm[i]
			peerNetAddress := peerNetAddresses[j]
			_, err := spc.discMgr.connectToOutboundPeer(&peerNetAddress, true)
			if err != nil {
				spc.Connected <- false
				logger.Warnf("Failed to connect to seed peer %v: %v", peerNetAddress.String(), err)
			} else {
				spc.Connected <- true
				logger.Infof("Successfully connected to seed peer %v", peerNetAddress.String())
			}
		}(i)
	}
}

func (spc *SeedPeerConnector) maintainConnectivityRoutine() {
	var seedsConnectivityCheckPulse *time.Ticker
	if spc.discMgr.seedPeerOnly {
		seedsConnectivityCheckPulse = time.NewTicker(highConnectivityCheckInterval * time.Second)
	} else {
		seedsConnectivityCheckPulse = time.NewTicker(lowConnectivityCheckInterval * time.Second)
	}

	for {
		select {
		case <-seedsConnectivityCheckPulse.C:
			spc.maintainConnectivity()
		}
	}
}

func (spc *SeedPeerConnector) maintainConnectivity() {
	allPeers := *(spc.discMgr.peerTable.GetAllPeers(true)) // not to count edge node peers
	if !spc.discMgr.seedPeerOnly {
		for _, pr := range allPeers {
			if pr.IsSeed() {
				// don't proceed if there's at least one seed in peer table
				return
			}
		}
	}

	perm := rand.Perm(len(spc.seedPeerNetAddresses))
	for i := 0; i < len(perm); i++ { // random order
		spc.wg.Add(1)
		go func(i int) {
			defer spc.wg.Done()

			time.Sleep(time.Duration(rand.Int63n(connectInterval)) * time.Millisecond)
			j := perm[i]
			peerNetAddress := spc.seedPeerNetAddresses[j]
			if !spc.discMgr.peerTable.PeerAddrExists(&peerNetAddress) {
				_, err := spc.discMgr.connectToOutboundPeer(&peerNetAddress, true)
				if err != nil {
					spc.Connected <- false
					logger.Warnf("Failed to connect to seed peer %v: %v", peerNetAddress.String(), err)
				} else {
					spc.Connected <- true
					logger.Infof("Successfully connected to seed peer %v", peerNetAddress.String())
				}
			}
		}(i)

		if !spc.discMgr.seedPeerOnly {
			break // if not seed peer only, sufficient to have at least one connection
		}
	}
}
