package buffer

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type blockingWriteStream struct {
	writeStarted chan struct{}
	releaseWrite chan struct{}
	startOnce    sync.Once
}

func (stream *blockingWriteStream) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (stream *blockingWriteStream) Write(bytes []byte) (int, error) {
	stream.startOnce.Do(func() { close(stream.writeStarted) })
	<-stream.releaseWrite
	return len(bytes), nil
}

func (stream *blockingWriteStream) Close() error {
	return nil
}

func TestSendBufferBlockedWriteReturnsFalseOnStop(t *testing.T) {
	rawStream := &blockingWriteStream{
		writeStarted: make(chan struct{}),
		releaseWrite: make(chan struct{}),
	}
	config := GetDefaultSendBufferConfig()
	config.timeOut = time.Second
	sb := NewSendBuffer(config, rawStream, nil)
	sb.Start(context.Background())
	defer func() {
		close(rawStream.releaseWrite)
		sb.Wait()
	}()

	require.True(t, sb.Write([]byte("being-written")))
	select {
	case <-rawStream.writeStarted:
	case <-time.After(time.Second):
		t.Fatal("send routine did not start writing")
	}
	require.True(t, sb.Write([]byte("queued")))

	result := make(chan bool, 1)
	go func() {
		result <- sb.Write([]byte("blocked"))
	}()
	time.Sleep(10 * time.Millisecond)
	sb.Stop()

	select {
	case success := <-result:
		require.False(t, success)
	case <-time.After(time.Second):
		t.Fatal("blocked write did not return after Stop")
	}
}

func TestSendBufferRejectsWritesAfterConcurrentStop(t *testing.T) {
	config := GetDefaultSendBufferConfig()
	sb := NewSendBuffer(config, &blockingWriteStream{
		writeStarted: make(chan struct{}),
		releaseWrite: make(chan struct{}),
	}, nil)
	sb.Start(context.Background())
	sb.Stop()

	var wg sync.WaitGroup
	results := make(chan bool, 64)
	for i := 0; i < cap(results); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- sb.Write([]byte("after-stop"))
		}()
	}
	wg.Wait()
	close(results)
	for success := range results {
		require.False(t, success)
	}
	sb.Wait()
}

func TestSendBufferWriteDoesNotSucceedIfStopWinsAfterEnqueue(t *testing.T) {
	config := GetDefaultSendBufferConfig()
	sb := NewSendBuffer(config, nil, nil)
	atomic.StoreUint32(&sb.stopped, 0)

	// Hold the lifecycle lock so Write can enqueue but cannot declare success.
	sb.stateMu.Lock()
	result := make(chan bool, 1)
	go func() {
		result <- sb.Write([]byte("racing-with-stop"))
	}()
	deadline := time.Now().Add(time.Second)
	for len(sb.queue) == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	require.Len(t, sb.queue, 1)
	atomic.StoreUint32(&sb.stopped, 1)
	close(sb.done)
	sb.stateMu.Unlock()

	select {
	case success := <-result:
		require.False(t, success)
	case <-time.After(time.Second):
		t.Fatal("write did not finish after lifecycle lock was released")
	}
}
