package common

import "io"

const (
	MaxBlockMessageSize  = 12 * 1024 * 1024 // 12 MBytes
	MaxNormalMessageSize = 1024 * 1024      // 1 MBytes
	MaxChunkSize         = 16 * 1024        // 16 kbytes, lowest common denominator among browsers
	//MaxChunkSize = 64 * 1024
	MaxSendRate = int64(128 * 1024 * 1024) // 128 Mbps
	MaxRecvRate = int64(128 * 1024 * 1024) // 128 Mbps
)

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

type ErrorHandler func(interface{})
