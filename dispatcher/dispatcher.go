package dispatcher

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
)

type Dispatcher struct {
	messenger *p2p.Messenger
}

var dispatcher *Dispatcher

func GetDispatcher() *Dispatcher {
	if dispatcher == nil {
		dispatcher = &Dispatcher{}
	}
	return dispatcher
}

func (dp *Dispatcher) SetMessenger(messenger *p2p.Messenger) {
	dp.messenger = messenger
}

func (dp *Dispatcher) OnStart() error {
	return nil
}

func (dp *Dispatcher) OnStop() {

}

func (dp *Dispatcher) GetInventory(syncType common.SyncType) {

}

func (dp *Dispatcher) SendInventory() {

}

func (dp *Dispatcher) GetData() {

}

func (dp *Dispatcher) SendData() {

}

func (dp *Dispatcher) AddPeer() {

}
