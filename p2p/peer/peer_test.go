package peer

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	cn "github.com/thetatoken/ukulele/p2p/connection"
	nu "github.com/thetatoken/ukulele/p2p/netutil"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/rlp"
)

func TestPeerHandshakeAndCommunication(t *testing.T) {
	assert := assert.New(t)

	port := 38857
	generatedPeerAAddrChan := make(chan string)
	receivedPeerBAddrChan := make(chan string)

	// ------ Simulate PeerA, a remote node that attempts to create a connection with PeerB (i.e. us) ------ //

	messagesAtoB := []string{
		"Hi there, this is Peer A",
		"How's everything?",
		"The Theta blockchain is awesome!",
		"The Theta blockchain is the only end-to-end infrastructure for decentralized video streaming and delivery that provides both technical and economic solutions...This is one of the most exciting new blockchain ventures I have heard about.",
	}
	numMsgs := len(messagesAtoB)

	go func() {
		outboundPeer := newOutboundPeer("127.0.0.1:" + strconv.Itoa(port))
		randPeerPubKey := p2ptypes.GetTestRandPubKey()
		peerANodeInfo := p2ptypes.CreateNodeInfo(randPeerPubKey)
		err := outboundPeer.Handshake(&peerANodeInfo) // send out PeerA's node info
		assert.Nil(err)
		assert.True(outboundPeer.IsOutbound())

		generatedPeerAAddr := peerANodeInfo.Address
		receivedPeerBAddr := outboundPeer.nodeInfo.Address
		t.Logf("Generated PeerA nodeInfo.Address: %v", generatedPeerAAddr)
		t.Logf("Received  PeerB nodeInfo.Address: %v", receivedPeerBAddr)

		assert.Equal(receivedPeerBAddr, outboundPeer.ID()) // ID check

		generatedPeerAAddrChan <- generatedPeerAAddr
		receivedPeerBAddrChan <- receivedPeerBAddr

		outboundPeer.OnStart()

		for i := 0; i < numMsgs; i++ {
			assert.True(outboundPeer.Send(common.ChannelIDTransaction, messagesAtoB[i]))
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// ------ Simulate PeerB (i.e. us) that receives the incoming connection attempt from PeerA ------ //

	listener := p2ptypes.GetTestListener(port)
	netconn, err := listener.Accept()
	if err != nil {
		panic(fmt.Sprintf("Failed to listen to the netconn: %v", err))
	}
	defer netconn.Close()

	// Handshake checks
	inboundPeer := newInboundPeer(netconn)
	peerBPubKey := p2ptypes.GetTestRandPubKey()
	peerBNodeInfo := p2ptypes.CreateNodeInfo(peerBPubKey)
	err = inboundPeer.Handshake(&peerBNodeInfo) // send out PeerB's node info
	assert.Nil(err)
	assert.False(inboundPeer.IsOutbound())

	receivedPeerAAddr := inboundPeer.nodeInfo.Address
	generatedPeerBAddr := peerBNodeInfo.Address
	t.Logf("Received  PeerA nodeInfo.Address: %v", receivedPeerAAddr)
	t.Logf("Generated PeerB nodeInfo.Address: %v", generatedPeerBAddr)

	generatedPeerAAddr := <-generatedPeerAAddrChan
	receivedPeerBAddr := <-receivedPeerBAddrChan

	assert.Equal(generatedPeerAAddr, receivedPeerAAddr)
	assert.Equal(generatedPeerBAddr, receivedPeerBAddr)

	// ID checks
	assert.Equal(receivedPeerAAddr, inboundPeer.ID())

	// Persistency checks
	inboundPeer.SetPersistency(false)
	assert.False(inboundPeer.IsPersistent())
	inboundPeer.SetPersistency(true)
	assert.True(inboundPeer.IsPersistent())

	// Peer-to-Peer communication checks
	basicMessageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		message := p2ptypes.Message{
			ChannelID: channelID,
			Content:   rawMessageBytes,
		}
		return message, nil
	}

	matchedChan := make(chan bool)
	basicReceiveHandler := func(message p2ptypes.Message) error {
		t.Logf("Received channelID: %v", message.ChannelID)
		t.Logf("Received bytes: %v", message.Content)
		receivedBytes := (message.Content).(common.Bytes)
		var receivedMsgStr string
		err := rlp.DecodeBytes(receivedBytes, &receivedMsgStr)
		assert.Nil(err)

		t.Logf("Received message: %v", receivedMsgStr)
		matched := false
		for i := 0; i < numMsgs; i++ { // messages may arrive out-of-order
			if messagesAtoB[i] == receivedMsgStr {
				matched = true
				break
			}
		}

		if !matched {
			matchedChan <- false
			return errors.New("mismatch")
		}
		matchedChan <- true
		return nil
	}

	inboundPeer.GetConnection().SetMessageParser(basicMessageParser)
	inboundPeer.GetConnection().SetReceiveHandler(basicReceiveHandler)

	inboundPeer.OnStart()
	defer inboundPeer.OnStop()

	for i := 0; i < numMsgs; i++ {
		matched := <-matchedChan
		assert.True(matched)
	}
}

// --------------- Test Utilities --------------- //

func newOutboundPeer(ipAddr string) *Peer {
	netaddr, err := nu.NewNetAddressString(ipAddr)
	if err != nil {
		panic(fmt.Sprintf("Failed to create net address: %v", err))
	}
	peerConfig := GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	outboundPeer, err := CreateOutboundPeer(netaddr, peerConfig, connConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create outbound peer: %v", err))
	}

	return outboundPeer
}

func newInboundPeer(netconn net.Conn) *Peer {
	peerConfig := GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	inboundPeer, err := CreateInboundPeer(netconn, peerConfig, connConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create outbound peer: %v", err))
	}
	return inboundPeer
}
