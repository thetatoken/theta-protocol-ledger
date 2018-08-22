package types

import (
	"crypto/ecdsa"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

//
// Message models the message sent/received through the P2P network
//
type Message struct {
	ChannelID common.ChannelIDEnum
	Content   common.Bytes
}

//
// NodeInfo provides the information of the corresponding blockchain node of the peer
//
type NodeInfo struct {
	PubKey  ecdsa.PublicKey
	address string
}

// CreateNodeInfo creates an instance of NodeInfo
func CreateNodeInfo(pubKey ecdsa.PublicKey) NodeInfo {
	return NodeInfo{
		PubKey: pubKey,
	}
}

// GetAddress returns the ledger address of the node
func (ni *NodeInfo) GetAddress() string {
	if len(ni.address) == 0 {
		addrBytes := crypto.PubkeyToAddress(ni.PubKey)
		ni.address = string(addrBytes[:])
	}
	return ni.address
}
