package peer

import (
	"fmt"
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/thetatoken/theta/common"
	mm "github.com/thetatoken/theta/common/math"
	nu "github.com/thetatoken/theta/p2p/netutil"

	"github.com/spf13/viper"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

const (
	// % of total peers known returned by GetSelection.
	getSelectionPercent = 23

	// min peers that must be returned by GetSelection. Useful for bootstrapping.
	minGetSelection = 32

	// max peers returned by GetSelection
	maxGetSelection = 250

	dbKey = "p2pPeer"
)

//
// PeerTable is a lookup table for peers
//
type PeerTable struct {
	mutex *sync.Mutex

	peerMap map[string]*Peer // map: peerID |-> *Peer
	peers   []*Peer          // For iteration with deterministic order
	addrMap map[string]*Peer

	db *leveldb.DB // peerTable persistence for restart
}

type PeerIDAddress struct {
	ID   string
	Addr *nu.NetAddress
}

// CreatePeerTable creates an instance of the PeerTable
func CreatePeerTable() PeerTable {
	cfgPath := filepath.Dir(viper.ConfigFileUsed())
	dbPath := path.Join(cfgPath, "db", "peer_table")

	db, err := leveldb.OpenFile(dbPath, &opt.Options{
		OpenFilesCacheCapacity: 0,
		BlockCacheCapacity:     256 / 2 * opt.MiB,
		WriteBuffer:            256 / 4 * opt.MiB, // Two of these are used internally
		Filter:                 filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(dbPath, nil)
	}
	// (Re)check for errors and abort if opening of the db failed
	if err != nil {
		logger.Errorf("Failed to create db for peer table, %v", err)
	}

	return PeerTable{
		mutex:   &sync.Mutex{},
		peerMap: make(map[string]*Peer),
		addrMap: make(map[string]*Peer),
		db:      db,
	}
}

// AddPeer adds the given peer to the PeerTable
func (pt *PeerTable) AddPeer(peer *Peer) bool {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	_, exists := pt.peerMap[peer.ID()]
	if exists {
		// Update existing entry with same ID.
		for i, p := range pt.peers {
			if p.ID() == peer.ID() {
				p.Stop()
				logger.Warnf("Stopping duplicated peer: %v", p.ID())
				pt.peers[i] = peer
				break
			}
		}
	} else {
		pt.peers = append(pt.peers, peer)
	}

	pt.peerMap[peer.ID()] = peer
	pt.addrMap[peer.NetAddress().String()] = peer

	pt.persistPeers()

	return true
}

// DeletePeer deletes the given peer from the PeerTable
func (pt *PeerTable) DeletePeer(peerID string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	var peer *Peer
	var ok bool
	if peer, ok = pt.peerMap[peerID]; !ok {
		return
	}

	delete(pt.peerMap, peerID)
	delete(pt.addrMap, peer.NetAddress().String())
	for idx, peer := range pt.peers {
		if peer.ID() == peerID {
			pt.peers = append(pt.peers[:idx], pt.peers[idx+1:]...) // not to break in case there are multiple matches
		}
	}

	logger.Infof("Deleted peer %v from the peer table", peerID)

	pt.persistPeers()
}

// PurgeOldestPeer purges the oldest peer from the PeerTable
func (pt *PeerTable) PurgeOldestPeer() *Peer {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	var peer *Peer
	var idx int
	for idx, peer = range pt.peers {
		if !peer.IsSeed() {
			break
		}
	}
	if peer != nil {
		delete(pt.peerMap, peer.ID())
		delete(pt.addrMap, peer.NetAddress().String())
		pt.peers = append(pt.peers[:idx], pt.peers[idx+1:]...)
	}

	logger.Infof("Purged the oldest peer %v from the peer table, idx: %v", peer.ID(), idx)

	pt.persistPeers()
	return peer
}

// GetPeer returns the peer for the given peerID (if exists)
func (pt *PeerTable) GetPeer(peerID string) *Peer {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	peer, exists := pt.peerMap[peerID]
	if !exists {
		return nil
	}
	return peer
}

// GetPeerWithAddr returns the peer for the given address (if exists)
func (pt *PeerTable) GetPeerWithAddr(addr *nu.NetAddress) *Peer {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	peer, exists := pt.addrMap[addr.String()]
	if !exists {
		return nil
	}
	return peer
}

// PeerExists indicates whether the PeerTable has a peer for the given peerID
func (pt *PeerTable) PeerExists(peerID string) bool {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	_, exists := pt.peerMap[peerID]
	return exists
}

// PeerAddrExists indicates whether the PeerTable has a peer for the given address
func (pt *PeerTable) PeerAddrExists(addr *nu.NetAddress) bool {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	_, exists := pt.addrMap[addr.String()]
	return exists
}

// GetAllPeers returns all the peers
func (pt *PeerTable) GetAllPeers(skipEdgeNode bool) *([]*Peer) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	//ret := make([]*Peer, len(pt.peers))
	var ret []*Peer
	for _, p := range pt.peers {
		if skipEdgeNode && p.NodeType() == common.NodeTypeEdgeNode {
			continue
		}
		//ret[i] = p
		ret = append(ret, p)
	}
	return &ret
}

// GetSelection randomly selects some peers. Suitable for peer-exchange protocols.
func (pt *PeerTable) GetSelection(skipEdgeNode bool) (peerIDAddrs []PeerIDAddress) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if len(pt.peers) == 0 {
		return nil
	}

	peers := make([]*Peer, len(pt.peers))
	copy(peers, pt.peers)

	numPeers := mm.MaxInt(
		mm.MinInt(minGetSelection, len(peers)),
		len(peers)*getSelectionPercent/100)
	numPeers = mm.MinInt(maxGetSelection, numPeers)

	// Fisher-Yates shuffle the array. We only need to do the first
	// `numPeers' since we are throwing the rest.
	for i := 0; i < numPeers; i++ {
		// pick a number between current index and the end
		j := rand.Intn(len(peers)-i) + i
		peers[i], peers[j] = peers[j], peers[i]
	}

	// slice off the limit we are willing to share.
	peers = peers[:numPeers]
	for _, peer := range peers {
		if skipEdgeNode && peer.NodeType() == common.NodeTypeEdgeNode {
			continue
		}
		peerIDAddr := PeerIDAddress{
			ID:   peer.ID(),
			Addr: peer.netAddress,
		}
		peerIDAddrs = append(peerIDAddrs, peerIDAddr)
	}
	return
}

