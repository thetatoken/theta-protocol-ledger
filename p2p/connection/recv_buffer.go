package connection

import (
	"fmt"
	"time"

	"github.com/thetatoken/theta/common"
)

type RecvBuffer struct {
	workspace []byte

	config  RecvBufferConfig
	chanSeq uint

	packetCount           int
	messageStartedAt      time.Time
	messageLastProgressAt time.Time
}

type RecvBufferConfig struct {
	workspaceCapacity int
	maxMessageSize    int
	messageTimeout    time.Duration
}

// createRecvBuffer creates a RecvBuffer instance for the given config
func createRecvBuffer(config RecvBufferConfig) RecvBuffer {
	return RecvBuffer{
		workspace: make([]byte, 0, config.workspaceCapacity),
		config:    config,
	}
}

// getDefaultRecvBufferConfig returns the default config for the RecvBuffer
func getDefaultRecvBufferConfig() RecvBufferConfig {
	return RecvBufferConfig{
		workspaceCapacity: 4 * 1024, // 4 KB
		maxMessageSize:    1024 * 1024,
		messageTimeout:    frameReadTimeout,
	}
}

// receivePacket handles incoming msgPackets. It returns a msg bytes if msg is
// complete (i.e. ends with EOF). It is not goroutine safe
func (rb *RecvBuffer) receivePacket(packet *Packet) ([]byte, bool) {
	bytes, success, err := rb.receivePacketWithError(packet)
	if err != nil {
		logger.Warnf("RecvBuffer failed to receive packet: %v", err)
	}
	return bytes, success
}

func (rb *RecvBuffer) receivePacketWithError(packet *Packet) ([]byte, bool, error) {
	if packet == nil {
		return nil, false, fmt.Errorf("nil packet")
	}
	if packet.IsEOF != byte(0x00) && packet.IsEOF != byte(0x01) {
		rb.reset()
		return nil, false, fmt.Errorf("invalid EOF marker %v", packet.IsEOF)
	}
	if len(packet.Bytes) > maxPayloadSize {
		rb.reset()
		return nil, false, fmt.Errorf("packet payload exceeds limit, size %v, limit %v", len(packet.Bytes), maxPayloadSize)
	}
	// Note: We do NOT need to worry about the order of the packets.
	//       TCP guarantees that if bytes arrive, they will be in the
	//       order they were sent, as long as the TCP connection stays open.
	//       But we do need to check if there's any missing packet
	if rb.chanSeq != packet.SeqID {
		rb.reset()
		return nil, false, fmt.Errorf("packet seqID mismatch, expected %v, actual %v", rb.chanSeq, packet.SeqID)
	}
	maxMessageSize := rb.config.maxMessageSize
	if maxMessageSize <= 0 {
		maxMessageSize = rb.config.workspaceCapacity
	}
	now := time.Now()
	if rb.packetCount == 0 {
		rb.messageStartedAt = now
		rb.messageLastProgressAt = now
	}
	deadline := rb.reassemblyDeadline()
	if !deadline.IsZero() && !now.Before(deadline) {
		rb.reset()
		return nil, false, fmt.Errorf("message reassembly exceeded deadline %v", deadline)
	}
	maxPacketCount := maxMessageSize/maxPayloadSize + 2
	if rb.packetCount >= maxPacketCount {
		rb.reset()
		return nil, false, fmt.Errorf("message exceeds packet count limit %v", maxPacketCount)
	}
	if len(packet.Bytes) > maxMessageSize || len(rb.workspace) > maxMessageSize-len(packet.Bytes) {
		rb.reset()
		return nil, false, fmt.Errorf("message exceeds receive limit, current %v, packet %v, limit %v",
			len(rb.workspace), len(packet.Bytes), maxMessageSize)
	}

	rb.workspace = append(rb.workspace, packet.Bytes...)
	rb.packetCount++
	rb.messageLastProgressAt = now
	if packet.IsEOF == byte(0x01) {
		bytes := rb.workspace

		// clear the slice without re-allocating.
		// http://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
		//   suggests this could be a memory leak, but we might as well keep the memory for the channel until it closes,
		//	at which point the recving slice stops being used and should be garbage collected
		rb.reset()

		return bytes, true, nil
	}

	rb.chanSeq++
	return nil, true, nil
}

func (rb *RecvBuffer) setMaxMessageSize(maxMessageSize int) {
	rb.config.maxMessageSize = maxMessageSize
}

func (rb *RecvBuffer) reassemblyDeadline() time.Time {
	return common.P2PReassemblyDeadline(rb.messageStartedAt, rb.messageLastProgressAt,
		rb.config.maxMessageSize, rb.config.messageTimeout)
}

func (rb *RecvBuffer) reset() {
	if cap(rb.workspace) > rb.config.workspaceCapacity {
		rb.workspace = make([]byte, 0, rb.config.workspaceCapacity)
	} else {
		rb.workspace = rb.workspace[:0]
	}
	rb.chanSeq = 0
	rb.packetCount = 0
	rb.messageStartedAt = time.Time{}
	rb.messageLastProgressAt = time.Time{}
}
