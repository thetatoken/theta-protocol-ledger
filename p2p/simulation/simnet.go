package simulation

import (
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
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
	msgHandler MessageHandler
	messages   chan Envelope
}

// NewSimnet creates a new instance of Simnet.
func NewSimnet() *Simnet {
	return &Simnet{
		messages: make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
	}
}

// NewSimnetWithHandler creates a new instance of Simnet with given MessageHandler as the default handler.
func NewSimnetWithHandler(msgHandler MessageHandler) *Simnet {
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
		endpoint.Start()
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

var _ Network = &SimnetEndpoint{}

// Start starts goroutines to receive/send message from network.
func (se *SimnetEndpoint) Start() {
	go func() {
		for {
			select {
			case envelope := <-se.incoming:
				if envelope.To == "" || envelope.To == se.ID() {
					se.HandleMessage(envelope.Content)
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
}

// Broadcast implements Network interface.
func (se *SimnetEndpoint) Broadcast(msg interface{}) error {
	go func() {
		se.network.messages <- Envelope{From: se.ID(), Content: msg}
	}()
	return nil
}

// Send implements Network interface.
func (se *SimnetEndpoint) Send(id string, msg interface{}) error {
	go func() {
		se.network.messages <- Envelope{From: se.ID(), To: id, Content: msg}
	}()
	return nil
}

// AddMessageHandler implements Network interface.
func (se *SimnetEndpoint) AddMessageHandler(handler MessageHandler) {
	se.handlers = append(se.handlers, handler)
}

// ID implements Network interface.
func (se *SimnetEndpoint) ID() string {
	return se.id
}

// HandleMessage implements MessageHandler interface.
func (se *SimnetEndpoint) HandleMessage(msg interface{}) {
	for _, handler := range se.handlers {
		handler.HandleMessage(se, msg)
	}
	if se.network.msgHandler != nil {
		se.network.msgHandler.HandleMessage(se, msg)
	}
}
