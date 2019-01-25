package connection

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
)

func TestDefaultRecvBuffer(t *testing.T) {
	assert := assert.New(t)
	drb := newTestDefaultRecvBuffer()

	msgBytes := []byte("hello world")
	packet := &Packet{
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
	drb := newTestDefaultRecvBuffer()

	msgBytes1 := []byte("hello ")
	packet1 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes1,
		IsEOF:     byte(0x00),
		SeqID:     0,
	}

	msgBytes2 := []byte("world")
	packet2 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes2,
		IsEOF:     byte(0x01),
		SeqID:     1,
	}

	msgBytes3 := []byte("You've got ")
	packet3 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes3,
		IsEOF:     byte(0x00),
		SeqID:     0,
	}

	msgBytes4 := []byte("an ")
	packet4 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes4,
		IsEOF:     byte(0x00),
		SeqID:     1,
	}

	msgBytes5 := []byte("email")
	packet5 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes5,
		IsEOF:     byte(0x01),
		SeqID:     2,
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
	drb := newTestDefaultRecvBuffer()

	expectedMsgBytes := []byte{}
	msgBytes := []byte("01234567890123450123456789012345012345678901234501234567890123450123456789012345012345678901234501234567890123450123456789012345") // 128 Bytes
	packet := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x00),
	}

	var success bool
	var recvBytes []byte
	var i uint
	for ; i < 32767; i++ {
		packet.SeqID = i
		recvBytes, success = drb.receivePacket(packet)
		assert.True(success)
		assert.Nil(recvBytes)

		expectedMsgBytes = append(expectedMsgBytes, packet.Bytes...)
	}

	endPacket := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
		SeqID:     i,
	}
	aggregatedBytes, success := drb.receivePacket(endPacket)
	assert.True(success)
	assert.NotNil(aggregatedBytes)
	expectedMsgBytes = append(expectedMsgBytes, endPacket.Bytes...)

	t.Logf("Length of the expectedMsgBytes: %v", len(expectedMsgBytes))
	t.Logf("Length of the aggregatedBytes:  %v", len(aggregatedBytes))

	assert.Equal(4194304, len(expectedMsgBytes)) // should be 4 MB
	assert.Equal(4194304, len(aggregatedBytes))  // should be 4 MB
	sameBytes := (bytes.Compare(expectedMsgBytes, aggregatedBytes) == 0)
	assert.True(sameBytes)
}

// --------------- Test Utilities --------------- //

func newTestDefaultRecvBuffer() RecvBuffer {
	defaultConfig := getDefaultRecvBufferConfig()
	recvBuffer := createRecvBuffer(defaultConfig)
	return recvBuffer
}
