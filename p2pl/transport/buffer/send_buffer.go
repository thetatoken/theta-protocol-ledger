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
)

//
// SendBuffer
//

const MaxErrorInARow = 3

type SendBuffer struct {
	workspace  []byte
	wsStartIdx int32

	queue     chan []byte
	queueSize int32

	rawStream   cmn.ReadWriteCloser
	sendMonitor *flowrate.Monitor

	config SendBufferConfig
	seqID  int32

	onError    cmn.ErrorHandler
	errorCount int

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

type SendBufferConfig struct {
	SendRate       int64
	queueCapacity  int
	timeOut        time.Duration
	ChunkBatchSize int64
}

// NewSendBuffer creates a SendBuffer instance for the given config
func NewSendBuffer(config SendBufferConfig, rawStream cmn.ReadWriteCloser, onError cmn.ErrorHandler) SendBuffer {
	return SendBuffer{
		workspace:   make([]byte, 0),
		wsStartIdx:  0,
		queue:       make(chan []byte, config.queueCapacity),
		rawStream:   rawStream,
		sendMonitor: flowrate.New(0, 0),
		config:      config,
		wg:          &sync.WaitGroup{},
		onError:     onError,
		stopped:     true,
	}
}

// GetDefaultSendBufferConfig returns the default config for the SendBuffer
func GetDefaultSendBufferConfig() SendBufferConfig {
	return SendBufferConfig{
		SendRate:       cmn.MaxSendRate,
		queueCapacity:  1,
		timeOut:        10 * time.Second,
		ChunkBatchSize: int64(10),
	}
}

func (sb *SendBuffer) Start(ctx context.Context) bool {
	ctx, cancel := context.WithCancel(ctx)
	sb.ctx = ctx
	sb.cancel = cancel
	sb.stopped = false

	sb.wg.Add(1)
	go sb.sendRoutine()

	return true
}

// Wait suspends the caller goroutine
func (sb *SendBuffer) Wait() {
	sb.wg.Wait()
}

// Stop is called when the SendBuffer stops
func (sb *SendBuffer) Stop() {
	defer func() {
		recover() // Ignore closing closed channel exception.
	}()

	if sb.stopped {
		return
	}
	sb.stopped = true
	sb.workspace = nil
	sb.cancel()
	close(sb.queue)
}

// GetSize returns the size of the SendBuffer. It is goroutine safe
func (sb *SendBuffer) GetSize() int {
	return int(atomic.LoadInt32(&sb.queueSize))
}

// IsEmpty indicates whether the SendBuffer is empty
func (sb *SendBuffer) IsEmpty() bool {
	return (len(sb.workspace) == 0 && len(sb.queue) == 0)
}

// CanInsert return whether more bytes can be inserted into the send buffer.
// It is goroutine safe
func (sb *SendBuffer) CanInsert() bool {
	return sb.GetSize() < sb.config.queueCapacity
}

// Write insert the bytes to queue, and times out after
// the configured timeout. It is goroutine safe
func (sb *SendBuffer) Write(bytes []byte) bool {
	defer sb.recover()

	if sb.stopped {
		return false
	}
	select {
	case sb.queue <- bytes:
		atomic.AddInt32(&sb.queueSize, 1)
		return true
	case <-time.After(sb.config.timeOut):
		return false
	}
}

func (sb *SendBuffer) sendRoutine() {
	defer sb.wg.Done()
	defer sb.recover()

	for {
		select {
		case <-sb.ctx.Done():
			return
		case msg, ok := <-sb.queue:
			if !ok {
				return
			}
			sb.workspace = msg
			for {
				// Block until sendMonitor allows sending
				sb.sendMonitor.Limit(cmn.MaxChunkSize, atomic.LoadInt64(&sb.config.SendRate), true)
				totalBytesSent, success, exhausted := sb.sendChunkBatch()
				sb.sendMonitor.Update(totalBytesSent)

				if exhausted {
					break
				}
				if !success {
					sb.errorCount++
					if sb.errorCount >= MaxErrorInARow {
						return
					}
					break
				} else {
					sb.errorCount = 0
				}
			}
		}

	}
}

func (sb *SendBuffer) sendChunkBatch() (totalBytesSent int, success bool, exhausted bool) {
	totalBytesSent = 0
	chunkBatchSize := sb.config.ChunkBatchSize
	for i := int64(0); i < chunkBatchSize; i++ {
		nonemptyChunk, numBytes, err := sb.sendChunk()
		totalBytesSent += numBytes
		if err != nil {
			log.Errorf("Failed to send chunck batch: %v", err)
			return totalBytesSent, false, !nonemptyChunk
		}
		if !nonemptyChunk {
			return totalBytesSent, true, true // Nothing to be sent
		}
	}

	return totalBytesSent, true, false
}

func (sb *SendBuffer) sendChunk() (nonemptyChunk bool, numBytes int, err error) {
	chunk := sb.emitChunk()
	if chunk.IsEmpty() {
		return false, int(0), nil
	}

	numBytes, err = sb.rawStream.Write(chunk.Bytes())
	if err != nil {
		return true, int(0), err
	}

	return true, numBytes, err
}

// emitChunk emits a chunk extracted from the bytes stored in the workspace
func (sb *SendBuffer) emitChunk() *Chunk {
	seqID := sb.seqID
	if sb.workspace == nil || len(sb.workspace) == 0 {
		return NewEmptyChunk(seqID)
	}

	wsSize := int32(len(sb.workspace))
	if sb.wsStartIdx > wsSize-1 {
		log.Errorf("Invalid sendBuffer state: wsStartIdx = %v, wsSize = %v", sb.wsStartIdx, wsSize)
		return nil
	}

	var isEOF byte
	var chunk *Chunk
	if sb.wsStartIdx+maxChunkPayloadSize < wsSize {
		payloadSize := int32(maxChunkPayloadSize)
		isEOF = byte(0x00)
		chunk = NewChunk(sb.workspace, sb.wsStartIdx, payloadSize, isEOF, seqID)

		sb.wsStartIdx += maxChunkPayloadSize
		sb.seqID++ // increment sequence id
	} else {
		payloadSize := wsSize - sb.wsStartIdx
		isEOF = byte(0x01) // EOF
		chunk = NewChunk(sb.workspace, sb.wsStartIdx, payloadSize, isEOF, seqID)

		// Reset the workspace
		sb.workspace = nil
		sb.wsStartIdx = 0
		sb.seqID = 0                       // reset sequence id
		atomic.AddInt32(&sb.queueSize, -1) // decrement queueSize
	}

	return chunk
}

func (sb *SendBuffer) recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := fmt.Errorf(string(stack))
		if sb.onError != nil {
			sb.onError(err)
		}
		sb.Stop()
	}
}
