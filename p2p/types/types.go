package types

import (
	"fmt"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
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
	PrivKey     *crypto.PrivateKey `rlp:"-"`
	PubKey      *crypto.PublicKey  `rlp:"-"`
	PubKeyBytes common.Bytes       // needed for RLP serialization
	Port        uint16
}

// CreateNodeInfo creates an instance of NodeInfo
func CreateNodeInfo(pubKey *crypto.PublicKey, port uint16) NodeInfo {
	nodeInfo := NodeInfo{
		PubKey:      pubKey,
		PubKeyBytes: pubKey.ToBytes(),
		Port:        port,
	}
	return nodeInfo
}

// CreateLocalNodeInfo creates an instance of NodeInfo
func CreateLocalNodeInfo(privateKey *crypto.PrivateKey, port uint16) NodeInfo {
	pubKey := privateKey.PublicKey()
	nodeInfo := NodeInfo{
		PrivKey:     privateKey,
		PubKey:      pubKey,
		PubKeyBytes: pubKey.ToBytes(),
		Port:        port,
	}
	return nodeInfo
}

const (
	// PingSignal represents a ping signal to a peer
	PingSignal = byte(0x0)

	// PongSignal represents a pong respond to a peer
	PongSignal = byte(0x1)
)

type StackError struct {
	Err   interface{}
	Stack []byte
}

func (se StackError) String() string {
	return fmt.Sprintf("Error: %v\nStack: %s", se.Err, se.Stack)
}

func (se StackError) Error() string {
	return se.String()
}
