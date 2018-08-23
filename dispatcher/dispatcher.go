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
	messenger *p2p.Messenger
}

var dispatcher *Dispatcher

// GetDispatcher returns the pointer to the Dispatcher singleton
func GetDispatcher() *Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{}
	}
	return dispatcher
}

// SetMessenger sets the messenger for the dispatcher
func (dp *Dispatcher) SetMessenger(messenger *p2p.Messenger) {
	dp.messenger = messenger
}

// GetMessenger gets the messenger instance attached to the dispatcher
func (dp *Dispatcher) GetMessenger() *p2p.Messenger {
	return dp.messenger
}

// OnStart is called when the dispatcher starts
func (dp *Dispatcher) OnStart() error {
	err := dp.messenger.OnStart()
	return err
}

// OnStop is called when the dispatcher stops
func (dp *Dispatcher) OnStop() {
	dp.messenger.OnStop()
}

// GetInventory sents out the InventoryRequest
func (dp *Dispatcher) GetInventory(peerIDs []string, invreq InventoryRequest) {
	dp.send(peerIDs, invreq.Type, invreq)
}

// SendInventory sents out the InventoryResponse
func (dp *Dispatcher) SendInventory(peerIDs []string, invrsp InventoryResponse) {
	dp.send(peerIDs, invrsp.Type, invrsp)
}

// GetData sents out the DataRequest
func (dp *Dispatcher) GetData(peerIDs []string, datareq DataRequest) {
	dp.send(peerIDs, datareq.Type, datareq)
}

// SendData sends out the DataResponse
func (dp *Dispatcher) SendData(peerIDs []string, datarsp DataResponse) {
	dp.send(peerIDs, datarsp.Type, datarsp)
}

func (dp *Dispatcher) send(peerIDs []string, syncType common.SyncType, content interface{}) {
	message := p2ptypes.Message{
		ChannelID: dp.getChannelID(syncType),
		Content:   content,
	}
	if len(peerIDs) == 0 {
		dp.messenger.Broadcast(message)
	} else {
		for _, peerID := range peerIDs {
			go func(peerID string) {
				dp.messenger.Send(peerID, message)
			}(peerID)
		}
	}
}

func (dp *Dispatcher) getChannelID(syncType common.SyncType) common.ChannelIDEnum {
	switch syncType {
	case common.SyncCheckpoint:
		return common.ChannelIDCheckpoint
	case common.SyncHeader:
		return common.ChannelIDHeader
	case common.SyncBlock:
		return common.ChannelIDBlock
	case common.SyncVote:
		return common.ChannelIDVote
	case common.SyncTransaction:
		return common.ChannelIDTransaction
	default:
		return common.ChannelIDInvalid
	}
}
