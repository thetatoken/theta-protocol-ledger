package types

import (
	"encoding/hex"
	"strings"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

//
// Message models the message sent/received through the P2P network
//
type Message struct {
	PeerID    string
	ChannelID common.ChannelIDEnum
	Content   interface{}
}

//
// NodeInfo provides the information of the corresponding blockchain node of the peer
//
type NodeInfo struct {
	PubKey  *crypto.PublicKey `rlp:"-"`
	Address string
}

// CreateNodeInfo creates an instance of NodeInfo
func CreateNodeInfo(pubKey *crypto.PublicKey) NodeInfo {
	nodeInfo := NodeInfo{
		PubKey:  pubKey,
		Address: calculateAddress(pubKey),
	}
	return nodeInfo
}

func calculateAddress(pubKey *crypto.PublicKey) string {
	addrBytes := pubKey.Address()
	address := strings.ToUpper(hex.EncodeToString(addrBytes[:]))
	return address
}

const (
	// PingSignal represents a ping signal to a peer
	PingSignal = byte(0x0)

	// PongSignal represents a pong respond to a peer
	PongSignal = byte(0x1)
)
