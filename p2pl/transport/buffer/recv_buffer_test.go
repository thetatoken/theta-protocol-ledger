package buffer

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	rootcmn "github.com/thetatoken/theta/common"
	cmn "github.com/thetatoken/theta/p2pl/common"
)

type dataAndEOFStream struct {
	data []byte
	read bool
}

func (stream *dataAndEOFStream) Read(p []byte) (int, error) {
	if stream.read {
		return 0, io.EOF
	}
	stream.read = true
	return copy(p, stream.data), io.EOF
}

func (*dataAndEOFStream) Write(p []byte) (int, error) { return len(p), nil }
func (*dataAndEOFStream) Close() error                { return nil }

func newTestRecvBuffer(maxMessageSize int, errCh chan interface{}) RecvBuffer {
	config := GetDefaultRecvBufferConfig()
	config.MaxMessageSize = maxMessageSize
	return NewRecvBuffer(config, nil, func(err interface{}) {
		errCh <- err
	})
}

func TestRecvBufferRejectsBogusLargePayloadSizeBeforeAllocation(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	raw := rawChunk(0, 1<<30, markerEOF, nil)

	rb.extractChunks(raw[:payloadSizeOffset+4], payloadSizeOffset+4)

	require.NotNil(t, <-errCh)
	require.Len(t, rb.rolloverBytes, 0)
	require.LessOrEqual(t, cap(rb.rolloverBytes), cmn.MaxChunkSize)
}

func TestRecvBufferRejectsNegativePayloadSizeBeforeAllocation(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	raw := rawChunk(0, -1, markerEOF, nil)

	rb.extractChunks(raw[:payloadSizeOffset+4], payloadSizeOffset+4)

	require.NotNil(t, <-errCh)
	require.Len(t, rb.rolloverBytes, 0)
	require.LessOrEqual(t, cap(rb.rolloverBytes), cmn.MaxChunkSize)
}

func TestRecvBufferStoresValidPartialChunkWithBoundedCapacity(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	raw := rawChunk(0, maxChunkPayloadSize, markerNotEOF, nil)

	rb.extractChunks(raw[:payloadSizeOffset+4], payloadSizeOffset+4)

	require.Empty(t, errCh)
	require.Len(t, rb.rolloverBytes, payloadSizeOffset+4)
	require.Equal(t, cmn.MaxChunkSize, cap(rb.rolloverBytes))
}

func TestRecvBufferRejectsInvalidEOFMarker(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	raw := rawChunk(0, 0, byte(0xff), nil)

	rb.extractChunks(raw, len(raw))

	require.NotNil(t, <-errCh)
}

func TestRecvBufferAggregatesMessageAtLimit(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(10, errCh)
	chunk1 := NewChunk([]byte("hello"), 0, 5, markerNotEOF, 0)
	chunk2 := NewChunk([]byte("world"), 0, 5, markerEOF, 1)
	raw := append(append([]byte{}, chunk1.Bytes()...), chunk2.Bytes()...)

	rb.extractChunks(raw, len(raw))

	require.Empty(t, errCh)
	require.Equal(t, []byte("helloworld"), <-rb.queue)
}

func TestRecvBufferStartsDeadlineForSecondMessageInSameRead(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(64, errCh)
	first := NewChunk([]byte("complete"), 0, 8, markerEOF, 0)
	second := NewChunk([]byte("partial"), 0, 7, markerNotEOF, 0)
	raw := append(append([]byte{}, first.Bytes()...), second.Bytes()...)

	rb.extractChunks(raw, len(raw))

	require.Empty(t, errCh)
	require.Equal(t, []byte("complete"), <-rb.queue)
	require.Equal(t, []byte("partial"), rb.workspace)
	require.False(t, rb.reassemblyStartedAt.IsZero())
	require.False(t, rb.reassemblyDeadline().IsZero())
}

func TestRecvBufferRejectsMessageAboveLimit(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(10, errCh)
	chunk1 := NewChunk(bytes.Repeat([]byte{0x01}, 8), 0, 8, markerNotEOF, 0)
	chunk2 := NewChunk(bytes.Repeat([]byte{0x02}, 8), 0, 8, markerEOF, 1)
	raw := append(append([]byte{}, chunk1.Bytes()...), chunk2.Bytes()...)

	rb.extractChunks(raw, len(raw))

	require.NotNil(t, <-errCh)
	require.Len(t, rb.workspace, 0)
	require.LessOrEqual(t, cap(rb.workspace), rb.config.workspaceCapacity)
}

func TestRecvBufferRejectsSequenceMismatch(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	chunk := NewChunk([]byte("wrong-seq"), 0, 9, markerEOF, 1)

	rb.extractChunks(chunk.Bytes(), len(chunk.Bytes()))

	require.NotNil(t, <-errCh)
}

func TestRecvBufferRejectsExcessChunkCount(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(10, errCh)
	maxChunkCount := rb.config.MaxMessageSize/maxChunkPayloadSize + 2

	for seq := 0; seq < maxChunkCount; seq++ {
		chunk := NewChunk(nil, 0, 0, markerNotEOF, int32(seq))
		rb.extractChunks(chunk.Bytes(), len(chunk.Bytes()))
		require.Empty(t, errCh)
	}

	chunk := NewChunk(nil, 0, 0, markerNotEOF, int32(maxChunkCount))
	rb.extractChunks(chunk.Bytes(), len(chunk.Bytes()))
	require.NotNil(t, <-errCh)
	require.Zero(t, rb.chunkCount)
	require.Empty(t, rb.workspace)
}

