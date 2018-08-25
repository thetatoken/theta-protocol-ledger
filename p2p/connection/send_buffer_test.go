package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func newDefaultSendBuffer() SendBuffer {
	defaultConfig := getDefaultSendBufferConfig()
	sendBuffer := createSendBuffer(defaultConfig)
	return sendBuffer
}

func TestDefaultSendBuffer(t *testing.T) {
	assert := assert.New(t)
	dsb := newDefaultSendBuffer()

	assert.True(dsb.isEmpty())
	assert.Equal(0, dsb.getSize())
	assert.True(dsb.canInsert())

	msgBytes := []byte("hello world")
	success := dsb.insert(msgBytes)
	assert.True(success)

	assert.False(dsb.isEmpty())
	assert.Equal(1, dsb.getSize())
	assert.False(dsb.canInsert())

	packet := dsb.emitPacket(common.ChannelIDTransaction)

	assert.True(dsb.isEmpty())
	assert.False(packet.isEmpty())
	assert.Equal(msgBytes, packet.Bytes)
	assert.Equal(byte(0x1), packet.IsEOF)
}

func TestSendLongMessage(t *testing.T) {
	assert := assert.New(t)
	dsb := newDefaultSendBuffer()

	// prepare a 3000-byte long []byte
	var msgStr string
	for i := 0; i < 300; i++ {
		msgStr = msgStr + "0123456789"
	}
	msgBytes := []byte(msgStr)

	success := dsb.insert(msgBytes)
	assert.True(success)

	assert.False(dsb.isEmpty())
	assert.Equal(1, dsb.getSize())
	assert.False(dsb.canInsert())

	packet1 := dsb.emitPacket(common.ChannelIDTransaction)
	assert.False(dsb.isEmpty())
	assert.Equal(1, dsb.getSize())
	assert.False(packet1.isEmpty())
	assert.Equal(byte(0x0), packet1.IsEOF)

	packet2 := dsb.emitPacket(common.ChannelIDTransaction)
	assert.False(dsb.isEmpty())
	assert.Equal(1, dsb.getSize())
	assert.False(packet2.isEmpty())
	assert.Equal(byte(0x0), packet2.IsEOF)

	packet3 := dsb.emitPacket(common.ChannelIDTransaction)
	assert.True(dsb.isEmpty())
	assert.Equal(0, dsb.getSize())
	assert.False(packet3.isEmpty())
	assert.Equal(byte(0x1), packet3.IsEOF)

	assembledMsgStr := string(packet1.Bytes) + string(packet2.Bytes) + string(packet3.Bytes)
	assert.Equal(msgStr, assembledMsgStr)
}
