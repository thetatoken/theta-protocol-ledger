package messenger

import (
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/thetatoken/theta/p2p/netutil"
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

func (spc *SeedPeerConnector) connectToSeedPeers() {
	logger.Infof("Connecting to seed peers...")
	perm := rand.Perm(len(spc.seedPeerNetAddresses))
	for i := 0; i < len(perm); i++ { // create outbound peers in a random order
		spc.wg.Add(1)
		go func(i int) {
			defer spc.wg.Done()

			time.Sleep(time.Duration(rand.Int63n(3000)) * time.Millisecond)
			j := perm[i]
			peerNetAddress := spc.seedPeerNetAddresses[j]
			_, err := spc.discMgr.connectToOutboundPeer(&peerNetAddress, true)
			if err != nil {
				spc.Connected <- false
				logger.Errorf("Failed to connect to seed peer %v: %v", peerNetAddress.String(), err)
			} else {
				spc.Connected <- true
				logger.Infof("Successfully connected to seed peer %v", peerNetAddress.String())
			}
		}(i)
	}
}
