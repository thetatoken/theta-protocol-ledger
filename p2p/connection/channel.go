package connection

import (
	"github.com/thetatoken/theta/common"
)

//
// Channel models a bi-directional channel for messsaging between two peers
//
type Channel struct {
	id common.ChannelIDEnum

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

// createDefaultChannel creates a channel with default configs
func createDefaultChannel(channelID common.ChannelIDEnum) Channel {
	chCfg := getDefaultChannelConfig()
	sbCfg := getDefaultSendBufferConfig()
	rbCfg := getDefaultRecvBufferConfig()

	channel := createChannel(channelID, chCfg, sbCfg, rbCfg)
	return channel
}

// createChannel creates a channel for the given configs
func createChannel(channelID common.ChannelIDEnum, channelConf ChannelConfig, sbConf SendBufferConfig, rbConf RecvBufferConfig) Channel {
	sendBuf := createSendBuffer(sbConf)
	recvBuf := createRecvBuffer(rbConf)
	return Channel{
		id:      channelID,
		sendBuf: sendBuf,
		recvBuf: recvBuf,
		config:  channelConf,
	}
}

// createChannel creates the default channel config
func getDefaultChannelConfig() ChannelConfig {
	return ChannelConfig{
		priority: 0,
	}
}

// getID returns the ID of the channel
func (ch *Channel) getID() common.ChannelIDEnum {
	return ch.id
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
func (ch *Channel) receivePacket(packet *Packet) ([]byte, bool) {
	bytes, success := ch.recvBuf.receivePacket(packet)
	return bytes, success
}

// sendPacketTo serializes and sends the next packet to the given writer
func (ch *Channel) sendPacketTo(conn *Connection) (nonemptyPacket bool, numBytes int, err error) {
	packet := ch.sendBuf.emitPacket(ch.id)
	if packet.isEmpty() {
		return false, int(0), nil
	}

	err = conn.writePacket(&packet)
	if err != nil {
		return true, int(0), nil
	}
	numBytes = 0

	// numBytes, err = writer.Write(packetBytes)
	return true, numBytes, err
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
