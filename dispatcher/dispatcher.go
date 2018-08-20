package dispatcher

import (
	"github.com/thetatoken/ukulele/common"
	msgr "github.com/thetatoken/ukulele/p2p/messenger"
)

type Dispatcher struct {
	messenger *msgr.Messenger
}

var dispatcher *Dispatcher

func GetDispatcher() *Dispatcher {
	if dispatcher == nil {
		messengerConfig := msgr.CreateDefaultMessengerConfig()
		messenger := msgr.CreateMessenger(messengerConfig)
		dispatcher = &Dispatcher{
			messenger: messenger,
		}
	}
	return dispatcher
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
