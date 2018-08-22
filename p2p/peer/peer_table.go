package peer

import (
	"sync"
)

//
// PeerTable is a lookup table for peers
//
type PeerTable struct {
	mutex sync.Mutex

	peerMap map[string]*Peer // map: peerID |-> *Peer
	peers   []*Peer          // For iteration with deterministic order
}

// AddPeer adds the given peer to the PeerTable
func (pt *PeerTable) AddPeer(peer *Peer) bool {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if pt.PeerExists(peer.ID()) {
		return false
	}

	pt.peerMap[peer.ID()] = peer
	pt.peers = append(pt.peers, peer)

	return true
}

// DeletePeer deletes the given peer from the PeerTable
func (pt *PeerTable) DeletePeer(peerID string) {
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
}

// GetPeer returns the peer for the given peerID (if exists)
func (pt *PeerTable) GetPeer(peerID string) *Peer {
	peer, exists := pt.peerMap[peerID]
	if !exists {
		return nil
	}
	return peer
}

// PeerExists indicates whether the PeerTable has a peer for the given peerID
func (pt *PeerTable) PeerExists(peerID string) bool {
	_, exists := pt.peerMap[peerID]
	return exists
}

// GetAllPeers returns all the peers
func (pt *PeerTable) GetAllPeers() *([]*Peer) {
	return &pt.peers
}

// GetTotalNumPeers returns the total number of peers in the PeerTable
func (pt *PeerTable) GetTotalNumPeers() uint {
	return uint(len(pt.peers))
}
