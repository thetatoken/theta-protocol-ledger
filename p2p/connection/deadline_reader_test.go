package connection

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFrameDeadlineReaderDoesNotExpireIdleConnection(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	reader := newFrameDeadlineReader(left, left, 20*time.Millisecond)
	require.NoError(t, reader.beginFrame(time.Time{}))
	defer reader.endFrame()

	result := make(chan error, 1)
	go func() {
		buffer := make([]byte, 1)
		_, err := io.ReadFull(reader, buffer)
		result <- err
	}()

	time.Sleep(2 * reader.timeout)
	_, err := right.Write([]byte{0x42})
	require.NoError(t, err)
	require.NoError(t, <-result)
}

func TestFrameDeadlineReaderExpiresPartialFrame(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	reader := newFrameDeadlineReader(left, left, 20*time.Millisecond)
	require.NoError(t, reader.beginFrame(time.Time{}))
	defer reader.endFrame()

	result := make(chan error, 1)
	go func() {
		buffer := make([]byte, 2)
		_, err := io.ReadFull(reader, buffer)
		result <- err
	}()
	_, err := right.Write([]byte{0x42})
	require.NoError(t, err)

	select {
	case err := <-result:
		require.Error(t, err)
		netErr, ok := err.(net.Error)
		require.True(t, ok)
		require.True(t, netErr.Timeout())
	case <-time.After(time.Second):
		t.Fatal("partial frame did not expire")
	}
}

func TestFrameDeadlineReaderHonorsEarlierReassemblyDeadline(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	reader := newFrameDeadlineReader(left, left, time.Second)
	require.NoError(t, reader.beginFrame(time.Now().Add(20*time.Millisecond)))
	defer reader.endFrame()

	buffer := make([]byte, 1)
	_, err := reader.Read(buffer)
	require.Error(t, err)
	netErr, ok := err.(net.Error)
	require.True(t, ok)
	require.True(t, netErr.Timeout())
}
