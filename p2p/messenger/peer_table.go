package messenger

import (
	"sync"

	pr "github.com/thetatoken/ukulele/p2p/peer"
)

//
// PeerTable is a lookup table for peers
//
type PeerTable struct {
	mutex sync.Mutex

	peerMap map[string]*pr.Peer // map: peerKey |-> *Peer
	peers   []*pr.Peer          // For iteration with deterministic order
}

func (pt *PeerTable) addPeer(peer *pr.Peer) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	pt.peerMap[peer.Key()] = peer
	pt.peers = append(pt.peers, peer)
}

func (pt *PeerTable) deletePeer(peerKey string) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()

	if _, ok := pt.peerMap[peerKey]; !ok {
		return
	}

	delete(pt.peerMap, peerKey)
	for idx, peer := range pt.peers {
		if peer.Key() == peerKey {
			pt.peers = append(pt.peers[:idx], pt.peers[idx+1:]...)
		}
	}
}

func (pt *PeerTable) getPeer(peerKey string) *pr.Peer {
	peer, ok := pt.peerMap[peerKey]
	if !ok {
		return nil
	}
	return peer
}

func (pt *PeerTable) getAllPeers() *([]*pr.Peer) {
	return &pt.peers
}

func (pt *PeerTable) getTotalNumPeerss() uint {
	return uint(len(pt.peers))
}
