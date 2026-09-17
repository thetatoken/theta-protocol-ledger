package connection

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestConnectionRejectsOversizedOutgoingMessages(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	conn := CreateConnection(left, GetDefaultConnectionConfig())
	conn.SetMessageEncoder(func(_ common.ChannelIDEnum, message interface{}) (common.Bytes, error) {
		return message.([]byte), nil
	})

	overLimit := bytes.Repeat([]byte{0x42}, common.MaxNormalMessageSize+1)
	assert.False(t, conn.EnqueueMessage(common.ChannelIDTransaction, overLimit))
	assert.False(t, conn.AttemptToEnqueueMessage(common.ChannelIDTransaction, overLimit))

	atLimit := bytes.Repeat([]byte{0x42}, common.MaxNormalMessageSize)
	assert.True(t, conn.EnqueueMessage(common.ChannelIDTransaction, atLimit))
}

func TestConnectionDisconnectsOnOversizedIncomingMessage(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()

	conn := CreateConnection(left, GetDefaultConnectionConfig())
	channel := conn.channelGroup.getChannel(common.ChannelIDTransaction)
	channel.recvBuf.setMaxMessageSize(2 * maxPayloadSize)
	errCh := make(chan interface{}, 1)
	conn.SetErrorHandler(func(reason interface{}) {
		errCh <- reason
	})
	conn.Start(context.Background())
	defer conn.Stop()

	go func() {
		for seq := uint(0); seq < 2; seq++ {
			_ = rlp.Encode(right, &Packet{
				ChannelID: common.ChannelIDTransaction,
				Bytes:     bytes.Repeat([]byte{0x42}, maxPayloadSize),
				SeqID:     seq,
			})
		}
		_ = rlp.Encode(right, &Packet{
			ChannelID: common.ChannelIDTransaction,
			Bytes:     []byte{0x42},
			IsEOF:     byte(0x01),
			SeqID:     2,
		})
	}()

	select {
	case reason := <-errCh:
		require.Contains(t, fmt.Sprint(reason), "message exceeds receive limit")
		require.True(t, IsProtocolError(reason))
	case <-time.After(2 * time.Second):
		t.Fatal("connection did not reject an oversized reassembled message")
	}
	require.Equal(t, uint32(1), atomic.LoadUint32(&conn.errored))
}

func TestConnectionDisconnectsWhenMessageNeverFinishes(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()

	conn := CreateConnection(left, GetDefaultConnectionConfig())
	channel := conn.channelGroup.getChannel(common.ChannelIDTransaction)
	channel.recvBuf.config.messageTimeout = 25 * time.Millisecond
	errCh := make(chan interface{}, 1)
	conn.SetErrorHandler(func(reason interface{}) {
		errCh <- reason
	})
	conn.Start(context.Background())
	defer conn.Stop()

	require.NoError(t, rlp.Encode(right, &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("partial"),
		SeqID:     0,
	}))

	select {
	case reason := <-errCh:
		require.Contains(t, fmt.Sprint(reason), "timeout")
	case <-time.After(time.Second):
		t.Fatal("connection did not expire an incomplete message")
	}
	require.Equal(t, uint32(1), atomic.LoadUint32(&conn.errored))
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
	senderDone := make(chan struct{})
	stopSender := make(chan struct{})
	go func(port int, origNodeInfo p2ptypes.NodeInfo) {
		defer close(senderDone)
		netconn := p2ptypes.GetTestNetconn(port)
		cfg := GetDefaultConnectionConfig()
		conn := CreateConnection(netconn, cfg)
		conn.Start(ctx)
		defer conn.Stop()
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
		<-stopSender
	}(port, origNodeInfo)

	matched := make(chan bool)
	receiverDone := make(chan struct{})
	go func() {
		defer close(receiverDone)
		listener := p2ptypes.GetTestListener(port)
		defer listener.Close()
		netconn, err := listener.Accept()
		assert.Nil(err)
		defer netconn.Close()

		for i := 0; i < numMessages; i++ {
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
	<-receiverDone
	close(stopSender)
	<-senderDone
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
