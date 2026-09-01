package buffer

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	rootcmn "github.com/thetatoken/theta/common"
	cmn "github.com/thetatoken/theta/p2pl/common"
	"github.com/thetatoken/theta/p2pl/transport/buffer/flowrate"
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

	chunkCount               int
	reassemblyStartedAt      time.Time
	reassemblyLastProgressAt time.Time

	onError cmn.ErrorHandler

	// Life cycle
	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	done     chan struct{}
	stopOnce sync.Once
	stopped  uint32
}

type RecvBufferConfig struct {
	workspaceCapacity int
	MaxMessageSize    int
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
		done:        make(chan struct{}),
		stopped:     1,
	}
}

// GetDefaultRecvBufferConfig returns the default config for the RecvBuffer
func GetDefaultRecvBufferConfig() RecvBufferConfig {
	return RecvBufferConfig{
		workspaceCapacity: cmn.MaxChunkSize,
		MaxMessageSize:    cmn.MaxNormalMessageSize,
		RecvRate:          cmn.MaxRecvRate, // 64 Mbps
		queueCapacity:     1,
		timeOut:           10 * time.Second,
	}
}

func (rb *RecvBuffer) Start(ctx context.Context) bool {
	ctx, cancel := context.WithCancel(ctx)
	rb.ctx = ctx
	rb.cancel = cancel
	atomic.StoreUint32(&rb.stopped, 0)

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
	rb.stopOnce.Do(func() {
		atomic.StoreUint32(&rb.stopped, 1)
		if rb.cancel != nil {
			rb.cancel()
		}
		close(rb.done)
	})
}

// Read blocks until a message can be retrived from the queue
func (rb *RecvBuffer) Read() ([]byte, error) {
	if msg, ok := rb.tryReadQueuedMessage(); ok {
		return msg, nil
	}
	if atomic.LoadUint32(&rb.stopped) != 0 {
		// A complete message may have been queued immediately before EOF stopped
		// the receive routine. Preserve that final validated message.
		if msg, ok := rb.tryReadQueuedMessage(); ok {
			return msg, nil
		}
		return nil, fmt.Errorf("RecvBuffer is already stopped")
	}
	select {
	case msg := <-rb.queue:
		atomic.AddInt32(&rb.queueSize, -1)
		return msg, nil
	case <-rb.done:
		if msg, ok := rb.tryReadQueuedMessage(); ok {
			return msg, nil
		}
		return nil, fmt.Errorf("RecvBuffer is already stopped")
	}
}

func (rb *RecvBuffer) tryReadQueuedMessage() ([]byte, bool) {
	select {
	case msg := <-rb.queue:
		atomic.AddInt32(&rb.queueSize, -1)
		return msg, true
	default:
		return nil, false
	}
}

// GetSize returns the size of the SendBuffer. It is goroutine safe
func (rb *RecvBuffer) GetSize() int {
	return int(atomic.LoadInt32(&rb.queueSize))
}

func (rb *RecvBuffer) recvRoutine() {
	defer rb.wg.Done()
	defer rb.Stop()
	defer rb.recover()
	defer rb.resetReassemblyState()

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
		rb.applyReadDeadline()
		numBytesRead, err := rb.rawStream.Read(bytes)
		if numBytesRead > 0 {
			rb.extractChunks(bytes, numBytesRead)
			rb.recvMonitor.Update(numBytesRead)
		}
		if err != nil {
			if rb.reassemblyTimedOut(time.Now()) {
				rb.fail(fmt.Errorf("message reassembly exceeded deadline %v", rb.reassemblyDeadline()))
				return
			}
			log.Warnf("Raw stream read error: %v", err)
			return
		}
	}
}

