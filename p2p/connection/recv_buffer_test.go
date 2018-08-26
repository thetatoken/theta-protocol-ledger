package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func newDefaultRecvBuffer() RecvBuffer {
	defaultConfig := getDefaultRecvBufferConfig()
	recvBuffer := createRecvBuffer(defaultConfig)
	return recvBuffer
}

func TestDefaultRecvBuffer(t *testing.T) {
	assert := assert.New(t)
	drb := newDefaultRecvBuffer()

	msgBytes := []byte("hello world")
	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
	}

	recvBytes, success := drb.receivePacket(packet)
	assert.True(success)
	assert.Equal(msgBytes, recvBytes)
}

func TestRecvMultipleMessages(t *testing.T) {
	assert := assert.New(t)
	drb := newDefaultRecvBuffer()

	msgBytes1 := []byte("hello ")
	packet1 := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes1,
		IsEOF:     byte(0x00),
	}

	msgBytes2 := []byte("world")
	packet2 := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes2,
		IsEOF:     byte(0x01),
	}

	msgBytes3 := []byte("You've got ")
	packet3 := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes3,
		IsEOF:     byte(0x00),
	}

	msgBytes4 := []byte("an ")
	packet4 := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes4,
		IsEOF:     byte(0x00),
	}

	msgBytes5 := []byte("email")
	packet5 := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes5,
		IsEOF:     byte(0x01),
	}

	// ---------- Check messageA ----------

	messageA := string(msgBytes1) + string(msgBytes2)

	recvBytes1, success := drb.receivePacket(packet1)
	assert.True(success)
	assert.Nil(recvBytes1)

	recvBytes2, success := drb.receivePacket(packet2)
	assert.True(success)
	assert.NotNil(recvBytes2)

	assert.Equal(messageA, string(recvBytes2))

	// ---------- Check messageB ----------

	messageB := string(msgBytes3) + string(msgBytes4) + string(msgBytes5)

	recvBytes3, success := drb.receivePacket(packet3)
	assert.True(success)
	assert.Nil(recvBytes3)

	recvBytes4, success := drb.receivePacket(packet4)
	assert.True(success)
	assert.Nil(recvBytes4)

	recvBytes5, success := drb.receivePacket(packet5)
	assert.True(success)
	assert.NotNil(recvBytes5)

	assert.Equal(messageB, string(recvBytes5))
}

func TestRecvExtraLongMessage(t *testing.T) {
	assert := assert.New(t)
	drb := newDefaultRecvBuffer()

	msgBytes := []byte("0123456789")
	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x00),
	}

	var success bool
	var recvBytes []byte
	for i := 0; i < 409; i++ {
		recvBytes, success = drb.receivePacket(packet)
		assert.True(success)
		assert.Nil(recvBytes)
	}

	// The following receivePacket exceeds the workspaceCapacity, should fail
	recvBytes, success = drb.receivePacket(packet)
	assert.False(success)
	assert.Nil(recvBytes)
}
