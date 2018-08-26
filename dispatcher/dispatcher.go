package dispatcher

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

//
// Dispatcher dispatches messages to approporiate destinations
//
type Dispatcher struct {
	p2pnet p2p.Network
}

var dispatcher *Dispatcher

// GetDispatcher returns the pointer to the Dispatcher singleton
func GetDispatcher() *Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{}
	}
	return dispatcher
}

// SetNetwork sets the p2p network handle for the dispatcher
func (dp *Dispatcher) SetP2PNetwork(p2pnet p2p.Network) {
	dp.p2pnet = p2pnet
}

// OnStart is called when the dispatcher starts
func (dp *Dispatcher) OnStart() error {
	err := dp.p2pnet.OnStart()
	return err
}

// OnStop is called when the dispatcher stops
func (dp *Dispatcher) OnStop() {
	dp.p2pnet.OnStop()
}

// GetInventory sents out the InventoryRequest
func (dp *Dispatcher) GetInventory(peerIDs []string, invreq InventoryRequest) {
	dp.send(peerIDs, invreq.ChannelID, invreq)
}

// SendInventory sents out the InventoryResponse
func (dp *Dispatcher) SendInventory(peerIDs []string, invrsp InventoryResponse) {
	dp.send(peerIDs, invrsp.ChannelID, invrsp)
}

// GetData sents out the DataRequest
func (dp *Dispatcher) GetData(peerIDs []string, datareq DataRequest) {
	dp.send(peerIDs, datareq.ChannelID, datareq)
}

// SendData sends out the DataResponse
func (dp *Dispatcher) SendData(peerIDs []string, datarsp DataResponse) {
	dp.send(peerIDs, datarsp.ChannelID, datarsp)
}

func (dp *Dispatcher) send(peerIDs []string, channelID common.ChannelIDEnum, content interface{}) {
	message := p2ptypes.Message{
		ChannelID: channelID,
		Content:   content,
	}
	if len(peerIDs) == 0 {
		dp.p2pnet.Broadcast(message)
	} else {
		for _, peerID := range peerIDs {
			go func(peerID string) {
				dp.p2pnet.Send(peerID, message)
			}(peerID)
		}
	}
}
