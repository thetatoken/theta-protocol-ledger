package buffer

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	cmn "github.com/thetatoken/theta/p2pl/common"
	"github.com/thetatoken/theta/p2pl/transport/buffer/flowrate"

	"github.com/libp2p/go-libp2p-core/network"
)

//
// RecvBuffer
//

type RecvBuffer struct {
	workspace []byte

	rolloverBytes  []byte
	precedingBytes []byte
	// rolloverLen    int
	// rolloverCap    int
	// precedingLen   int

	queue     chan []byte
	queueSize int32

	rawStream   cmn.ReadWriteCloser
	recvMonitor *flowrate.Monitor

	config RecvBufferConfig
	seqID  int32

	onError cmn.ErrorHandler

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

type RecvBufferConfig struct {
	workspaceCapacity int
	RecvRate          int64
	queueCapacity     int
	timeOut           time.Duration
}

// NewRecvBuffer creates a RecvBuffer instance for the given config
func NewRecvBuffer(config RecvBufferConfig, rawStream cmn.ReadWriteCloser, onError cmn.ErrorHandler) RecvBuffer {
	return RecvBuffer{
		workspace: make([]byte, 0, config.workspaceCapacity),

		rolloverBytes:  make([]byte, 0, cmn.MaxChunkSize),
		precedingBytes: make([]byte, 0, isEOFOffset),
		// rolloverLen:    0,
		// rolloverCap:    0,
		// precedingLen:   0,

		queue:       make(chan []byte, config.queueCapacity),
		rawStream:   rawStream,
		recvMonitor: flowrate.New(0, 0),
		config:      config,
		wg:          &sync.WaitGroup{},
		onError:     onError,
		stopped:     true,
	}
}

// GetDefaultRecvBufferConfig returns the default config for the RecvBuffer
func GetDefaultRecvBufferConfig() RecvBufferConfig {
	return RecvBufferConfig{
		workspaceCapacity: cmn.MaxChunkSize,
		RecvRate:          cmn.MaxRecvRate, // 64 Mbps
		queueCapacity:     1,
		timeOut:           10 * time.Second,
	}
}

func (rb *RecvBuffer) Start(ctx context.Context) bool {
	ctx, cancel := context.WithCancel(ctx)
	rb.ctx = ctx
	rb.cancel = cancel
	rb.stopped = false

	rb.wg.Add(1)
	go rb.recvRoutine()

	return true
}

// Wait suspends the caller goroutine
func (rb *RecvBuffer) Wait() {
	rb.wg.Wait()
}

// Stop is called when the RecvBuffer stops
func (rb *RecvBuffer) Stop() {
	defer func() {
		recover() // Ignore closing closed channel exception.
	}()

	if rb.stopped {
		return
	}
	rb.stopped = true
	rb.workspace = nil

	rb.rolloverBytes = nil
	rb.precedingBytes = nil

	rb.cancel()
	close(rb.queue)
}

// Read blocks until a message can be retrived from the queue
func (rb *RecvBuffer) Read() ([]byte, error) {
	if rb.stopped {
		return nil, fmt.Errorf("RecvBuffer is already stopped")
	}
	msg, ok := <-rb.queue
	if !ok {
		return nil, fmt.Errorf("queue closed")
	}
	atomic.AddInt32(&rb.queueSize, -1)
	return msg, nil
}

// GetSize returns the size of the SendBuffer. It is goroutine safe
func (rb *RecvBuffer) GetSize() int {
	return int(atomic.LoadInt32(&rb.queueSize))
}

// TODO: protection for attacks, e.g. send a very large message to peers
func (rb *RecvBuffer) recvRoutine() {
	defer rb.wg.Done()
	defer rb.recover()

	bytes := make([]byte, cmn.MaxChunkSize)
	defer func() { bytes = nil }()

	for {
		select {
		case <-rb.ctx.Done():
			return
		default:
		}

		// Block until recvMonitor allows reading
		rb.recvMonitor.Limit(cmn.MaxChunkSize, atomic.LoadInt64(&rb.config.RecvRate), true)
		numBytesRead, err := rb.rawStream.Read(bytes)
		if err != nil {
			rawStream := rb.rawStream.(network.Stream)
			log.Warnf("Raw stream %v read error: %v", rawStream.Conn().RemotePeer(), err)
			break
		}

		rb.extractChunks(bytes, numBytesRead)
		rb.recvMonitor.Update(numBytesRead)
	}
}

// extractChunks extract the chunks from the bytes read from the stream. Note that the bytes
// read from the stream might contain multiple chunks or partial chunk from the sender. Hence
// we need to handle rollover and preceding bytes properly
func (rb *RecvBuffer) extractChunks(bytes []byte, numBytesRead int) {
	const int32Bytes = 4
	for start := 0; start < numBytesRead; {
		var chunkBytes []byte
		var increment int
		rolloverLen := len(rb.rolloverBytes)
		rolloverCap := cap(rb.rolloverBytes)

		if start == 0 && rolloverLen > 0 {
			residueLen := rolloverCap - rolloverLen
			if residueLen > numBytesRead {
				rb.rolloverBytes = rb.rolloverBytes[:rolloverLen+numBytesRead]
				copy(rb.rolloverBytes[rolloverLen:rolloverLen+numBytesRead], bytes[:numBytesRead])
				break
			}

			rb.rolloverBytes = rb.rolloverBytes[:rolloverCap]
			copy(rb.rolloverBytes[rolloverLen:rolloverCap], bytes[:residueLen])
			chunkBytes = rb.rolloverBytes
			increment = residueLen
		} else {
			if start+isEOFOffset > numBytesRead {
				rb.precedingBytes = make([]byte, numBytesRead-start, isEOFOffset)
				copy(rb.precedingBytes, bytes[start:numBytesRead])
				break
			}

			var payloadSize int
			precedingLen := len(rb.precedingBytes)
			if precedingLen > 0 {
				rb.precedingBytes = rb.precedingBytes[:isEOFOffset]
				copy(rb.precedingBytes[precedingLen:], bytes[:isEOFOffset-precedingLen])
				payloadSize = int(int32FromBytes(rb.precedingBytes[payloadSizeOffset : payloadSizeOffset+int32Bytes]))
				start -= precedingLen
			} else {
				payloadSize = int(int32FromBytes(bytes[start+payloadSizeOffset : start+payloadSizeOffset+int32Bytes]))
			}

			chunkSize := headerSize + payloadSize

			if start+chunkSize > numBytesRead {
				rb.rolloverBytes = make([]byte, numBytesRead-start, chunkSize) // memory usage: will garbage collect previous rolloverBytes?
				copy(rb.rolloverBytes, bytes[start:numBytesRead])
				break
			}

			if start < 0 {
				chunkBytes = append(rb.precedingBytes, bytes[isEOFOffset-precedingLen:chunkSize-precedingLen]...)
			} else {
				chunkBytes = bytes[start : start+chunkSize]
			}
			increment = chunkSize
		}

		chunk, err := NewChunkFromRawBytes(chunkBytes)
		if err == nil {
			completeMessage, success := rb.aggregateChunk(chunk)
			if success {
				if completeMessage != nil {
					rb.queue <- completeMessage
					atomic.AddInt32(&rb.queueSize, 1)
				}
			}
		} else {
			log.Errorf("RecvBuffer failed to create new chunk from raw bytes: %v", err)
		}

		rb.rolloverBytes = nil //rolloverBytes[:0]
		rb.precedingBytes = nil
		start += increment
	}
}

// aggregateChunk aggregates incoming chunks. It returns the message bytes if the message is
// complete (i.e. ends with EOF). It is not goroutine safe
func (rb *RecvBuffer) aggregateChunk(chunk *Chunk) (completeMessage []byte, success bool) {
	// Note: We do NOT need to worry about the order of the chunks.
	//       TCP guarantees that if bytes arrive, they will be in the
	//       order they were sent, as long as the TCP connection stays open.
	//       But we do need to check if there's any missing chunk
	if rb.seqID != chunk.SeqID() {
		log.Warnf("chunk seqID mismatch. expected: %v, actual: %v", rb.seqID, chunk.SeqID())
		return nil, false
	}

	chunkPayload := chunk.Payload()
	log.Debugf("Aggregate chunk: payloadSize = %v, seqID = %v, isEOF = %v", len(chunkPayload), chunk.SeqID(), chunk.IsEOF())

	rb.workspace = append(rb.workspace, chunkPayload...)
	if chunk.IsEOF() {
		msgSize := len(rb.workspace)
		completeMessage := make([]byte, msgSize)
		copy(completeMessage, rb.workspace)

		// clear the slice without re-allocating.
		// http://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
		//   suggests this could be a memory leak, but we might as well keep the memory for the buffer until it closes,
		//	at which point the recving slice stops being used and should be garbage collected
		rb.workspace = rb.workspace[:0] // make([]byte, 0, rb.config.workspaceCapacity)
		rb.seqID = 0

		return completeMessage, true
	}

	rb.seqID++
	return nil, true
}

func (rb *RecvBuffer) recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := fmt.Errorf(string(stack))
		if rb.onError != nil {
			rb.onError(err)
		}
	}
	rb.Stop()
}

