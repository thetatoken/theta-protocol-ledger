package messenger

import (
	"errors"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/p2p"
	pr "github.com/thetatoken/ukulele/p2p/peer"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

//
// Messenger implements the Network interface
//
type Messenger struct {
	msgHandlerMap map[string](*p2p.MessageHandler) // Map: handlerName -> messageHandler

	peerListeners []PeerListener
	peerTable     PeerTable
	nodeInfo      p2ptypes.NodeInfo

	config MessengerConfig
}

type MessengerConfig struct {
}

func CreateMessenger(nodeInfo p2ptypes.NodeInfo, config MessengerConfig) *Messenger {
	return &Messenger{
		nodeInfo: nodeInfo,
		config:   config,
	}
}

func CreateDefaultMessengerConfig() MessengerConfig {
	return MessengerConfig{}
}

func (msgr *Messenger) Send(peerKey string, message p2p.Message) bool {
	peer := msgr.peerTable.getPeer(peerKey)
	if peer == nil {
		return false
	}

	success := peer.Send(message.ChannelID, message.Content)

	return success
}

func (msgr *Messenger) Broadcast(message p2p.Message) (successes chan bool) {
	allPeers := msgr.peerTable.getAllPeers()
	successes = make(chan bool, len(*allPeers))
	for _, peer := range *allPeers {
		go func(peer *pr.Peer) {
			success := msgr.Send(peer.Key(), message)
			successes <- success
		}(peer)
	}
	return successes
}

func (msgr *Messenger) AddPeer(peer *pr.Peer) error {
	selfKeyBytes := crypto.PubkeyToAddress(msgr.nodeInfo.PubKey)
	selfKey := string(selfKeyBytes[:])
	if peer.Key() == selfKey {
		errMsg := "[p2p] Cannot connect to self"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if err := peer.Handshake(); err != nil {
		return err
	}

	if !peer.OnStart() {
		errMsg := "[p2p] Failed to start peer"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	if !msgr.peerTable.addPeer(peer) {
		errMsg := "[p2p] Failed to add peer to the peerTable"
		log.Errorf(errMsg)
		return errors.New(errMsg)
	}

	for _, msgHandler := range msgr.msgHandlerMap {
		(*msgHandler).AttachToPeer(peer)
	}

	return nil
}

func (msgr *Messenger) AddMessageHandler(name string, msgHandler *p2p.MessageHandler) {
	msgr.msgHandlerMap[name] = msgHandler
}

func (msgr *Messenger) AddPeerListener() {

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
