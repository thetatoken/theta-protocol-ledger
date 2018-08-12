package connection

import "fmt"

const (
	maxPayloadSize        = 1024
	maxAdditionalDataSize = 10
	maxPacketTotalSize    = maxPayloadSize + maxAdditionalDataSize
	packetTypePing        = byte(0x01)
	packetTypePong        = byte(0x02)
	packetTypeMsg         = byte(0x03)
)

type Packet struct {
	ChannelID byte
	Bytes     []byte
	IsEOF     byte // 1 means message ends here.
}

func (p Packet) String() string {
	return fmt.Sprintf("Packet{%X:%X T:%X}", p.ChannelID, p.Bytes, p.IsEOF)
}
