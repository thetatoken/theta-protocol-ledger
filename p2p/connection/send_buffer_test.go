package connection

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func TestDefaultSendBuffer(t *testing.T) {
	assert := assert.New(t)
	dsb := newTestDefaultSendBuffer()

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
	assert.True(dsb.canInsert())
}

func TestSendLongMessage(t *testing.T) {
	assert := assert.New(t)
	dsb := newTestDefaultSendBuffer()

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
	assert.False(dsb.canInsert())

	packet2 := dsb.emitPacket(common.ChannelIDTransaction)
	assert.False(dsb.isEmpty())
	assert.Equal(1, dsb.getSize())
	assert.False(packet2.isEmpty())
	assert.Equal(byte(0x0), packet2.IsEOF)
	assert.False(dsb.canInsert())

	packet3 := dsb.emitPacket(common.ChannelIDTransaction)
	assert.True(dsb.isEmpty())
	assert.Equal(0, dsb.getSize())
	assert.False(packet3.isEmpty())
	assert.Equal(byte(0x1), packet3.IsEOF)
	assert.True(dsb.canInsert())

	assembledMsgStr := string(packet1.Bytes) + string(packet2.Bytes) + string(packet3.Bytes)
	assert.Equal(msgStr, assembledMsgStr)
}

func TestSequentialSendMultipleMessages(t *testing.T) {
	assert := assert.New(t)
	dsb := newTestDefaultSendBuffer()

	for i := 0; i < 16; i++ {
		msgBytes := []byte("cool stuff!")
		success := dsb.insert(msgBytes)
		assert.True(success)
		assert.False(dsb.canInsert())
		assert.Equal(1, dsb.getSize())

		packet := dsb.emitPacket(common.ChannelIDTransaction)
		assert.Equal(msgBytes, packet.Bytes)
		assert.Equal(byte(0x01), packet.IsEOF)
	}
}

func TestConcurrentSendMultipleMessages(t *testing.T) {
	assert := assert.New(t)
	dsb := newTestDefaultSendBuffer()

	msgBytesBase := []byte(" - cool stuff!")
	numMsgs := 16

	sendSuccesses := make(chan bool, numMsgs)
	go func(sendSuccesses chan bool) { // send routine
		for i := 0; i < numMsgs; i++ {
			ithMsgBytes := []byte(strconv.Itoa(i) + string(msgBytesBase))
			success := dsb.insert(ithMsgBytes)
			sendSuccesses <- success
		}
	}(sendSuccesses)

	emitBytesList := make(chan []byte, numMsgs)
	emitEOFs := make(chan byte, numMsgs)
	go func(emitBytesList chan []byte, emitEOFs chan byte) { // emit packet routine
		for {
			packet := dsb.emitPacket(common.ChannelIDTransaction)
			if packet.Bytes != nil {
				emitBytesList <- packet.Bytes
				emitEOFs <- packet.IsEOF
			}
		}
	}(emitBytesList, emitEOFs)

	for i := 0; i < numMsgs; i++ {
		sendSuc := <-sendSuccesses
		assert.True(sendSuc)

		emitBytes := <-emitBytesList
		ithMsgBytes := []byte(strconv.Itoa(i) + string(msgBytesBase))
		assert.Equal(ithMsgBytes, emitBytes)
		t.Logf("emitted bytes: %v", string(emitBytes))

		emitEOF := <-emitEOFs
		assert.Equal(byte(0x01), emitEOF)
	}
}

func TestAttemptInsert(t *testing.T) {
	assert := assert.New(t)
	dsb := newTestDefaultSendBuffer()
	assert.True(dsb.canInsert())

	msgBytes := []byte("hello world")

	success := dsb.insert(msgBytes)
	assert.True(success)
	assert.False(dsb.canInsert())
	assert.Equal(1, dsb.getSize())

	success = dsb.attemptInsert(msgBytes)
	assert.False(success)
	assert.False(dsb.canInsert())
	assert.Equal(1, dsb.getSize())

	packet := dsb.emitPacket(common.ChannelIDTransaction)
	assert.Equal(msgBytes, packet.Bytes)
	assert.Equal(byte(0x01), packet.IsEOF)

	success = dsb.attemptInsert(msgBytes)
	assert.True(success)
	assert.False(dsb.canInsert())
	assert.Equal(1, dsb.getSize())

	packet = dsb.emitPacket(common.ChannelIDTransaction)
	assert.Equal(msgBytes, packet.Bytes)
	assert.Equal(byte(0x01), packet.IsEOF)
}

// --------------- Test Utilities --------------- //

func newTestDefaultSendBuffer() SendBuffer {
	defaultConfig := getDefaultSendBufferConfig()
	sendBuffer := createSendBuffer(defaultConfig)
	return sendBuffer
}
