package simulation

import (
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

// Envelope wraps a message with network information for delivery.
type Envelope struct {
	From    string
	To      string
	Content interface{}
}

// Simnet represents an instance of simluated network.
type Simnet struct {
	Endpoints  []*SimnetEndpoint
	msgHandler p2p.MessageHandler
	messages   chan Envelope
}

// NewSimnet creates a new instance of Simnet.
func NewSimnet() *Simnet {
	return &Simnet{
		messages: make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
	}
}

// NewSimnetWithHandler creates a new instance of Simnet with given MessageHandler as the default handler.
func NewSimnetWithHandler(msgHandler p2p.MessageHandler) *Simnet {
	return &Simnet{
		msgHandler: msgHandler,
		messages:   make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
	}
}

// AddEndpoint adds an endpoint with given ID to the Simnet instance.
func (sn *Simnet) AddEndpoint(id string) *SimnetEndpoint {
	endpoint := &SimnetEndpoint{
		id:       id,
		network:  sn,
		incoming: make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
		outgoing: make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
	}
	sn.Endpoints = append(sn.Endpoints, endpoint)
	return endpoint
}

// Start is the main entry point for Simnet. It starts all endpoints and start a goroutine to handle message dlivery.
func (sn *Simnet) Start() {
	for _, endpoint := range sn.Endpoints {
		endpoint.OnStart()
	}

	go func() {
		for {
			select {
			case envelope := <-sn.messages:
				time.Sleep(1 * time.Microsecond)
				for _, endpoint := range sn.Endpoints {
					// Allow broadcast/send to self
					if envelope.To == "" || envelope.To == endpoint.ID() {
						go func(endpoint *SimnetEndpoint, envelope Envelope) {
							// Simulate network delay except for messages to self.
							if envelope.From != endpoint.ID() {
								// time.Sleep(100 * time.Millisecond)
							}
							endpoint.incoming <- envelope

						}(endpoint, envelope)
					}
				}
			}
		}
	}()
}

// SimnetEndpoint is the implementation of Network interface for Simnet.
type SimnetEndpoint struct {
	id       string
	network  *Simnet
	handlers []p2p.MessageHandler
	incoming chan Envelope
	outgoing chan Envelope
}

var _ p2p.Network = &SimnetEndpoint{}

// OnStart implements the Network interface. It starts goroutines to receive/send message from network.
func (se *SimnetEndpoint) OnStart() error {
	go func() {
		for {
			select {
			case envelope := <-se.incoming:
				if envelope.To == "" || envelope.To == se.ID() {
					peerID := se.ID()
					message := p2ptypes.Message{
						Content: envelope.Content,
					}
					se.HandleMessage(peerID, message)
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case envelope := <-se.outgoing:
				se.network.messages <- envelope
			}
		}
	}()

	return nil
}

// OnStop implements the Network interface.
func (se *SimnetEndpoint) OnStop() {
}

// Broadcast implements the Network interface.
func (se *SimnetEndpoint) Broadcast(message p2ptypes.Message) error {
	go func() {
		se.network.messages <- Envelope{From: se.ID(), Content: message.Content}
	}()
	return nil
}

// Send implements the Network interface.
func (se *SimnetEndpoint) Send(id string, message p2ptypes.Message) error {
	go func() {
		se.network.messages <- Envelope{From: se.ID(), To: id, Content: message.Content}
	}()
	return nil
}

// AddMessageHandler implements the Network interface.
func (se *SimnetEndpoint) AddMessageHandler(handler p2p.MessageHandler) {
	se.handlers = append(se.handlers, handler)
}

// ID implements the Network interface.
func (se *SimnetEndpoint) ID() string {
	return se.id
}

// HandleMessage implements the MessageHandler interface.
func (se *SimnetEndpoint) HandleMessage(peerID string, message p2ptypes.Message) {
	for _, handler := range se.handlers {
		handler.HandleMessage(peerID, message)
	}
	if se.network.msgHandler != nil {
		se.network.msgHandler.HandleMessage(peerID, message)
	}
}
