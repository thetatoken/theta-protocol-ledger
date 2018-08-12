package connection

type RecvBuffer struct {
	workspace []byte

	config RecvBufferConfig
}

type RecvBufferConfig struct {
	workspaceCapacity int
}

// createRecvBuffer creates a RecvBuffer instance for the given config
func createRecvBuffer(config RecvBufferConfig) RecvBuffer {
	return RecvBuffer{
		workspace: make([]byte, 0, config.workspaceCapacity),
		config:    config,
	}
}

// getDefaultRecvBufferConfig returns the default config for the RecvBuffer
func getDefaultRecvBufferConfig() RecvBufferConfig {
	return RecvBufferConfig{
		workspaceCapacity: 4096,
	}
}

// receivePacket handles incoming msgPackets. It returns a msg bytes if msg is
// complete (i.e. ends with EOF). It is not go-routine safe
func (rb *RecvBuffer) receivePacket(packet Packet) ([]byte, bool) {
	if len(rb.workspace)+len(packet.Bytes) > rb.config.workspaceCapacity {
		return nil, false
	}
	rb.workspace = append(rb.workspace, packet.Bytes...)
	if packet.IsEOF == byte(0x01) {
		bytes := rb.workspace
		rb.workspace = make([]byte, 0, rb.config.workspaceCapacity)
		return bytes, true
	}
	return nil, true
}
