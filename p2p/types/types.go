package types

import (
	"fmt"

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
	PubKey      *crypto.PublicKey `rlp:"-"`
	PubKeyBytes common.Bytes      // needed for RLP serialization
}

// CreateNodeInfo creates an instance of NodeInfo
func CreateNodeInfo(pubKey *crypto.PublicKey) NodeInfo {
	nodeInfo := NodeInfo{
		PubKey:      pubKey,
		PubKeyBytes: pubKey.ToBytes(),
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

const DefaultPort = 1688
