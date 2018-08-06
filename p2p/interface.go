package p2p

// MessageHandler defines the interface for handling network messages.
type MessageHandler interface {
	HandleMessage(self Network, msg interface{})
}

// Network is a handle to the P2P network.
type Network interface {
	Broadcast(msg interface{}) error
	Send(ID string, msg interface{}) error

	AddMessageHandler(handler MessageHandler)

	ID() string
}
