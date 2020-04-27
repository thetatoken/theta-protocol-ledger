package messenger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/p2p/netutil"
	pr "github.com/thetatoken/theta/p2p/peer"
	"github.com/thetatoken/theta/rlp"
)

func TestPeerEportUpdate(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr, _ := netutil.NewNetAddressString("127.0.0.1:55555")

	updatedEport := uint16(33333)
	peerANetAddr.Port = updatedEport
	pia := pr.PeerIDAddress{
		ID:   "peerA",
		Addr: peerANetAddr,
	}
	peerIDAddrs := []pr.PeerIDAddress{pia}
	t.Logf("Original address: %v", peerIDAddrs[0].Addr.String())

	message := PeerDiscoveryMessage{
		Type:      peerAddressesReplyType,
		Addresses: peerIDAddrs,
	}

	msgBytes, err := rlp.EncodeToBytes(message)
	assert.Equal(err, nil, err)

	var decodedMsg PeerDiscoveryMessage
	err = rlp.DecodeBytes(msgBytes, &decodedMsg)
	assert.Equal(err, nil, err)

	assert.Equal(updatedEport, decodedMsg.Addresses[0].Addr.Port)

	t.Logf("parsed port: %v", decodedMsg.Addresses[0].Addr.Port)
	t.Logf("parsed address: %v", decodedMsg.Addresses[0].Addr.String())
}
