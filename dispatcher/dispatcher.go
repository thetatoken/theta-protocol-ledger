package dispatcher

import (
	"context"
	"reflect"
	"sync"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/p2pl"
)

//
// Dispatcher dispatches messages to approporiate destinations
//
type Dispatcher struct {
	p2pnet  p2p.Network
	p2plnet p2pl.Network

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewLDispatcher returns the pointer to the Dispatcher singleton
func NewDispatcher(p2pnet p2p.Network, p2plnet p2pl.Network) *Dispatcher {
	return &Dispatcher{
		p2pnet:  p2pnet,
		p2plnet: p2plnet,
		wg:      &sync.WaitGroup{},
	}
}

// Start is called when the dispatcher starts
func (dp *Dispatcher) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	dp.ctx = c
	dp.cancel = cancel
	var err error

	if !reflect.ValueOf(dp.p2pnet).IsNil() {
		err = dp.p2pnet.Start(c)
		if err != nil {
			return err
		}
	}
	if !reflect.ValueOf(dp.p2plnet).IsNil() {
		err = dp.p2plnet.Start(c)
	}
	return err
}

// Stop is called when the dispatcher stops
func (dp *Dispatcher) Stop() {
	dp.cancel()
}

// Wait suspends the caller goroutine
func (dp *Dispatcher) Wait() {
	if !reflect.ValueOf(dp.p2pnet).IsNil() {
		dp.p2pnet.Wait()
	}
	if !reflect.ValueOf(dp.p2plnet).IsNil() {
		dp.p2plnet.Wait()
	}
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

// Peers returns the IDs of all peers
func (dp *Dispatcher) Peers() []string {
	if !reflect.ValueOf(dp.p2pnet).IsNil() {
		return dp.p2pnet.Peers()
	}
	if !reflect.ValueOf(dp.p2plnet).IsNil() {
		return dp.p2plnet.Peers()
	}
	return []string{}
}

func (dp *Dispatcher) send(peerIDs []string, channelID common.ChannelIDEnum, content interface{}) {
	messageOld := p2ptypes.Message{
		ChannelID: channelID,
		Content:   content,
	}
	message := p2ptypes.Message{
		ChannelID: channelID,
		Content:   content,
	}
	if len(peerIDs) == 0 {
		if !reflect.ValueOf(dp.p2pnet).IsNil() {
			dp.p2pnet.Broadcast(messageOld)
		}
		if !reflect.ValueOf(dp.p2plnet).IsNil() {
			if message.ChannelID == common.ChannelIDGuardian {
				// Send guardian vote to immediate neighbors only.
				dp.p2plnet.BroadcastToNeighbors(message)
			} else {
				dp.p2plnet.Broadcast(message)
			}
		}
	} else {
		for _, peerID := range peerIDs {
			go func(peerID string) {
				if !reflect.ValueOf(dp.p2pnet).IsNil() {
					dp.p2pnet.Send(peerID, messageOld)
				}
				if !reflect.ValueOf(dp.p2plnet).IsNil() {
					dp.p2plnet.Send(peerID, message)
				}
			}(peerID)
		}
	}
}
