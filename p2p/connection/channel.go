package connection

import (
	"io"
)

//
// Channel models a bi-directional channel for messsage between two peers
//
type Channel struct {
	id byte

	sendBuf SendBuffer
	recvBuf RecvBuffer

	config ChannelConfig
}

//
// ChannelConfig specifies the configuration of a Channel
//
type ChannelConfig struct {
	priority uint
}

// createChannel creates a channel for the given configs
func createChannel(channelID byte, channelConf ChannelConfig, sbConf SendBufferConfig, rbConf RecvBufferConfig) *Channel {
	sendBuf := createSendBuffer(sbConf)
	recvBuf := createRecvBuffer(rbConf)
	return &Channel{
		id:      channelID,
		sendBuf: sendBuf,
		recvBuf: recvBuf,
		config:  channelConf,
	}
}

// send sends the given bytes through the channel
func (ch *Channel) send(bytes []byte) bool {
	success := ch.sendBuf.insert(bytes)
	return success
}

// attemptSend attempts to send the given bytes through the channel (non-blocking)
func (ch *Channel) attemptSend(bytes []byte) bool {
	success := ch.sendBuf.attemptInsert(bytes)
	return success
}

// receivePacket receives packet and return the converted bytes
func (ch *Channel) receivePacket(packet Packet) ([]byte, bool) {
	bytes, success := ch.recvBuf.receivePacket(packet)
	return bytes, success
}

// writePacketTo serializes and writes the packet to the given writer
func (ch *Channel) writePacketTo(writer io.Writer) bool {
	//packet := ch.sendBuf.emitPacket(ch.id)

	// TODO: serialize packet and write to the writer

	return true
}