// GetTotalNumPeers returns the total number of peers in the PeerTable
func (pt *PeerTable) GetTotalNumPeers(skipEdgeNode bool) uint {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	var numPeers uint
	if !skipEdgeNode {
		numPeers = uint(len(pt.peers))
	} else {
		for _, peer := range pt.peers {
			if peer.NodeType() == common.NodeTypeEdgeNode {
				continue
			}
			numPeers = numPeers + 1
		}
	}

	return numPeers
}

func (pt *PeerTable) RetrievePreviousPeers() ([]*nu.NetAddress, error) {
	if pt.db == nil {
		return []*nu.NetAddress{}, fmt.Errorf("peerTable DB not ready yet")
	}

	dat, err := pt.db.Get([]byte(dbKey), nil)
	if err != nil {
		logger.Warnf("Failed to retrieve previously persisted peers")
		return nil, err
	}
	addrs := strings.Split(string(dat), "|")
	return nu.NewNetAddressStrings(addrs)
}

func (pt *PeerTable) persistPeers() {
	maxPeerPersistence := viper.GetInt(common.CfgMaxNumPersistentPeers)
	numPeers := len(pt.peers)
	numInDB := numPeers
	if numPeers > maxPeerPersistence {
		numInDB = maxPeerPersistence
	}

	peerAddrs := make([]string, numInDB)
	dbPeers := pt.peers[numPeers-numInDB:]
	for i, p := range dbPeers {
		peerAddrs[i] = p.NetAddress().String()
	}
	go pt.writeToDB(dbKey, strings.Join(peerAddrs, "|"))
}

func (pt *PeerTable) writeToDB(key, value string) {
	if pt.db != nil {
		pt.db.Put([]byte(key), []byte(value), nil)
	}
}