// extractChunks extract the chunks from the bytes read from the stream. Note that the bytes
// read from the stream might contain multiple chunks or partial chunk from the sender. Hence
// we need to handle rollover and preceding bytes properly
func (rb *RecvBuffer) extractChunks(bytes []byte, numBytesRead int) {
	if numBytesRead > 0 {
		now := time.Now()
		if rb.reassemblyTimedOut(now) {
			rb.fail(fmt.Errorf("message reassembly exceeded deadline %v", rb.reassemblyDeadline()))
			return
		}
		if rb.reassemblyStartedAt.IsZero() {
			rb.reassemblyStartedAt = now
		}
		rb.reassemblyLastProgressAt = now
	}
	pending := bytes[:numBytesRead]
	if len(rb.rolloverBytes) > 0 {
		pending = append(rb.rolloverBytes, pending...)
		rb.rolloverBytes = nil
	}
	rb.precedingBytes = nil

	for len(pending) > 0 {
		if rb.reassemblyStartedAt.IsZero() {
			now := time.Now()
			rb.reassemblyStartedAt = now
			rb.reassemblyLastProgressAt = now
		}
		chunkSize, err := validateChunkHeader(pending)
		if err != nil {
			rb.fail(err)
			return
		}
		if chunkSize == 0 {
			rb.storeRollover(pending, 0)
			return
		}
		if len(pending) < chunkSize {
			rb.storeRollover(pending, chunkSize)
			return
		}

		chunk, err := NewChunkFromRawBytes(pending[:chunkSize])
		if err == nil {
			completeMessage, success, err := rb.aggregateChunk(chunk)
			if err != nil {
				rb.fail(err)
				return
			}
			if success {
				if completeMessage != nil {
					atomic.AddInt32(&rb.queueSize, 1)
					select {
					case rb.queue <- completeMessage:
					case <-rb.done:
						atomic.AddInt32(&rb.queueSize, -1)
						return
					}
				}
			}
		} else {
			rb.fail(fmt.Errorf("RecvBuffer failed to create new chunk from raw bytes: %v", err))
			return
		}

		pending = pending[chunkSize:]
	}
}

// aggregateChunk aggregates incoming chunks. It returns the message bytes if the message is
// complete (i.e. ends with EOF). It is not goroutine safe
func (rb *RecvBuffer) aggregateChunk(chunk *Chunk) (completeMessage []byte, success bool, err error) {
	// Note: We do NOT need to worry about the order of the chunks.
	//       TCP guarantees that if bytes arrive, they will be in the
	//       order they were sent, as long as the TCP connection stays open.
	//       But we do need to check if there's any missing chunk
	if rb.seqID != chunk.SeqID() {
		return nil, false, fmt.Errorf("chunk seqID mismatch, expected %v, actual %v", rb.seqID, chunk.SeqID())
	}
	if rb.reassemblyTimedOut(time.Now()) {
		deadline := rb.reassemblyDeadline()
		rb.resetReassemblyState()
		return nil, false, fmt.Errorf("message reassembly exceeded deadline %v", deadline)
	}

	chunkPayload := chunk.Payload()
	log.Debugf("Aggregate chunk: payloadSize = %v, seqID = %v, isEOF = %v", len(chunkPayload), chunk.SeqID(), chunk.IsEOF())
	maxMessageSize := rb.config.MaxMessageSize
	if maxMessageSize <= 0 {
		maxMessageSize = cmn.MaxNormalMessageSize
	}
	maxChunkCount := maxMessageSize/maxChunkPayloadSize + 2
	if rb.chunkCount >= maxChunkCount {
		rb.resetReassemblyState()
		return nil, false, fmt.Errorf("message exceeds chunk count limit %v", maxChunkCount)
	}
	if len(chunkPayload) > maxMessageSize || len(rb.workspace) > maxMessageSize-len(chunkPayload) {
		currentSize := len(rb.workspace)
		rb.resetReassemblyState()
		return nil, false, fmt.Errorf("message exceeds receive limit, current %v, chunk %v, limit %v",
			currentSize, len(chunkPayload), maxMessageSize)
	}

	rb.workspace = append(rb.workspace, chunkPayload...)
	rb.chunkCount++
	if chunk.IsEOF() {
		msgSize := len(rb.workspace)
		completeMessage := make([]byte, msgSize)
		copy(completeMessage, rb.workspace)

		// clear the slice without re-allocating.
		// http://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
		//   suggests this could be a memory leak, but we might as well keep the memory for the buffer until it closes,
		//	at which point the recving slice stops being used and should be garbage collected
		rb.resetReassemblyState()

		return completeMessage, true, nil
	}

	rb.seqID++
	return nil, true, nil
}

