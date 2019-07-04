package connection

import (
	"context"
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

func TestNetconnBasics(t *testing.T) {
	assert := assert.New(t)
	port := 43251
	msgBytes := []byte("Hello world")
	go func() {
		netconn := p2ptypes.GetTestNetconn(port)
		defer netconn.Close()
		netconn.Write(msgBytes)
	}()

	listener := p2ptypes.GetTestListener(port)

	netconn, err := listener.Accept()
	assert.Nil(err)
	defer netconn.Close()

	buf, err := ioutil.ReadAll(netconn)
	assert.Nil(err)

	t.Logf(string(buf[:]))
	assert.Equal(buf, msgBytes)
}

func TestNetconnSendPacket(t *testing.T) {
	assert := assert.New(t)
	port := 43252
	msgBytes := []byte("Hello world")
	go func() {
		netconn := p2ptypes.GetTestNetconn(port)
		defer netconn.Close()
		packet := Packet{
			ChannelID: common.ChannelIDTransaction,
			Bytes:     msgBytes,
			IsEOF:     byte(0x01),
		}
		packetBytes, _ := rlp.EncodeToBytes(packet)
		netconn.Write(packetBytes)
		//rlp.Encode(netconn, packet)
	}()

	listener := p2ptypes.GetTestListener(port)

	netconn, err := listener.Accept()
	assert.Nil(err)
	defer netconn.Close()

	var packet Packet
	err = rlp.Decode(netconn, &packet)
	assert.Nil(err)
	assert.Equal(common.ChannelIDTransaction, packet.ChannelID)
	assert.Equal(byte(0x01), packet.IsEOF)
	assert.Equal(msgBytes, packet.Bytes)

	t.Logf("Received packet.Bytes: %v", string(packet.Bytes[:]))
}

func TestChannelSendPacketThroughNetconn(t *testing.T) {
	assert := assert.New(t)
	msgBytes := []byte("Hello world")
	port := 43253
	go func() {
		netconn := p2ptypes.GetTestNetconn(port)
		defer netconn.Close()
		channel := createDefaultChannel(common.ChannelIDTransaction)
		channel.enqueueMessage(msgBytes)
		cfg := GetDefaultConnectionConfig()
		conn := CreateConnection(netconn, cfg)
		conn.Start(context.Background())
		nonemptyPacket, _, err := channel.sendPacketTo(conn)
		assert.True(nonemptyPacket)
		assert.Nil(err)
		conn.flush()
	}()

	listener := p2ptypes.GetTestListener(port)

	netconn, err := listener.Accept()
	assert.Nil(err)
	defer netconn.Close()

	var packet Packet
	err = rlp.Decode(netconn, &packet)
	assert.Nil(err)
	assert.Equal(common.ChannelIDTransaction, packet.ChannelID)
	assert.Equal(byte(0x01), packet.IsEOF)
	assert.Equal(msgBytes, packet.Bytes)

	t.Logf("Received packet.Bytes: %v", string(packet.Bytes[:]))
}

func TestConnectionSendNodeInfo(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	port := 43254

	_, randPubKey, err := crypto.GenerateKeyPair()
	origNodeInfo := p2ptypes.CreateNodeInfo(randPubKey, uint16(port))
	assert.Nil(err)

	basicMessageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		message := p2ptypes.Message{
			ChannelID: channelID,
			Content:   rawMessageBytes,
		}
		return message, nil
	}

	basicReceiveHandler := func(message p2ptypes.Message) error {
		t.Logf("Received channelID: %v", message.ChannelID)
		t.Logf("Received bytes: %v", message.Content)
		receivedBytes := (message.Content).(common.Bytes)
		var receivedNodeInfo p2ptypes.NodeInfo
		err := rlp.DecodeBytes(receivedBytes, &receivedNodeInfo)
		assert.Nil(err)

		receivedNodeInfo.PubKey, err = crypto.PublicKeyFromBytes(receivedNodeInfo.PubKeyBytes)
		assert.Nil(err)

		t.Logf("receivedNodeInfo.Address: %v", receivedNodeInfo.PubKey.Address().Hex())
		if origNodeInfo.PubKey.Address() != receivedNodeInfo.PubKey.Address() {
			return errors.New("mismatch")
		}
		return nil
	}

	numMessages := 1
	go func(port int, origNodeInfo p2ptypes.NodeInfo) {
		netconn := p2ptypes.GetTestNetconn(port)
		cfg := GetDefaultConnectionConfig()
		conn := CreateConnection(netconn, cfg)
		conn.Start(ctx)
		//defer conn.Stop()
		numMsgSent := 0
		for {
			if conn.CanEnqueueMessage(common.ChannelIDTransaction) {
				assert.True(conn.EnqueueMessage(common.ChannelIDTransaction, origNodeInfo))
				numMsgSent++
			}
			if numMsgSent >= numMessages {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}(port, origNodeInfo)

	matched := make(chan bool)
	go func() {
		listener := p2ptypes.GetTestListener(port)
		netconn, err := listener.Accept()
		assert.Nil(err)
		defer netconn.Close()

		for {
			if len(matched) >= numMessages {
				break
			}

			var packet Packet
			err = rlp.Decode(netconn, &packet)
			if err != nil {
				matched <- false
				continue
			}

			if (common.ChannelIDTransaction != packet.ChannelID) || packet.IsEOF != byte(0x01) {
				matched <- false
				continue
			}

			message, err := basicMessageParser(packet.ChannelID, packet.Bytes)
			if err != nil {
				matched <- false
				continue
			}

			err = basicReceiveHandler(message)
			if err != nil {
				matched <- false
				continue
			}

			matched <- true

			t.Logf("origNodeInfo.Address:     %v", origNodeInfo.PubKey.Address().Hex())
			t.Logf("packet.ChannelID: %v", packet.ChannelID)
			t.Logf("packet.Bytes: %v", string(packet.Bytes[:]))
			t.Logf("packet.IsEOF: %v", packet.IsEOF)
		}
	}()

	for i := 0; i < numMessages; i++ {
		resultMatched := <-matched
		assert.True(resultMatched)
	}
}

func TestConnectionRecvNodeInfo(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	port := 43255

	_, randPubKey, err := crypto.GenerateKeyPair()
	origNodeInfo := p2ptypes.CreateNodeInfo(randPubKey, uint16(port))
	assert.Nil(err)

	basicMessageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		message := p2ptypes.Message{
			ChannelID: channelID,
			Content:   rawMessageBytes,
		}
		return message, nil
	}

	matched := make(chan bool)
	basicReceiveHandler := func(message p2ptypes.Message) error {
		t.Logf("Received channelID: %v", message.ChannelID)
		t.Logf("Received bytes: %v", message.Content)
		receivedBytes := (message.Content).(common.Bytes)
		var receivedNodeInfo p2ptypes.NodeInfo
		err := rlp.DecodeBytes(receivedBytes, &receivedNodeInfo)
		assert.Nil(err)

		receivedNodeInfo.PubKey, err = crypto.PublicKeyFromBytes(receivedNodeInfo.PubKeyBytes)
		assert.Nil(err)

		t.Logf("origNodeInfo.Address:     %v", origNodeInfo.PubKey.Address().Hex())
		t.Logf("receivedNodeInfo.Address: %v", receivedNodeInfo.PubKey.Address().Hex())
		if origNodeInfo.PubKey.Address() != receivedNodeInfo.PubKey.Address() {
			matched <- false
			return errors.New("mismatch")
		}
		matched <- true
		return nil
	}

	numMessages := 8
	go func(port int, origNodeInfo p2ptypes.NodeInfo) {
		netconn := p2ptypes.GetTestNetconn(port)
		msgBytes, err := rlp.EncodeToBytes(origNodeInfo)
		assert.Nil(err)
		packet := Packet{
			ChannelID: common.ChannelIDTransaction,
			Bytes:     msgBytes,
			IsEOF:     byte(0x01),
		}
		packetBytes, err := rlp.EncodeToBytes(packet)
		assert.Nil(err)
		for i := 0; i < numMessages; i++ {
			netconn.Write(packetBytes)
		}
	}(port, origNodeInfo)

	listener := p2ptypes.GetTestListener(port)
	netconn, err := listener.Accept()
	assert.Nil(err)

	cfg := GetDefaultConnectionConfig()
	conn := CreateConnection(netconn, cfg)
	conn.SetMessageParser(basicMessageParser)
	conn.SetReceiveHandler(basicReceiveHandler)
	conn.Start(ctx)
	defer conn.Stop()

	for i := 0; i < numMessages; i++ {
		resultMatched := <-matched
		assert.True(resultMatched)
	}
}
