package discovery

import (
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/p2p/netutil"
)

//
// SeedPeerConnector proactively connects to seed peers
//
type SeedPeerConnector struct {
	discMgr *PeerDiscoveryManager

	selfNetAddress       netutil.NetAddress
	seedPeerNetAddresses []netutil.NetAddress
}

// createSeedPeerConnector creates an instance of the SeedPeerConnector
func createSeedPeerConnector(discMgr *PeerDiscoveryManager,
	selfNetAddressStr string, seedPeerNetAddressStrs []string) (SeedPeerConnector, error) {
	spc := SeedPeerConnector{
		discMgr: discMgr,
	}

	selfNetAddress, err := netutil.NewNetAddressString(selfNetAddressStr)
	if err != nil {
		log.Errorf("[p2p] Failed to parse the self network address: %v", selfNetAddressStr)
		return spc, err
	}
	spc.selfNetAddress = *selfNetAddress

	for _, seedPeerNetAddressStr := range seedPeerNetAddressStrs {
		seedNetAddress, err := netutil.NewNetAddressString(seedPeerNetAddressStr)
		if err != nil {
			log.Errorf("[p2p] Failed to parse the seed network address: %v", seedPeerNetAddressStr)
			return spc, err
		}
		if seedNetAddress.Equals(selfNetAddress) {
			continue
		}
		spc.seedPeerNetAddresses = append(spc.seedPeerNetAddresses, *seedNetAddress)
		spc.discMgr.addrBook.AddAddress(seedNetAddress, selfNetAddress)
	}

	spc.discMgr.addrBook.Save()

	return spc, nil
}

// OnStart is called when the SeedPeerConnector starts
func (spc *SeedPeerConnector) OnStart() error {
	spc.connectToSeedPeers()
	return nil
}

// OnStop is called when the SeedPeerConnector stops
func (spc *SeedPeerConnector) OnStop() {
}

func (spc *SeedPeerConnector) connectToSeedPeers() {
	perm := rand.Perm(len(spc.seedPeerNetAddresses))
	for i := 0; i < len(perm); i++ { // create outbound peers in a random order
		go func(i int) {
			time.Sleep(time.Duration(rand.Int63n(3000)) * time.Millisecond)
			j := perm[i]
			spc.discMgr.connectToOutboundPeer(&spc.seedPeerNetAddresses[j], true)
		}(i)
	}
}
