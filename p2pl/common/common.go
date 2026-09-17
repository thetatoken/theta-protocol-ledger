package common

import (
	"io"

	rootcmn "github.com/thetatoken/theta/common"
)

const (
	MaxBlockMessageSize  = rootcmn.MaxBlockMessageSize
	MaxNormalMessageSize = rootcmn.MaxNormalMessageSize
	MaxChunkSize         = 16 * 1024 // 16 kbytes, lowest common denominator among browsers
	//MaxChunkSize = 64 * 1024
	MaxSendRate = int64(128 * 1024 * 1024) // 128 Mbps
	MaxRecvRate = int64(128 * 1024 * 1024) // 128 Mbps
)

func MaxMessageSizeForChannel(channelID rootcmn.ChannelIDEnum) int {
	return rootcmn.MaxP2PMessageSize(channelID)
}

type ReadWriteCloser interface {
	io.Reader
	io.Writer
	io.Closer
}

type ErrorHandler func(interface{})
