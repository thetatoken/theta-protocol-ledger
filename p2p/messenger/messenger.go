package messenger

import (
	"github.com/thetatoken/ukulele/p2p"
)

//
// Messenger implements the Network interface
//
type Messenger struct {
	peerTable PeerTable
	nodeInfo  p2p.NodeInfo
}

type MessengerConfig struct {
}

func CreateMessenger() {

}

func (msgr *Messenger) Send() {

}

func (msgr *Messenger) Broadcast() {

}

func (msgr *Messenger) AddPeer() {

}

func (msgr *Messenger) AddReactor() {

}

func (msgr *Messenger) AddListener() {

}

func (msgr *Messenger) DialSeedPeers() {

}

func (msgr *Messenger) GetAllPeers() {

}

func (msgr *Messenger) StopPeer() {

}

func (msgr *Messenger) StopPeerForError() {

}

func (msgr *Messenger) listenerRoutine() {

}
