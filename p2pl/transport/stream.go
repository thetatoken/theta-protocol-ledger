package transport

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	p2pcmn "github.com/thetatoken/theta/p2pl/common"
	buf "github.com/thetatoken/theta/p2pl/transport/buffer"
)

// BufferedStream is a bidirectional I/O pipe that supports
// sending/receiving arbitrarily long messages
type BufferedStream struct {
	rawStream p2pcmn.ReadWriteCloser

	sendBuf buf.SendBuffer
	recvBuf buf.RecvBuffer

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

func NewBufferedStream(rawStream p2pcmn.ReadWriteCloser, onError p2pcmn.ErrorHandler) *BufferedStream {
	s := &BufferedStream{
		rawStream: rawStream,
		sendBuf:   buf.NewSendBuffer(buf.GetDefaultSendBufferConfig(), rawStream, onError),
		recvBuf:   buf.NewRecvBuffer(buf.GetDefaultRecvBufferConfig(), rawStream, onError),
		stopped:   true,
	}

	return s
}

func (s *BufferedStream) Start(ctx context.Context) bool {
	ctx, cancel := context.WithCancel(ctx)
	s.ctx = ctx
	s.cancel = cancel

	s.sendBuf.Start(ctx)
	s.recvBuf.Start(ctx)

	s.stopped = false

	return true
}

// Wait suspends the caller goroutine
func (s *BufferedStream) Wait() {
	s.wg.Wait()
}

// Stop is called when the BufferedStream stops
func (s *BufferedStream) Stop() {
	if s.stopped {
		return
	}

	s.stopped = true

	s.recvBuf.Stop()
	s.sendBuf.Stop()
	s.Close()

	s.cancel()
}

// TODO: Read implements the io.Reader
func (s *BufferedStream) Read(bufferPool chan []byte) ([]byte, int, error) {
	var err error
	msgRead, err := s.recvBuf.Read()
	if err != nil {
		return nil, 0, err
	}
	toCopy := len(msgRead)

	msg := <-bufferPool
	n := copy(msg, msgRead)
	if n < toCopy {
		err = io.ErrShortBuffer
	}

	return msg, n, err
}

// Write implements the io.Writer, and supports writting
// arbitrarily long messages
func (s *BufferedStream) Write(msg []byte) (int, error) {
	success := s.sendBuf.Write(msg)
	if !success {
		return 0, fmt.Errorf("Failed to write message to stream")
	}
	return len(msg), nil
}

// Close closes the stream for writing. Reading will still work (that
// is, the remote side can still write).
func (s *BufferedStream) Close() error {
	// TODO: figure out close vs reset
	return s.rawStream.Close()
}

// Reset closes both ends of the stream. Use this to tell the remote
// side to hang up and go away.
func (s *BufferedStream) Reset() error {
	// TODO: figure out close vs reset
	return s.rawStream.Close()
}

// SetDeadline is a stub
func (s *BufferedStream) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline is a stub
func (s *BufferedStream) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline is a stub
func (s *BufferedStream) SetWriteDeadline(t time.Time) error {
	return nil
}
