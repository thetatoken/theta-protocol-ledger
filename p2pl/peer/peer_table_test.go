package peer

import (
	"sync"
	"testing"

	pr "github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/require"
)

func TestPeerTableNilDBDoesNotPanic(t *testing.T) {
	pt := PeerTable{
		mutex:   &sync.Mutex{},
		peerMap: make(map[pr.ID]*Peer),
		db:      nil,
	}

	require.NotPanics(t, func() {
		_, _ = pt.RetrievePreviousPeers()
		pt.persistPeers()
		pt.writeToDB("key", "value")
	})
}

func TestInMemoryPeerTableDoesNotPersist(t *testing.T) {
	pt := CreateInMemoryPeerTable()

	previous, err := pt.RetrievePreviousPeers()
	require.NoError(t, err)
	require.Empty(t, previous)
	require.NotPanics(t, func() {
		pt.persistPeers()
		pt.writeToDB("key", "value")
	})
}
