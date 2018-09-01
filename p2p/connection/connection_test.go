package connection

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

func getNetconn(port int) net.Conn {
	netconn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to create a net connection: %v", err))
	}
	return netconn
}

func getListener(port int) net.Listener {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen: %v", err))
	}

	return listener
}

func TestNetconnBasics(t *testing.T) {
	assert := assert.New(t)
	port := 43251
	msgBytes := []byte("Hello world")
	go func() {
		netconn := getNetconn(port)
		defer netconn.Close()
		netconn.Write(msgBytes)
	}()

	listener := getListener(port)

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
		netconn := getNetconn(port)
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

	listener := getListener(port)

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
		netconn := getNetconn(port)
		defer netconn.Close()
		channel := createDefaultChannel(common.ChannelIDTransaction)
		channel.enqueueMessage(msgBytes)
		channel.sendPacketTo(netconn)
	}()

	listener := getListener(port)

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
	port := 43254

	randPrivKey, err := crypto.GenerateKey()
	origNodeInfo := p2ptypes.CreateNodeInfo(randPrivKey.PublicKey)
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

		t.Logf("receivedNodeInfo.Address: %v", receivedNodeInfo.Address)
		if origNodeInfo.Address != receivedNodeInfo.Address {
			return errors.New("mismatch")
		}
		return nil
	}

	go func(port int, origNodeInfo p2ptypes.NodeInfo) {
		netconn := getNetconn(port)
		cfgA := GetDefaultConnectionConfig()
		connA := CreateConnection(netconn, cfgA)
		connA.OnStart()
		assert.True(connA.EnqueueMessage(common.ChannelIDTransaction, origNodeInfo))
	}(port, origNodeInfo)

	listener := getListener(port)
	netconn, err := listener.Accept()
	assert.Nil(err)
	defer netconn.Close()

	var packet Packet
	err = rlp.Decode(netconn, &packet)
	assert.Nil(err)

	assert.Equal(common.ChannelIDTransaction, packet.ChannelID)
	assert.Equal(byte(0x01), packet.IsEOF)
	message, err := basicMessageParser(packet.ChannelID, packet.Bytes)
	assert.Nil(err)
	err = basicReceiveHandler(message)
	assert.Nil(err)

	t.Logf("origNodeInfo.Address:     %v", origNodeInfo.Address)

	t.Logf("packet.ChannelID: %v", packet.ChannelID)
	t.Logf("packet.Bytes: %v", string(packet.Bytes[:]))
	t.Logf("packet.IsEOF: %v", packet.IsEOF)
}