/*
// ---- Memory optimized version (WIP) -----
// extractChunks extract the chunks from the bytes read from the stream. Note that the bytes
// read from the stream might contain multiple chunks or partial chunk from the sender. Hence
// we need to handle rollover and preceding bytes properly
func (rb *RecvBuffer) extractChunks(bytes []byte, numBytesRead int) {
	const int32Bytes = 4

	for start := 0; start < numBytesRead; {
		var chunkBytes []byte
		var increment int
		// rolloverLen := len(rb.rolloverBytes)
		// rolloverCap := cap(rb.rolloverBytes)

		if start == 0 && rb.rolloverLen > 0 {
			residueLen := rb.rolloverCap - rb.rolloverLen
			if residueLen > numBytesRead {
				rb.rolloverBytes = rb.rolloverBytes[:rb.rolloverLen+numBytesRead]
				copy(rb.rolloverBytes[rb.rolloverLen:rb.rolloverLen+numBytesRead], bytes[:numBytesRead])
				rb.rolloverLen += numBytesRead
				break
			}

			rb.rolloverBytes = rb.rolloverBytes[:rb.rolloverCap]
			copy(rb.rolloverBytes[rb.rolloverLen:rb.rolloverCap], bytes[:residueLen])
			rb.rolloverLen = rb.rolloverCap
			chunkBytes = rb.rolloverBytes
			increment = residueLen
		} else {
			if start+isEOFOffset > numBytesRead {
				//rb.precedingBytes = make([]byte, numBytesRead-start, isEOFOffset)
				rb.precedingBytes = rb.precedingBytes[:0]
				copy(rb.precedingBytes, bytes[start:numBytesRead])
				rb.precedingLen = numBytesRead - start
				break
			}

			var payloadSize int
			//precedingLen := len(rb.precedingBytes)
			if rb.precedingLen > 0 {
				rb.precedingBytes = rb.precedingBytes[:isEOFOffset]
				copy(rb.precedingBytes[rb.precedingLen:], bytes[:isEOFOffset-rb.precedingLen])
				payloadSize = int(int32FromBytes(rb.precedingBytes[payloadSizeOffset : payloadSizeOffset+int32Bytes]))
				start -= rb.precedingLen
				rb.precedingLen = isEOFOffset
			} else {
				payloadSize = int(int32FromBytes(bytes[start+payloadSizeOffset : start+payloadSizeOffset+int32Bytes]))
			}

			chunkSize := headerSize + payloadSize

			if start+chunkSize > numBytesRead {
				//rb.rolloverBytes = make([]byte, numBytesRead-start, chunkSize) // memory usage: will garbage collect previous rolloverBytes?
				rb.rolloverBytes = rb.rolloverBytes[:0]

				copy(rb.rolloverBytes, bytes[start:numBytesRead])
				rb.rolloverLen = numBytesRead - start
				rb.rolloverCap = chunkSize
				break
			}

			if start < 0 {
				chunkBytes = append(rb.precedingBytes, bytes[isEOFOffset-rb.precedingLen:chunkSize-rb.precedingLen]...)
			} else {
				chunkBytes = bytes[start : start+chunkSize]
			}
			increment = chunkSize
		}

		chunk, err := NewChunkFromRawBytes(chunkBytes)
		if err == nil {
			completeMessage, success := rb.aggregateChunk(chunk)
			if success {
				if completeMessage != nil {
					rb.queue <- completeMessage
					atomic.AddInt32(&rb.queueSize, 1)
				}
			}
		} else {
			log.Errorf("RecvBuffer failed to create new chunk from raw bytes: %v", err)
		}

		// rb.rolloverBytes = nil
		// rb.precedingBytes = nil
		rb.rolloverBytes = rb.rolloverBytes[:0]
		rb.precedingBytes = rb.precedingBytes[:0]
		rb.rolloverLen = 0
		rb.rolloverCap = 0
		rb.precedingLen = 0

		start += increment
	}
}
*/
