package connection

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
)

func TestPacketEmptiness(t *testing.T) {
	assert := assert.New(t)

	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
	}
	assert.True(packet.isEmpty())

	packet.Bytes = []byte("hello world")
	assert.False(packet.isEmpty())
}

func TestPacketRLPEncoding1(t *testing.T) {
	assert := assert.New(t)

	msgBytes := []byte("hello world")
	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
	}

	// ------ EncodeToBytes/DecodeBytes ------

	encodedPacketBytes, err := rlp.EncodeToBytes(packet)
	if err != nil {
		t.Logf("Encode error: %v", err)
	}
	t.Logf("encodedPacketBytes = %v", string(encodedPacketBytes))

	var decodedPacket Packet
	rlp.DecodeBytes(encodedPacketBytes, &decodedPacket)
	t.Logf("decodedPacket: channelID = %v", decodedPacket.ChannelID)
	t.Logf("decodedPacket: Bytes = %v", string(decodedPacket.Bytes))
	t.Logf("decodedPacket: IsEOF = %v", decodedPacket.IsEOF)

	assert.Equal(common.ChannelIDTransaction, decodedPacket.ChannelID)
	assert.Equal(msgBytes, decodedPacket.Bytes)
	assert.Equal(byte(0x01), decodedPacket.IsEOF)
}

func TestPacketRLPEncoding2(t *testing.T) {
	assert := assert.New(t)

	msgBytes := []byte("hello world")
	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
	}

	// ------ Encode/Decode ------

	strBuf := bytes.NewBufferString("")
	err := rlp.Encode(strBuf, packet)
	if err != nil {
		t.Logf("Encode error: %v", err)
	}
	t.Logf("encodedPacketBytes = %v", strBuf)

	var decodedPacket Packet

	rlp.Decode(strBuf, &decodedPacket)
	t.Logf("decodedPacket: channelID = %v", decodedPacket.ChannelID)
	t.Logf("decodedPacket: Bytes = %v", string(decodedPacket.Bytes))
	t.Logf("decodedPacket: IsEOF = %v", decodedPacket.IsEOF)

	assert.Equal(common.ChannelIDTransaction, decodedPacket.ChannelID)
	assert.Equal(msgBytes, decodedPacket.Bytes)
	assert.Equal(byte(0x01), decodedPacket.IsEOF)
}