func validateChunkHeader(bytes []byte) (int, error) {
	const payloadSizeBytes = 4
	if len(bytes) < payloadSizeOffset+payloadSizeBytes {
		return 0, nil
	}
	seqID := int32FromBytes(bytes[seqIDOffset : seqIDOffset+payloadSizeBytes])
	if seqID < 0 {
		return 0, fmt.Errorf("invalid chunk seqID %v", seqID)
	}
	payloadSize := int32FromBytes(bytes[payloadSizeOffset : payloadSizeOffset+payloadSizeBytes])
	if payloadSize < 0 || payloadSize > maxChunkPayloadSize {
		return 0, fmt.Errorf("invalid chunk payloadSize %v", payloadSize)
	}
	if len(bytes) > isEOFOffset && bytes[isEOFOffset] != markerNotEOF && bytes[isEOFOffset] != markerEOF {
		return 0, fmt.Errorf("invalid chunk EOF marker %v", bytes[isEOFOffset])
	}
	return headerSize + int(payloadSize), nil
}

func (rb *RecvBuffer) storeRollover(bytes []byte, chunkSize int) {
	if chunkSize > 0 {
		rb.rolloverBytes = make([]byte, len(bytes), chunkSize)
	} else {
		rb.rolloverBytes = make([]byte, len(bytes), headerSize)
	}
	copy(rb.rolloverBytes, bytes)
}

func (rb *RecvBuffer) resetWorkspace() {
	if cap(rb.workspace) > rb.config.workspaceCapacity {
		rb.workspace = make([]byte, 0, rb.config.workspaceCapacity)
	} else {
		rb.workspace = rb.workspace[:0]
	}
}

func (rb *RecvBuffer) resetReassemblyState() {
	rb.resetWorkspace()
	rb.rolloverBytes = rb.rolloverBytes[:0]
	rb.precedingBytes = rb.precedingBytes[:0]
	rb.seqID = 0
	rb.chunkCount = 0
	rb.reassemblyStartedAt = time.Time{}
	rb.reassemblyLastProgressAt = time.Time{}
}

type readDeadlineSetter interface {
	SetReadDeadline(time.Time) error
}

func (rb *RecvBuffer) applyReadDeadline() {
	stream, ok := rb.rawStream.(readDeadlineSetter)
	if !ok {
		return
	}
	deadline := rb.reassemblyDeadline()
	if err := stream.SetReadDeadline(deadline); err != nil {
		log.Warnf("Failed to set raw stream read deadline: %v", err)
	}
}

func (rb *RecvBuffer) reassemblyTimedOut(now time.Time) bool {
	deadline := rb.reassemblyDeadline()
	return !deadline.IsZero() && !now.Before(deadline)
}

func (rb *RecvBuffer) reassemblyDeadline() time.Time {
	return rootcmn.P2PReassemblyDeadline(rb.reassemblyStartedAt, rb.reassemblyLastProgressAt,
		rb.config.MaxMessageSize, rb.config.timeOut)
}

func (rb *RecvBuffer) fail(err error) {
	log.Warnf("RecvBuffer protocol error: %v", err)
	rb.resetReassemblyState()
	if rb.onError != nil {
		rb.onError(err)
	}
	rb.Stop()
}

func (rb *RecvBuffer) recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := fmt.Errorf("%s", stack)
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
