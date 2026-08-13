package transport

import (
	"bytes"
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBufferedStreamReadsMessageAtLimit(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	receiver := NewBufferedStreamWithMaxMessageSize(left, nil, 64)
	sender := NewBufferedStreamWithMaxMessageSize(right, nil, 64)
	receiver.Start(ctx)
	sender.Start(ctx)
	defer receiver.Stop()
	defer sender.Stop()

	msg := bytes.Repeat([]byte{0x42}, 64)
	go func() {
		_, _ = sender.Write(msg)
	}()

	bufferPool := make(chan []byte, 1)
	bufferPool <- make([]byte, 64)
	received, n, err := receiver.Read(bufferPool)
	require.NoError(t, err)
	require.Equal(t, 64, n)
	require.Equal(t, msg, received[:n])
}

func TestBufferedStreamRejectsMessageAboveLimit(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan interface{}, 1)
	receiver := NewBufferedStreamWithMaxMessageSize(left, func(err interface{}) {
		errCh <- err
	}, 64)
	sender := NewBufferedStreamWithMaxMessageSize(right, nil, 64)
	receiver.Start(ctx)
	sender.Start(ctx)
	defer receiver.Stop()
	defer sender.Stop()

	go func() {
		_, _ = sender.Write(bytes.Repeat([]byte{0x42}, 65))
	}()

	select {
	case err := <-errCh:
		require.NotNil(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("expected receiver to reject the oversized message")
	}
}

func TestBufferedStreamRejectsBogusLargeChunkHeader(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan interface{}, 1)
	receiver := NewBufferedStreamWithMaxMessageSize(left, func(err interface{}) {
		errCh <- err
	}, 64)
	receiver.Start(ctx)
	defer receiver.Stop()

	go func() {
		header := make([]byte, 8)
		binary.BigEndian.PutUint32(header[0:4], 0)
		binary.BigEndian.PutUint32(header[4:8], 1<<30)
		_, _ = right.Write(header)
	}()

	select {
	case err := <-errCh:
		require.NotNil(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("expected receiver to reject the bogus large chunk header")
	}
}

func TestBufferedStreamStopIsConcurrentSafe(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()

	stream := NewBufferedStreamWithMaxMessageSize(left, nil, 64)
	stream.Start(context.Background())

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			stream.Stop()
		}()
	}
	wg.Wait()

	_, err := stream.Write([]byte("after-stop"))
	require.Error(t, err)
}

func TestBufferedStreamWaitReturnsAfterStop(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()

	stream := NewBufferedStreamWithMaxMessageSize(left, nil, 64)
	stream.Start(context.Background())
	stream.Stop()

	done := make(chan struct{})
	go func() {
		stream.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("BufferedStream.Wait did not return after Stop")
	}
}
