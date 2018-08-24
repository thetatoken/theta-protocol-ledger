package connection

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

const (
	maxPayloadSize        = 1024
	maxAdditionalDataSize = 10
	maxPacketTotalSize    = maxPayloadSize + maxAdditionalDataSize
	packetTypePing        = byte(0x01)
	packetTypePong        = byte(0x02)
	packetTypeMsg         = byte(0x03)
)

type Packet struct {
	ChannelID common.ChannelIDEnum
	Bytes     []byte
	IsEOF     byte // 1 means message ends here.
}

func (p *Packet) isEmpty() bool {
	return (p.Bytes == nil || len(p.Bytes) == 0)
}

func (p Packet) String() string {
	return fmt.Sprintf("Packet{%X:%X T:%X}", p.ChannelID, p.Bytes, p.IsEOF)
}
