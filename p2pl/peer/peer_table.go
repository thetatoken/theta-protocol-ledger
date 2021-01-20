package peer

import (
	"math/rand"
	"path"
	"path/filepath"
	"strings"
	"sync"

	pr "github.com/libp2p/go-libp2p-core/peer"
	"github.com/thetatoken/theta/common"
	mm "github.com/thetatoken/theta/common/math"

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

	dbKey = "peers"
)

//
// PeerTable is a lookup table for peers
//
type PeerTable struct {
	mutex *sync.Mutex

	peerMap map[pr.ID]*Peer // map: peerID |-> *Peer
	peers   []*Peer         // For iteration with deterministic order

	db *leveldb.DB // peerTable persistence for restart
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
		peerMap: make(map[pr.ID]*Peer),
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

	pt.persistPeers()

	return true
}

// DeletePeer deletes the given peer from the PeerTable
func (pt *PeerTable) DeletePeer(peerID pr.ID) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if _, ok := pt.peerMap[peerID]; !ok {
		return
	}

	delete(pt.peerMap, peerID)
	for idx, peer := range pt.peers {
		if peer.ID() == peerID {
			pt.peers = append(pt.peers[:idx], pt.peers[idx+1:]...)
		}
	}

	pt.persistPeers()
}

// GetPeer returns the peer for the given peerID (if exists)
func (pt *PeerTable) GetPeer(peerID pr.ID) *Peer {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	peer, exists := pt.peerMap[peerID]
	if !exists {
		return nil
	}
	return peer
}

// PeerExists indicates whether the PeerTable has a peer for the given peerID
func (pt *PeerTable) PeerExists(peerID pr.ID) bool {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	_, exists := pt.peerMap[peerID]
	return exists
}

// GetAllPeers returns all the peers
func (pt *PeerTable) GetAllPeers(skipEdgeNode bool) *([]*Peer) {
	// TODO: support skipEdgeNode
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	ret := make([]*Peer, len(pt.peers))
	for i, p := range pt.peers {
		ret[i] = p
	}
	return &ret
}

// GetAllPeers returns all the peers
func (pt *PeerTable) GetAllPeerIDs() *[]pr.ID {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	ret := make([]pr.ID, len(pt.peers))
	for i, p := range pt.peers {
		ret[i] = p.ID()
	}
	return &ret
}

func (pt *PeerTable) RetrievePreviousPeers() (res []*pr.AddrInfo, err error) {
	dat, err := pt.db.Get([]byte(dbKey), nil)
	if err != nil {
		logger.Warnf("Failed to retrieve previously persisted peers")
		return
	}
	arr := strings.Split(string(dat), "|")
	for _, json := range arr {
		var addrInfo pr.AddrInfo
		err = addrInfo.UnmarshalJSON([]byte(json))
		if err != nil {
			logger.Warnf("Failed to unmarshal peer addr info, %v", json)
			break
		}
		res = append(res, &addrInfo)
	}
	return
}

func (pt *PeerTable) persistPeers() {
	maxPeerPersistence := viper.GetInt(common.CfgMaxNumPersistentPeers)
	numPeers := len(pt.peers)
	numInDB := numPeers
	if numPeers > maxPeerPersistence {
		numInDB = maxPeerPersistence
	}

	peerAddrInfos := make([]string, numInDB)
	dbPeers := pt.peers[numPeers-numInDB:]
	for i, p := range dbPeers {
		json, err := p.addrInfo.MarshalJSON()
		if err == nil {
			peerAddrInfos[i] = string(json)
		}
	}
	go pt.writeToDB(dbKey, strings.Join(peerAddrInfos, "|"))
}

func (pt *PeerTable) writeToDB(key, value string) {
	pt.db.Put([]byte(key), []byte(value), nil)
}

// GetSelection randomly selects some peers. Suitable for peer-exchange protocols.
func (pt *PeerTable) GetSelection() (peerIDAddrs []pr.ID) {
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
		// peerIDAddr := PeerIDAddress{
		// 	ID:   peer.ID(),
		// 	Addr: peer.netAddress,
		// }
		peerIDAddrs = append(peerIDAddrs, peer.ID())
	}
	return
}

// GetTotalNumPeers returns the total number of peers in the PeerTable
func (pt *PeerTable) GetTotalNumPeers(skipEdgeNode bool) uint {
	// TODO: support skipEdgeNode
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	return uint(len(pt.peers))
}
