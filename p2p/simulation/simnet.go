package simulation

import (
	"context"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

// Envelope wraps a message with network information for delivery.
type Envelope struct {
	From    string
	To      string
	Content interface{}
}

// Simnet represents an instance of simulated network.
type Simnet struct {
	Endpoints  []*SimnetEndpoint
	msgHandler p2p.MessageHandler
	messages   chan Envelope
	MsgLogs    []Envelope

	// Life cycle.
	wg      *sync.WaitGroup
	mu      *sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewSimnet creates a new instance of Simnet.
func NewSimnet() *Simnet {
	return &Simnet{
		messages: make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
		MsgLogs:  []Envelope{},
		wg:       &sync.WaitGroup{},
		mu:       &sync.Mutex{},
	}
}

// NewSimnetWithHandler creates a new instance of Simnet with given MessageHandler as the default handler.
func NewSimnetWithHandler(msgHandler p2p.MessageHandler) *Simnet {
	return &Simnet{
		msgHandler: msgHandler,
		messages:   make(chan Envelope, viper.GetInt(common.CfgP2PMessageQueueSize)),
		wg:         &sync.WaitGroup{},
		mu:         &sync.Mutex{},
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
func (sn *Simnet) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	sn.ctx = c
	sn.cancel = cancel

	for _, endpoint := range sn.Endpoints {
		endpoint.Start(ctx)
	}

	go sn.mainLoop()
}

// Stop notifies all goroutines to stop without blocking.
func (sn *Simnet) Stop() {
	sn.cancel()
}

// Wait blocks until all goroutines have stopped.
func (sn *Simnet) Wait() {
	sn.wg.Wait()
}

func (sn *Simnet) mainLoop() {
	sn.wg.Add(1)
	defer sn.wg.Done()

	for {
		select {
		case <-sn.ctx.Done():
			sn.mu.Lock()
			sn.stopped = true
			sn.mu.Unlock()
			return
		case envelope := <-sn.messages:
			time.Sleep(1 * time.Microsecond)
			for _, endpoint := range sn.Endpoints {
				if (envelope.To == "" && envelope.From != endpoint.ID()) || envelope.To == endpoint.ID() {
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
}

// AddMessage send a message through the network.
func (sn *Simnet) AddMessage(msg Envelope) {
	sn.mu.Lock()
	defer sn.mu.Unlock()

	if sn.stopped {
		return
	}

	sn.MsgLogs = append(sn.MsgLogs, msg)
	sn.messages <- msg
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

// Start implements the Network interface. It starts goroutines to receive/send message from network.
func (se *SimnetEndpoint) Start(ctx context.Context) error {
	go func() {
		for {
			select {
			case envelope := <-se.incoming:
				message := p2ptypes.Message{
					PeerID:  envelope.From,
					Content: envelope.Content,
				}
				se.HandleMessage(message)
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

// Stop implements the Network interface.
func (se *SimnetEndpoint) Stop() {
}

// Wait blocks until all goroutines have stopped.
func (se *SimnetEndpoint) Wait() {
}

// Broadcast implements the Network interface.
func (se *SimnetEndpoint) Broadcast(message p2ptypes.Message, skipEdgeNode bool) (successes chan bool) {
	successes = make(chan bool, 10)
	go func() {
		se.network.AddMessage(Envelope{From: se.ID(), Content: message.Content})
		successes <- true
	}()
	return successes
}

// BroadcastToNeighbors implements the Network interface.
func (se *SimnetEndpoint) BroadcastToNeighbors(message p2ptypes.Message, maxNumPeersToBroadcast int, skipEdgeNode bool) (successes chan bool) {
	successes = make(chan bool, 10)
	go func() {
		se.network.AddMessage(Envelope{From: se.ID(), Content: message.Content})
		successes <- true
	}()
	return successes
}

// Send implements the Network interface.
func (se *SimnetEndpoint) Send(id string, message p2ptypes.Message) bool {
	go func() {
		se.network.AddMessage(Envelope{From: se.ID(), To: id, Content: message.Content})
	}()
	return true
}

// Peers returns the IDs of all peers
func (se *SimnetEndpoint) Peers(skipEdgeNode bool) []string {
	return []string{}
}

// PeerURLs returns the URLs of all peers
func (se *SimnetEndpoint) PeerURLs(skipEdgeNode bool) []string {
	return []string{}
}

// PeerExists indicates if the given peerID is a neighboring peer
func (se *SimnetEndpoint) PeerExists(peerID string) bool {
	return false
}

// RegisterMessageHandler implements the Network interface.
func (se *SimnetEndpoint) RegisterMessageHandler(handler p2p.MessageHandler) {
	se.handlers = append(se.handlers, handler)
}

// ID implements the Network interface.
func (se *SimnetEndpoint) ID() string {
	return se.id
}

// HandleMessage implements the MessageHandler interface.
func (se *SimnetEndpoint) HandleMessage(message p2ptypes.Message) error {
	for _, handler := range se.handlers {
		handler.HandleMessage(message)
	}
	if se.network.msgHandler != nil {
		se.network.msgHandler.HandleMessage(message)
	}
	return nil
}
