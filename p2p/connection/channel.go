package connection

import (
	"io"

	"github.com/thetatoken/ukulele/serialization/rlp"
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
func createChannel(channelID byte, channelConf ChannelConfig, sbConf SendBufferConfig, rbConf RecvBufferConfig) Channel {
	sendBuf := createSendBuffer(sbConf)
	recvBuf := createRecvBuffer(rbConf)
	return Channel{
		id:      channelID,
		sendBuf: sendBuf,
		recvBuf: recvBuf,
		config:  channelConf,
	}
}

// enqueueMessage queues the the given message into the channel
func (ch *Channel) enqueueMessage(bytes []byte) bool {
	success := ch.sendBuf.insert(bytes)
	return success
}

// attemptToEnqueueMessage attempts to queue the given message into the channel (non-blocking)
func (ch *Channel) attemptToEnqueueMessage(bytes []byte) bool {
	success := ch.sendBuf.attemptInsert(bytes)
	return success
}

// receivePacket receives packet and return the converted bytes
func (ch *Channel) receivePacket(packet Packet) ([]byte, bool) {
	bytes, success := ch.recvBuf.receivePacket(packet)
	return bytes, success
}

// sendPacketTo serializes and sends the next packet to the given writer
func (ch *Channel) sendPacketTo(writer io.Writer) (nonemptyPacket bool, numBytes int, err error) {
	packet := ch.sendBuf.emitPacket(ch.id)
	if packet.isEmpty() {
		return false, int(0), nil
	}
	packetBytes, err := rlp.EncodeToBytes(packet)
	if err != nil {
		return true, int(0), nil
	}

	// FIXME: may not be efficient to first EncodeToBytes and then Encode to the writer, but needs
	//        to get the size of the packetBytes here..
	numBytes = len(packetBytes)
	err = rlp.Encode(writer, packetBytes)
	return false, numBytes, err
}

// canEnqueueMessage returns whether more messages can be queued into the channel
func (ch *Channel) canEnqueueMessage() bool {
	return ch.sendBuf.canInsert()
}

// hasPacketToSend returns whether there are pending data in the sendBuffer
func (ch *Channel) hasPacketToSend() bool {
	hasPacket := !ch.sendBuf.isEmpty()
	return hasPacket
}
