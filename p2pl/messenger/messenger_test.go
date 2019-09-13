package messenger

import (
	// "strconv"
	// "strings"
	"testing"

	// "github.com/stretchr/testify/assert"
	// "github.com/thetatoken/theta/common"
	// ks "github.com/thetatoken/theta/wallet/softwallet/keystore"
	// cr "github.com/libp2p/go-libp2p-crypto"
	// p2ptypes "github.com/thetatoken/theta/p2p/types"
)

func TestPubsubStreamSize(t *testing.T) {
	// assert := assert.New(t)

	// privKey1, pubKey1, _ := crypto.GenerateKeyPair()
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	// }
	// hostId1, _, err := cr.GenerateEd25519Key(strings.NewReader(common.Bytes2Hex(pubKey1.ToBytes())))
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to generate host id: %v", err))
	// }

	// port1 := 11001
	// port2 := 12001

	// privKey2, pubKey2, _ := crypto.GenerateKeyPair()
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	// }
	// hostId2, _, err := cr.GenerateEd25519Key(strings.NewReader(common.Bytes2Hex(pubKey2.ToBytes())))
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to generate host id: %v", err))
	// }

	// host1Seed := "/ip4/127.0.0.1/tcp/" + port2 + "/ipfs/" + hostId2
	// host2Seed := "/ip4/127.0.0.1/tcp/" + port1 + "/ipfs/" + hostId1
	
	// node1 := newMessenger(privKey1, {host1Seed}, port1)
	// err = node1.Start(context.Background())
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to start node: %v", err))
	// }

	// node2 := newMessenger(privKey2, {host2Seed}, port2)
	// err = node2.Start(context.Background())
	// if err != nil {
	// 	panic(fmt.Sprintf("Failed to start node: %v", err))
	// }

	// var msgStr string
	// for i := 0; i < 300; i++ {
	// 	msgStr = msgStr + "0123456789"
	// }
	// msgBytes := []byte(msgStr)

	// message := p2ptypes.Message{
	// 	ChannelID: 1,
	// 	Content:   msgBytes,
	// }

	// node1.Publish(message)
}

// func newMessenger(privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int) *msgl.Messenger {
// 	msgrConfig := GetDefaultMessengerConfig()
// 	messenger, _ := CreateMessenger(privKey.PublicKey(), seedPeerNetAddresses, port, msgrConfig)
// 	return messenger
// }