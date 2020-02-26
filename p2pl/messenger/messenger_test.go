package messenger

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

type MockMsgHandler struct {
	C chan interface{}
}

func (m *MockMsgHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDBlock,
	}
}

func decodeMessage(raw common.Bytes) (interface{}, error) {
	data := []byte{}
	err := rlp.DecodeBytes(raw, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (m *MockMsgHandler) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
	}
	data, err := decodeMessage(rawMessageBytes)
	message.Content = data
	return message, err
}

func (m *MockMsgHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	var buf bytes.Buffer
	err := rlp.Encode(&buf, message)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *MockMsgHandler) HandleMessage(message p2ptypes.Message) error {
	m.C <- message.Content
	return nil
}

func newMessenger(privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int) *Messenger {
	msgrConfig := GetDefaultMessengerConfig()
	messenger, _ := CreateMessenger(privKey.PublicKey(), seedPeerNetAddresses, port, true, msgrConfig, false)
	return messenger
}

func TestSendMessage(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	var err error
	var ok bool
	var content []byte

	mockMsgHandler1 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler2 := &MockMsgHandler{C: make(chan interface{}, 1)}

	port1 := 11001
	port2 := 12001

	privKey1, _, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey2, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}

	node1 := newMessenger(privKey1, []string{}, port1)
	node1Id := node1.host.ID()
	node1.RegisterMessageHandler(mockMsgHandler1)

	host2Seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port1, node1Id)
	node2 := newMessenger(privKey2, []string{host2Seed}, port2)
	node2Id := node2.host.ID()
	node2.RegisterMessageHandler(mockMsgHandler2)

	err = node1.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node1: %v", err))
	}

	err = node2.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node2: %v", err))
	}

	time.Sleep(1 * time.Second)

	msgBytes := []byte{}
	n := uint(8192 * 32)
	bytes := []byte("01234567890123450123456789012345012345678901234501234567890123450123456789012345012345678901234501234567890123450123456789012345") // 128 Bytes
	i := uint(0)
	// make just below 1MB
	for ; i < n; i++ {
		msgBytes = append(msgBytes, bytes...)
	}
	message := p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   msgBytes,
	}

	for k := 0; k < 16; k++ {
		// Send from node1 to node2
		node1.Send(node2Id.String(), message)
		data := <-mockMsgHandler2.C
		content, ok = data.([]byte)
		assert.True(ok)
		assert.Equal(128*n, uint(len(content)))

		// Send from node2 to node1
		node2.Send(node1Id.String(), message)
		data = <-mockMsgHandler1.C
		content, ok = data.([]byte)
		assert.True(ok)
		assert.Equal(128*n, uint(len(content)))
	}
}

func TestBroadcastMessage(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	var err error
	var ok bool
	var content []byte

	mockMsgHandler1 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler2 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler3 := &MockMsgHandler{C: make(chan interface{}, 1)}

	port1 := 11001
	port2 := 12001
	port3 := 13001

	privKey1, _, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey2, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey3, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}

	node1 := newMessenger(privKey1, []string{}, port1)
	node1Id := node1.host.ID()
	node1.RegisterMessageHandler(mockMsgHandler1)
	node1Addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port1, node1Id)

	node2 := newMessenger(privKey2, []string{node1Addr}, port2)
	node2Id := node2.host.ID()
	node2.RegisterMessageHandler(mockMsgHandler2)
	node2Addr := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port2, node2Id)

	node3 := newMessenger(privKey3, []string{node1Addr, node2Addr}, port3)
	// node3Id := node3.host.ID()
	node3.RegisterMessageHandler(mockMsgHandler3)

	err = node1.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node1: %v", err))
	}

	err = node2.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node2: %v", err))
	}

	err = node3.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node3: %v", err))
	}

	time.Sleep(1 * time.Second)

	msgBytes := []byte("0123456789")
	n := len(msgBytes)
	message := p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   msgBytes,
	}

	for k := 0; k < 10; k++ {
		go func() {
			node1.Broadcast(message)
			data := <-mockMsgHandler2.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
			data = <-mockMsgHandler3.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
		}()

		go func() {
			node2.Broadcast(message)
			data := <-mockMsgHandler1.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
			data = <-mockMsgHandler3.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
		}()

		go func() {
			node3.Broadcast(message)
			data := <-mockMsgHandler1.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
			data = <-mockMsgHandler2.C
			content, ok = data.([]byte)
			assert.True(ok)
			assert.Equal(n, len(content))
		}()
	}
}

