package messenger

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/p2p"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

func TestMessengerBroadcastMessages(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

	peerAPort := 24611
	peerBPort := 24612
	peerCPort := 24613
	peerANetAddr := "127.0.0.1:" + strconv.Itoa(peerAPort)
	//peerBNetAddr := "127.0.0.1:" + strconv.Itoa(peerBPort)
	peerCNetAddr := "127.0.0.1:" + strconv.Itoa(peerCPort)

	peerCMessages := []string{
		"Hi this is Peer C",
		"Let me tell you guys something exciting",
		"Theta is awesome, period",
	}

	// ---------------- Simulate PeerA ---------------- //

	peerAReady := make(chan bool)
	var peerAMessageHandler p2p.MessageHandler
	go func() {
		seedPeerNetAddressStrs := []string{} // passively listen
		messenger := newTestMessenger(seedPeerNetAddressStrs, peerAPort)
		peerID := messenger.nodeInfo.PubKey.Address().Hex()
		peerAMessageHandler = newTestMessageHandler(peerID, t, assert)
		messenger.RegisterMessageHandler(peerAMessageHandler)
		messenger.Start(ctx)

		peerAReady <- true // Peer A is ready, it has started
		log.Infof(">>> Peer A ID: %v", peerID)
	}()

	// ---------------- Simulate PeerB ---------------- //

	peerBReady := make(chan bool)
	var peerBMessageHandler p2p.MessageHandler
	go func() {
		seedPeerNetAddressStrs := []string{peerCNetAddr} // passively listen + actively connect to Peer C
		messenger := newTestMessenger(seedPeerNetAddressStrs, peerBPort)
		peerID := messenger.nodeInfo.PubKey.Address().Hex()
		peerBMessageHandler = newTestMessageHandler(peerID, t, assert)
		messenger.RegisterMessageHandler(peerBMessageHandler)
		messenger.Start(ctx)

		numPeers := len(seedPeerNetAddressStrs)
		for i := 0; i < numPeers; i++ {
			connected := <-messenger.discMgr.seedPeerConnector.Connected
			assert.True(connected)
		}
		peerBReady <- true // Peer B is ready, it has started and connected with all its seed peers (i.e. Peer C)
		log.Infof(">>> Peer B ID: %v", peerID)
	}()

	// ---------------- Simulate PeerC (i.e. us) ---------------- //

	seedPeerNetAddressStrs := []string{peerANetAddr} // passively listen + actively connect to Peer A
	messenger := newTestMessenger(seedPeerNetAddressStrs, peerCPort)
	peerID := messenger.nodeInfo.PubKey.Address().Hex()
	peerCMessageHandler := newTestMessageHandler(peerID, t, assert)
	messenger.RegisterMessageHandler(peerCMessageHandler)
	messenger.Start(ctx)

	numPeers := len(seedPeerNetAddressStrs)
	for i := 0; i < numPeers; i++ {
		connected := <-messenger.discMgr.seedPeerConnector.Connected
		assert.True(connected)
	} // Peer C is ready, it has started and connected with all its seed peers (i.e. Peer A)
	log.Infof(">>> Peer C ID: %v", peerID)

	// ---------------- Wait until all peers are ready ---------------- //

	_ = <-peerAReady
	_ = <-peerBReady

	// ---------------- PeerC broadcasts messages to PeerA and PeerB ---------------- //

	for _, peerCMsg := range peerCMessages {
		message := p2ptypes.Message{
			ChannelID: common.ChannelIDTransaction,
			Content:   peerCMsg,
		}
		messenger.Broadcast(message)
	}

	// ---------------- Check PeerA and PeerB both received the broadcasted messages ---------------- //
	numExpectedMsgs := len(peerCMessages)
	for i := 0; i < numExpectedMsgs; i++ {
		msgA := <-(peerAMessageHandler.(*TestMessageHandler)).recvMsgChan
		msgB := <-(peerBMessageHandler.(*TestMessageHandler)).recvMsgChan
		assert.True(checkReceivedMessage(&msgA, &peerCMessages))
		assert.True(checkReceivedMessage(&msgB, &peerCMessages))
	}
}

// --------------- Test Utilities --------------- //

// TestMessageHandler implements the MessageHandler interface
type TestMessageHandler struct {
	selfPeerID  string
	t           *testing.T
	assert      *assert.Assertions
	recvMsgChan chan string
}

func newTestMessageHandler(selfPeerID string, t *testing.T, assert *assert.Assertions) p2p.MessageHandler {
	return &TestMessageHandler{
		selfPeerID:  selfPeerID,
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

func (thm *TestMessageHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

func (thm *TestMessageHandler) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
		Content:   rawMessageBytes,
	}
	return message, nil
}

func (thm *TestMessageHandler) HandleMessage(message p2ptypes.Message) error {
	receivedBytes := (message.Content).(common.Bytes)
	var receivedMsgStr string
	err := rlp.DecodeBytes(receivedBytes, &receivedMsgStr)
	thm.assert.Nil(err)
	thm.recvMsgChan <- receivedMsgStr

	log.Infof(">>> HandleMessage\nPeer %v received a message on channelID: %v\nfrom %v\nReceived message: \"%v\"\n",
		thm.selfPeerID, message.ChannelID, message.PeerID, receivedMsgStr)

	return nil
}

func newTestMessenger(seedPeerNetAddressStrs []string, port int) *Messenger {
	randPeerPrivKey, _, _ := crypto.GenerateKeyPair()
	localNetworkAddress := "127.0.0.1:" + strconv.Itoa(port)
	testMsgrConfig := MessengerConfig{
		addrBookFilePath:    "./.addrbooks/addrbook_" + localNetworkAddress + ".json",
		routabilityRestrict: false,
		skipUPNP:            true,
		networkProtocol:     "tcp",
	}
	messenger, err := CreateMessenger(randPeerPrivKey, seedPeerNetAddressStrs, port, testMsgrConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Messenger instance: %v", err))
	}
	return messenger
}

func checkReceivedMessage(recvMsg *string, expectedMsgs *[]string) bool {
	for _, expectedMsg := range *expectedMsgs {
		if *recvMsg == expectedMsg {
			return true
		}
	}
	return false
}
