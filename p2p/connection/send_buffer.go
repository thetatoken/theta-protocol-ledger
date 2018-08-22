package connection

import (
	"sync/atomic"
	"time"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/math"
)

type SendBuffer struct {
	workspace []byte
	queue     chan []byte
	queueSize int32

	config SendBufferConfig
}

type SendBufferConfig struct {
	queueCapacity int
	timeOut       time.Duration
}

// createSendBuffer creates a SendBuffer instance for the given config
func createSendBuffer(config SendBufferConfig) SendBuffer {
	return SendBuffer{
		queue:  make(chan []byte, config.queueCapacity),
		config: config,
	}
}

// getDefaultSendBufferConfig returns the default config for the SendBuffer
func getDefaultSendBufferConfig() SendBufferConfig {
	return SendBufferConfig{
		queueCapacity: 1,
		timeOut:       10 * time.Second,
	}
}

// getSize returns the size of the SendBuffer. It is goroutine safe
func (sb *SendBuffer) getSize() int {
	return int(atomic.LoadInt32(&sb.queueSize))
}

// isEmpty indicates whether the SendBuffer is empty
func (sb *SendBuffer) isEmpty() bool {
	return (len(sb.workspace) == 0 && len(sb.queue) == 0)
}

// canInsert return whether more bytes can be inserted into the send buffer.
// It is goroutine safe
func (sb *SendBuffer) canInsert() bool {
	return sb.getSize() < sb.config.queueCapacity
}

// Insert insert the bytes to queue, and times out after after
// the configured timeout. It is goroutine safe
func (sb *SendBuffer) insert(bytes []byte) bool {
	select {
	case sb.queue <- bytes:
		atomic.AddInt32(&sb.queueSize, 1)
		return true
	case <-time.After(sb.config.timeOut):
		return false
	}
}

// attemptInsert attemps to insert bytes into the queue. It is a
// non-blocking call. It is goroutine safe
func (sb *SendBuffer) attemptInsert(bytes []byte) bool {
	select {
	case sb.queue <- bytes:
		atomic.AddInt32(&sb.queueSize, 1)
		return true
	default:
		return false
	}
}

// EmitPacket emits a packet extracted from the bytes stored in the workspace
func (sb *SendBuffer) emitPacket(channelID common.ChannelIDEnum) Packet {
	if sb.workspace == nil {
		return Packet{
			ChannelID: channelID,
			Bytes:     nil,
			IsEOF:     byte(0x01),
		}
	}

	if len(sb.workspace) == 0 && len(sb.queue) > 0 {
		sb.workspace = <-sb.queue // update workspace if necessary
	}

	bytes := sb.workspace[:math.MinInt(maxPayloadSize, len(sb.workspace))]
	isEOF := byte(0x00)
	if len(sb.workspace) <= maxPayloadSize {
		isEOF = byte(0x01) // EOF
		sb.workspace = nil
		atomic.AddInt32(&sb.queueSize, -1) // decrement queueSize
	} else {
		isEOF = byte(0x00)
		sb.workspace = sb.workspace[math.MinInt(maxPayloadSize, len(sb.workspace)):]
	}

	packet := Packet{
		ChannelID: channelID,
		Bytes:     bytes,
		IsEOF:     isEOF,
	}

	return packet
}
