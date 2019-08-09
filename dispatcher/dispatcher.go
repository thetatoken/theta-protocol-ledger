package dispatcher

import (
	"context"
	"sync"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p"
	// p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/p2pl"
	p2pltypes "github.com/thetatoken/theta/p2pl/types"
)

//
// Dispatcher dispatches messages to approporiate destinations
//
type Dispatcher struct {
	p2pnet p2p.Network
	p2plnet p2pl.Network

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewDispatcher returns the pointer to the Dispatcher singleton
func NewDispatcher(p2pnet p2p.Network) *Dispatcher {
	return &Dispatcher{
		p2pnet: p2pnet,
		wg:     &sync.WaitGroup{},
	}
}

// NewLDispatcher returns the pointer to the Dispatcher singleton
func NewLDispatcher(p2pnet p2pl.Network) *Dispatcher {
	return &Dispatcher{
		p2plnet: p2pnet,
		wg:     &sync.WaitGroup{},
	}
}

// Start is called when the dispatcher starts
func (dp *Dispatcher) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	dp.ctx = c
	dp.cancel = cancel

	err := dp.p2plnet.Start(c)
	return err
}

// Stop is called when the dispatcher stops
func (dp *Dispatcher) Stop() {
	dp.cancel()
}

// Wait suspends the caller goroutine
func (dp *Dispatcher) Wait() {
	dp.p2plnet.Wait()
	dp.wg.Wait()
}

// GetInventory sends out the InventoryRequest
func (dp *Dispatcher) GetInventory(peerIDs []string, invreq InventoryRequest) {
	dp.send(peerIDs, invreq.ChannelID, invreq)
}

// SendInventory sends out the InventoryResponse
func (dp *Dispatcher) SendInventory(peerIDs []string, invrsp InventoryResponse) {
	dp.send(peerIDs, invrsp.ChannelID, invrsp)
}

// GetData sends out the DataRequest
func (dp *Dispatcher) GetData(peerIDs []string, datareq DataRequest) {
	dp.send(peerIDs, datareq.ChannelID, datareq)
}

// SendData sends out the DataResponse
func (dp *Dispatcher) SendData(peerIDs []string, datarsp DataResponse) {
	dp.send(peerIDs, datarsp.ChannelID, datarsp)
}

func (dp *Dispatcher) send(peerIDs []string, channelID common.ChannelIDEnum, content interface{}) {
	message := p2pltypes.Message{
		ChannelID: channelID,
		Content:   content,
	}
	if len(peerIDs) == 0 {
		dp.p2plnet.Broadcast(message)
	} else {
		for _, peerID := range peerIDs {
			go func(peerID string) {
				dp.p2plnet.Send(peerID, message)
			}(peerID)
		}
	}
}
