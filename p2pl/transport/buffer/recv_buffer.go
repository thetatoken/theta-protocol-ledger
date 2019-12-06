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
// SendBuffer
//

type RecvBuffer struct {
	workspace []byte

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
		workspace:   make([]byte, 0, config.workspaceCapacity),
		queue:       make(chan []byte, config.queueCapacity),
		rawStream:   rawStream,
		recvMonitor: flowrate.New(0, 0),
		config:      config,
		wg:          &sync.WaitGroup{},
		onError:     onError,
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
	rb.workspace = nil
	rb.cancel()
}

// Read blocks until a message can be retrived from the queue
func (rb *RecvBuffer) Read() []byte {
	msg := <-rb.queue
	atomic.AddInt32(&rb.queueSize, -1)
	return msg
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
	}
}

// extractChunks extract the chunks from the bytes read from the stream. Note that
func (rb *RecvBuffer) extractChunks(bytes []byte, numBytesRead int) {
	const int32Bytes = 4
	var rolloverBytes, precedingBytes []byte
	for start := 0; start < numBytesRead; {
		var chunkBytes []byte
		var increment int
		rolloverLen := len(rolloverBytes)
		rolloverCap := cap(rolloverBytes)

		if start == 0 && rolloverLen > 0 {
			residueLen := rolloverCap - rolloverLen
			if residueLen > numBytesRead {
				rolloverBytes = rolloverBytes[:rolloverLen+numBytesRead]
				copy(rolloverBytes[rolloverLen:rolloverLen+numBytesRead], bytes[:numBytesRead])
				rb.recvMonitor.Update(numBytesRead) // ?
				break
			}

			rolloverBytes = rolloverBytes[:rolloverCap]
			copy(rolloverBytes[rolloverLen:rolloverCap], bytes[:residueLen])
			chunkBytes = rolloverBytes
			increment = residueLen
		} else {
			if start+isEOFOffset > numBytesRead {
				precedingBytes = make([]byte, numBytesRead-start, isEOFOffset)
				copy(precedingBytes, bytes[start:numBytesRead])
				break
			}

			var payloadSize int
			precedingLen := len(precedingBytes)
			if precedingLen > 0 {
				precedingBytes = precedingBytes[:isEOFOffset]
				copy(precedingBytes[precedingLen:], bytes[:isEOFOffset-precedingLen])
				payloadSize = int(int32FromBytes(precedingBytes[payloadSizeOffset : payloadSizeOffset+int32Bytes]))
				start -= precedingLen
			} else {
				payloadSize = int(int32FromBytes(bytes[start+payloadSizeOffset : start+payloadSizeOffset+int32Bytes]))
			}

			chunkSize := headerSize + payloadSize

			if start+chunkSize > numBytesRead {
				rolloverBytes = make([]byte, numBytesRead-start, chunkSize) // memory usage: will garbage collect previous rolloverBytes?
				copy(rolloverBytes, bytes[start:numBytesRead])

				rb.recvMonitor.Update(numBytesRead - start) //?
				break
			}

			if start < 0 {
				chunkBytes = append(precedingBytes, bytes[isEOFOffset-precedingLen:chunkSize-precedingLen]...)
			} else {
				chunkBytes = bytes[start : start+chunkSize]
			}
			increment = chunkSize
		}

		chunk, err := NewChunkFromRawBytes(chunkBytes)
		if err == nil {
			rb.recvMonitor.Update(increment)

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

		rolloverBytes = nil //rolloverBytes[:0]
		precedingBytes = nil
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
		rb.Stop()
	}
}
