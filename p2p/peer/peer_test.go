package peer

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	cn "github.com/thetatoken/theta/p2p/connection"
	nu "github.com/thetatoken/theta/p2p/netutil"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

func TestPeerHandshakeAndCommunication(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()

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
		randPeerPrivKey, _, _ := crypto.GenerateKeyPair()
		peerANodeInfo := p2ptypes.CreateLocalNodeInfo(randPeerPrivKey, uint16(port))
		err := outboundPeer.Handshake(&peerANodeInfo) // send out PeerA's node info
		assert.Nil(err)
		assert.True(outboundPeer.IsOutbound())

		generatedPeerAAddr := peerANodeInfo.PubKey.Address().Hex()
		receivedPeerBAddr := outboundPeer.nodeInfo.PubKey.Address().Hex()
		log.Infof("Generated PeerA nodeInfo.Address: %v", generatedPeerAAddr)
		log.Infof("Received  PeerB nodeInfo.Address: %v", receivedPeerBAddr)

		assert.Equal(receivedPeerBAddr, outboundPeer.ID()) // ID check

		generatedPeerAAddrChan <- generatedPeerAAddr
		receivedPeerBAddrChan <- receivedPeerBAddr

		outboundPeer.Start(ctx)

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
	peerBPrivKey, _, _ := crypto.GenerateKeyPair()
	peerBNodeInfo := p2ptypes.CreateLocalNodeInfo(peerBPrivKey, uint16(port))
	err = inboundPeer.Handshake(&peerBNodeInfo) // send out PeerB's node info
	assert.Nil(err)
	assert.False(inboundPeer.IsOutbound())

	receivedPeerAAddr := inboundPeer.nodeInfo.PubKey.Address().Hex()
	generatedPeerBAddr := peerBNodeInfo.PubKey.Address().Hex()
	log.Infof("Received  PeerA nodeInfo.Address: %v", receivedPeerAAddr)
	log.Infof("Generated PeerB nodeInfo.Address: %v", generatedPeerBAddr)

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
		log.Infof("Received channelID: %v", message.ChannelID)
		log.Infof("Received bytes: %v", message.Content)
		receivedBytes := (message.Content).(common.Bytes)
		var receivedMsgStr string
		err := rlp.DecodeBytes(receivedBytes, &receivedMsgStr)
		assert.Nil(err)

		log.Infof("Received message: %v", receivedMsgStr)
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

	inboundPeer.Start(ctx)
	defer inboundPeer.Stop()

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