func TestFullyConnected(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	var err error
	var ok bool
	var content []byte

	mockMsgHandler1 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler2 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler3 := &MockMsgHandler{C: make(chan interface{}, 1)}

	port1 := 11001
	port2 := 12001
	port3 := 13001

	privKey1, _, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey2, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey3, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}

	node1 := newMessenger(privKey1, []string{}, port1)
	node1Id := node1.host.ID()
	node1.RegisterMessageHandler(mockMsgHandler1)

	host2Seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port1, node1Id)
	node2 := newMessenger(privKey2, []string{host2Seed}, port2)
	node2Id := node2.host.ID()
	node2.RegisterMessageHandler(mockMsgHandler2)

	host3Seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port2, node2Id)
	node3 := newMessenger(privKey3, []string{host3Seed}, port3)
	// node3Id := node3.host.ID()
	node3.RegisterMessageHandler(mockMsgHandler3)

	err = node1.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node1: %v", err))
	}

	err = node2.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node2: %v", err))
	}

	err = node3.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node3: %v", err))
	}

	time.Sleep(3 * time.Second)

	msgBytes := []byte{}
	n := uint(8191 * 16)
	bytes := []byte("01234567890123450123456789012345012345678901234501234567890123450123456789012345012345678901234501234567890123450123456789012345") // 128 Bytes
	i := uint(0)
	// make just below 1MB
	for ; i < n; i++ {
		msgBytes = append(msgBytes, bytes...)
	}
	message := p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   msgBytes,
	}

	// // Send from node1 to node2
	// node1.Send(node2Id.String(), message)
	// data := <-mockMsgHandler2.C
	// content, ok = data.([]byte)
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// assert.Equal(0, len(mockMsgHandler1.C))
	// assert.Equal(0, len(mockMsgHandler2.C))
	// assert.Equal(0, len(mockMsgHandler3.C))

	// // Send from node2 to node1
	// node2.Send(node1Id.String(), message)
	// data = <-mockMsgHandler1.C
	// content, ok = data.([]byte)
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// assert.Equal(0, len(mockMsgHandler3.C))

	// // Send from node2 to node3
	// node2.Send(node3Id.String(), message)
	// data = <-mockMsgHandler3.C
	// content, ok = data.([]byte)
	// assert.True(ok)
	// assert.Equal(128*n, uint(uint(len(content))))

	// assert.Equal(0, len(mockMsgHandler1.C))

	// // Publish from node1
	// err = node1.Publish(message)
	// assert.Nil(err)

	// data = <-mockMsgHandler2.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler3.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// // Publish from node2
	// err = node2.Publish(message)
	// assert.Nil(err)

	// data = <-mockMsgHandler1.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler3.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// // Publish from node3
	// err = node3.Publish(message)
	// assert.Nil(err)

	// data = <-mockMsgHandler1.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler2.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// msg size -> 1MB
	msgBytes = append(msgBytes, bytes...)
	message = p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   msgBytes,
	}
	n++

	// Broadcast from node1
	node1.Broadcast(message)

	data := <-mockMsgHandler2.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler3.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// // Broadcast from node2
	// node2.Broadcast(message)

	// data = <-mockMsgHandler1.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler3.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// // Broadcast from node3
	// node3.Broadcast(message)

	// data = <-mockMsgHandler1.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))

	// data = <-mockMsgHandler2.C
	// if content, ok = data.([]byte); ok {
	// }
	// assert.True(ok)
	// assert.Equal(128*n, uint(len(content)))
}

func TestPartiallyConnected(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	var err error
	var ok bool
	var content []byte

	mockMsgHandler1 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler2 := &MockMsgHandler{C: make(chan interface{}, 1)}
	mockMsgHandler3 := &MockMsgHandler{C: make(chan interface{}, 1)}

	port1 := 11001
	port2 := 12001
	port3 := 13001

	privKey1, _, err := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey2, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}
	privKey3, _, _ := crypto.GenerateKeyPair()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair: %v", err))
	}

	node1 := newMessenger(privKey1, []string{}, port1)
	node1Id := node1.host.ID()
	node1.RegisterMessageHandler(mockMsgHandler1)

	host2Seed := fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port1, node1Id)
	node2 := newMessenger(privKey2, []string{host2Seed}, port2)
	node2Id := node2.host.ID()
	node2.RegisterMessageHandler(mockMsgHandler2)

	node3 := newMessenger(privKey3, []string{}, port3)
	node3Id := node3.host.ID()
	node3.RegisterMessageHandler(mockMsgHandler3)

	err = node1.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node1: %v", err))
	}

	err = node2.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node2: %v", err))
	}

	err = node3.Start(ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start node3: %v", err))
	}

	time.Sleep(1 * time.Second)

	msgBytes := []byte("01234567890123450123456789012345012345678901234501234567890123450123456789012345012345678901234501234567890123450123456789012345") // 128 Bytes
	message := p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   msgBytes,
	}

	// Send from node1
	node1.Send(node2Id.String(), message)
	data := <-mockMsgHandler2.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))

	node1.Send(node3Id.String(), message)
	assert.Equal(0, len(mockMsgHandler3.C))

	// Send from node2
	node2.Send(node1Id.String(), message)
	data = <-mockMsgHandler1.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))

	node2.Send(node3Id.String(), message)
	assert.Equal(0, len(mockMsgHandler3.C))

	// Send from node3
	node3.Send(node1Id.String(), message)
	assert.Equal(0, len(mockMsgHandler1.C))

	node3.Send(node2Id.String(), message)
	assert.Equal(0, len(mockMsgHandler2.C))

	// Broadcast from node1
	node1.Broadcast(message)
	data = <-mockMsgHandler2.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))
	assert.Equal(0, len(mockMsgHandler3.C))

	// Broadcast from node2
	node2.Broadcast(message)
	data = <-mockMsgHandler1.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))
	assert.Equal(0, len(mockMsgHandler3.C))

	// Broadcast from node3
	node3.Broadcast(message)
	assert.Equal(0, len(mockMsgHandler1.C))
	assert.Equal(0, len(mockMsgHandler2.C))

	// Publish from node1
	err = node1.Publish(message)
	assert.Nil(err)
	data = <-mockMsgHandler2.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))
	assert.Equal(0, len(mockMsgHandler3.C))

	// Publish from node2
	err = node2.Publish(message)
	assert.Nil(err)
	data = <-mockMsgHandler1.C
	if content, ok = data.([]byte); ok {
	}
	assert.True(ok)
	assert.Equal(128, len(content))
	assert.Equal(0, len(mockMsgHandler3.C))

	// Publish from node3
	err = node3.Publish(message)
	assert.Nil(err)
	assert.Equal(0, len(mockMsgHandler1.C))
	assert.Equal(0, len(mockMsgHandler2.C))
}
