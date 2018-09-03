package messenger

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

func TestMessenger(t *testing.T) {
	assert := assert.New(t)

	peerANetAddr := "127.0.0.1:24611"
	peerBNetAddr := "127.0.0.1:24612"
	peerCNetAddr := "127.0.0.1:24613"

	peerAMsg := "Hi this is Peer A"
	peerBMsg := "Hi this is Peer B"
	peerCMsg := "Hi this is Peer C"

	//numExpectedMessages := 2

	// ---------------- Simulate PeerA ---------------- //

	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		localNetworkAddress := peerANetAddr
		messenger := newTestMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		messageHandler := newTestMessageHandler(t, assert)
		messenger.AddMessageHandler(messageHandler)
		messenger.OnStart()

		message := p2ptypes.Message{
			ChannelID: common.ChannelIDTransaction,
			Content:   peerAMsg,
		}
		messenger.Broadcast(message)

		peerID := messenger.nodeInfo.Address
		t.Logf("[Peer A] ID: %v", peerID)
	}()

	// ---------------- Simulate PeerB ---------------- //

	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // passively listen + actively connect to Peer C
		localNetworkAddress := peerBNetAddr
		messenger := newTestMessenger(seedPeerNetAddressStrs, localNetworkAddress)
		messageHandler := newTestMessageHandler(t, assert)
		messenger.AddMessageHandler(messageHandler)
		messenger.OnStart()

		message := p2ptypes.Message{
			ChannelID: common.ChannelIDTransaction,
			Content:   peerBMsg,
		}
		messenger.Broadcast(message)

		peerID := messenger.nodeInfo.Address
		t.Logf("[Peer B] ID: %v", peerID)
	}()

	// ---------------- Simulate PeerC (i.e. us) ---------------- //

	seedPeerNetAddressStrs := []string{peerANetAddr} // passively listen + actively connect to Peer A
	localNetworkAddress := peerCNetAddr
	messenger := newTestMessenger(seedPeerNetAddressStrs, localNetworkAddress)
	messageHandler := newTestMessageHandler(t, assert)
	messenger.AddMessageHandler(messageHandler)
	messenger.OnStart()

	message := p2ptypes.Message{
		ChannelID: common.ChannelIDTransaction,
		Content:   peerCMsg,
	}
	messenger.Broadcast(message)

	/*
		for i := 0; i < numExpectedMessages; i++ {
			tmh := messageHandler.(*TestMessageHandler)
			recvMsg := <-tmh.recvMsgChan
			t.Logf("[Peer C] received: %v", recvMsg)
		}
	*/
}

// --------------- Test Utilities --------------- //

// TestMessageHandler implements the MessageHandler interface
type TestMessageHandler struct {
	t           *testing.T
	assert      *assert.Assertions
	recvMsgChan chan string
}

func newTestMessageHandler(t *testing.T, assert *assert.Assertions) p2p.MessageHandler {
	return &TestMessageHandler{
		t:           t,
		assert:      assert,
		recvMsgChan: make(chan string),
	}
}

func (thm *TestMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDTransaction,
	}
}

func (thm *TestMessageHandler) ParseMessage(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		ChannelID: channelID,
		Content:   rawMessageBytes,
	}
	return message, nil
}

func (thm *TestMessageHandler) HandleMessage(peerID string, message p2ptypes.Message) error {
	thm.t.Logf("Received channelID: %v, from peerID: %v", message.ChannelID, peerID)
	thm.t.Logf("Received bytes: %v", message.Content)
	receivedBytes := (message.Content).(common.Bytes)
	var receivedMsgStr string
	err := rlp.DecodeBytes(receivedBytes, &receivedMsgStr)
	thm.assert.Nil(err)
	thm.t.Logf("Received message: %v", receivedMsgStr)
	thm.recvMsgChan <- receivedMsgStr

	return nil
}

func newTestMessenger(seedPeerNetAddressStrs []string, localNetworkAddress string) *Messenger {
	peerPubKey := p2ptypes.GetTestRandPubKey()
	peerNodeInfo := p2ptypes.CreateNodeInfo(peerPubKey)
	addrbookPath := "./.addrbooks/addrbook_" + localNetworkAddress + ".json"
	routabilityRestrict := false
	selfNetAddressStr := "104.105.23.92:8888" // not important for the test
	networkProtocol := "tcp"
	skipUPNP := true
	messenger, err := CreateMessenger(peerNodeInfo, addrbookPath, routabilityRestrict, selfNetAddressStr,
		seedPeerNetAddressStrs, networkProtocol, localNetworkAddress, skipUPNP)
	if err != nil {
		panic(fmt.Sprintf("Failed to create PeerDiscoveryManager instance: %v", err))
	}
	return messenger
}
