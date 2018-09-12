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
		workspaceCapacity: 4 * 1024, // 4 KB
	}
}

// receivePacket handles incoming msgPackets. It returns a msg bytes if msg is
// complete (i.e. ends with EOF). It is not goroutine safe
func (rb *RecvBuffer) receivePacket(packet *Packet) ([]byte, bool) {
	// Here we disable the check below to allow arbitrarily long message
	// if len(rb.workspace)+len(packet.Bytes) > rb.config.workspaceCapacity {
	// 	return nil, false
	// }

	// TODO: out-of-order packet delivery handling?

	rb.workspace = append(rb.workspace, packet.Bytes...)
	if packet.IsEOF == byte(0x01) {
		bytes := rb.workspace

		// clear the slice without re-allocating.
		// http://stackoverflow.com/questions/16971741/how-do-you-clear-a-slice-in-go
		//   suggests this could be a memory leak, but we might as well keep the memory for the channel until it closes,
		//	at which point the recving slice stops being used and should be garbage collected
		rb.workspace = rb.workspace[:0] // make([]byte, 0, rb.config.workspaceCapacity)

		return bytes, true
	}
	return nil, true
}