func TestRecvBufferTimesOutIncompleteMessage(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	config := GetDefaultRecvBufferConfig()
	config.MaxMessageSize = 64
	config.timeOut = 50 * time.Millisecond
	errCh := make(chan interface{}, 1)
	rb := NewRecvBuffer(config, left, func(err interface{}) {
		errCh <- err
	})
	rb.Start(context.Background())
	defer rb.Stop()

	chunk := NewChunk([]byte("partial"), 0, 7, markerNotEOF, 0)
	go func() {
		_, _ = right.Write(chunk.Bytes()[:payloadSizeOffset+4])
	}()

	select {
	case err := <-errCh:
		require.NotNil(t, err)
	case <-time.After(time.Second):
		t.Fatal("expected incomplete message to exceed the reassembly deadline")
	}
	rb.Wait()
	require.Empty(t, rb.workspace)
}

func TestRecvBufferDoesNotTimeOutIdleStream(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	config := GetDefaultRecvBufferConfig()
	config.MaxMessageSize = 64
	config.timeOut = 30 * time.Millisecond
	rb := NewRecvBuffer(config, left, nil)
	rb.Start(context.Background())
	defer rb.Stop()

	time.Sleep(2 * config.timeOut)
	chunk := NewChunk([]byte("after-idle"), 0, 10, markerEOF, 0)
	go func() {
		_, _ = right.Write(chunk.Bytes())
	}()

	select {
	case msg := <-rb.queue:
		require.Equal(t, []byte("after-idle"), msg)
	case <-time.After(time.Second):
		t.Fatal("idle stream was closed despite having no partial message")
	}
}

func TestRecvBufferReadUnblocksWhenRawStreamCloses(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()

	rb := NewRecvBuffer(GetDefaultRecvBufferConfig(), left, nil)
	rb.Start(context.Background())
	result := make(chan error, 1)
	go func() {
		_, err := rb.Read()
		result <- err
	}()
	require.NoError(t, right.Close())

	select {
	case err := <-result:
		require.Error(t, err)
	case <-time.After(time.Second):
		t.Fatal("RecvBuffer.Read remained blocked after the raw stream closed")
	}
	rb.Wait()
}

func TestRecvBufferDeliversQueuedMessageBeforeEOF(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()

	rb := NewRecvBuffer(GetDefaultRecvBufferConfig(), left, nil)
	rb.Start(context.Background())
	chunk := NewChunk([]byte("final-message"), 0, 13, markerEOF, 0)

	go func() {
		_, _ = right.Write(chunk.Bytes())
		_ = right.Close()
	}()

	rb.Wait()
	require.Equal(t, 1, rb.GetSize())
	msg, err := rb.Read()
	require.NoError(t, err)
	require.Equal(t, []byte("final-message"), msg)
	require.Zero(t, rb.GetSize())

	_, err = rb.Read()
	require.Error(t, err)
}

func TestRecvBufferProcessesFinalBytesReturnedWithEOF(t *testing.T) {
	chunk := NewChunk([]byte("final-message"), 0, 13, markerEOF, 0)
	rawStream := &dataAndEOFStream{data: chunk.Bytes()}
	rb := NewRecvBuffer(GetDefaultRecvBufferConfig(), rawStream, nil)

	rb.Start(context.Background())
	rb.Wait()

	msg, err := rb.Read()
	require.NoError(t, err)
	require.Equal(t, []byte("final-message"), msg)
}

func TestRecvBufferAllowsProgressPastInactivityTimeout(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(cmn.MaxNormalMessageSize, errCh)
	rb.config.timeOut = 10 * time.Millisecond

	first := NewChunk([]byte("part-one"), 0, 8, markerNotEOF, 0)
	rb.extractChunks(first.Bytes(), len(first.Bytes()))
	require.Empty(t, errCh)
	rb.reassemblyStartedAt = time.Now().Add(-2 * rb.config.timeOut)
	rb.reassemblyLastProgressAt = time.Now()

	second := NewChunk([]byte("part-two"), 0, 8, markerEOF, 1)
	rb.extractChunks(second.Bytes(), len(second.Bytes()))
	require.Empty(t, errCh)
	require.Equal(t, []byte("part-onepart-two"), <-rb.queue)
}

func TestRecvBufferRejectsTricklePastAbsoluteDeadline(t *testing.T) {
	errCh := make(chan interface{}, 1)
	rb := newTestRecvBuffer(64, errCh)
	rb.config.timeOut = 10 * time.Millisecond

	first := NewChunk([]byte("partial"), 0, 7, markerNotEOF, 0)
	rb.extractChunks(first.Bytes(), len(first.Bytes()))
	require.Empty(t, errCh)
	rb.reassemblyStartedAt = time.Now().Add(-rootcmn.MaxP2PReassemblyDuration(
		rb.config.MaxMessageSize, rb.config.timeOut))
	rb.reassemblyLastProgressAt = time.Now()

	second := NewChunk([]byte("still-sending"), 0, 13, markerEOF, 1)
	rb.extractChunks(second.Bytes(), len(second.Bytes()))
	require.NotNil(t, <-errCh)
	require.Empty(t, rb.workspace)
}
