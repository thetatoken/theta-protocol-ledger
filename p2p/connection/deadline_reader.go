package connection

import (
	"io"
	"net"
	"time"
)

// frameDeadlineReader leaves idle connections open, then applies a deadline
// once any bytes for a packet arrive. An earlier message-reassembly deadline
// remains in force while waiting for the next packet.
type frameDeadlineReader struct {
	reader          io.Reader
	conn            net.Conn
	timeout         time.Duration
	armed           bool
	frameStarted    bool
	initialDeadline time.Time
}

func newFrameDeadlineReader(reader io.Reader, conn net.Conn, timeout time.Duration) *frameDeadlineReader {
	return &frameDeadlineReader{reader: reader, conn: conn, timeout: timeout}
}

func (reader *frameDeadlineReader) beginFrame(initialDeadline time.Time) error {
	reader.armed = true
	reader.frameStarted = false
	reader.initialDeadline = initialDeadline
	return reader.conn.SetReadDeadline(initialDeadline)
}

func (reader *frameDeadlineReader) endFrame() {
	reader.armed = false
	reader.frameStarted = false
	reader.initialDeadline = time.Time{}
	_ = reader.conn.SetReadDeadline(time.Time{})
}

func (reader *frameDeadlineReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	if deadlineErr := reader.markFrameStarted(n); err == nil && deadlineErr != nil {
		err = deadlineErr
	}
	return n, err
}

func (reader *frameDeadlineReader) ReadByte() (byte, error) {
	byteReader, ok := reader.reader.(io.ByteReader)
	if !ok {
		var buffer [1]byte
		_, err := io.ReadFull(reader, buffer[:])
		return buffer[0], err
	}
	b, err := byteReader.ReadByte()
	if deadlineErr := reader.markFrameStarted(1); err == nil && deadlineErr != nil {
		err = deadlineErr
	}
	return b, err
}

func (reader *frameDeadlineReader) markFrameStarted(bytesRead int) error {
	if bytesRead <= 0 || !reader.armed || reader.frameStarted {
		return nil
	}
	reader.frameStarted = true
	deadline := time.Now().Add(reader.timeout)
	if !reader.initialDeadline.IsZero() && reader.initialDeadline.Before(deadline) {
		deadline = reader.initialDeadline
	}
	return reader.conn.SetReadDeadline(deadline)
}
