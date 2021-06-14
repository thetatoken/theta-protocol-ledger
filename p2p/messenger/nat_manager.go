package messenger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p/nat"
	pr "github.com/thetatoken/theta/p2p/peer"
	"github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

const (
	natMappingPulseInterval = 1 * time.Minute
)

// NatMappingMessage defines the structure of the NAT mapping message
type NATMappingMessage struct {
	EPort uint16
}

type NATManager struct {
	port  int
	eport int

	natDevice nat.NAT

	messenger *Messenger
	peerTable *pr.PeerTable

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

func CreateNATManager(port int) *NATManager {
	nmgr := &NATManager{
		natDevice: nil,
		port:      port,
		wg:        &sync.WaitGroup{},
	}
	return nmgr
}

// DiscoverGateway discovers the gateway for the NAT mapping
func (nmgr *NATManager) DiscoverGateway() error {
	logger.Infof("Discovering NAT gateway...")
	natDevice, err := nat.DiscoverGateway()
	if err != nil {
		nmgr.natDevice = nil
		logger.Warnf("Failed to detect the NAT device: %v", err)
		return err
	}

	logger.Infof("NAT type: %s", natDevice.Type())
	nmgr.natDevice = natDevice
	return nil
}

// SetMessenger sets the Messenger for the NATManager
func (nmgr *NATManager) SetMessenger(msgr *Messenger) {
	nmgr.messenger = msgr
	nmgr.peerTable = &msgr.peerTable
}

// Start is called when the NATManager instance starts
func (nmgr *NATManager) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	nmgr.ctx = c
	nmgr.cancel = cancel

	if nmgr.natDevice != nil {
		nmgr.wg.Add(1)
		go nmgr.maintainNATMappingRoutine()
	}

	return nil
}

// Wait suspends the caller goroutine
func (nmgr *NATManager) Wait() {
	nmgr.wg.Wait()
}

// Stop is called when the NATManager instance stops
func (nmgr *NATManager) Stop() {
	nmgr.cancel()
}

func (nmgr *NATManager) maintainNATMappingRoutine() {
	defer nmgr.wg.Done()

	natMappingPulse := time.NewTicker(natMappingPulseInterval)
	for {
		select {
		case <-natMappingPulse.C:
			nmgr.maintainNATMapping()
		}
	}
}

func (nmgr *NATManager) maintainNATMapping() {
	eport, err := nmgr.NatMapping(nmgr.port)
	if err != nil {
		logger.Warnf("Failed to perform NAT mapping: %v", err)
	}

	if nmgr.eport != eport {
		// notify peers
		content := NATMappingMessage{
			EPort: uint16(eport),
		}
		message := types.Message{
			ChannelID: common.ChannelIDNATMapping,
			Content:   content,
		}

		nmgr.messenger.Broadcast(message, false /* should inform both blockchain and edge nodes */)

		nmgr.eport = eport

		logger.Debugf("Notify peers with the new external port: %v", eport)
	}
}

func (nmgr *NATManager) NatMapping(port int) (eport int, err error) {
	if nmgr.natDevice == nil {
		return port, fmt.Errorf("No available NAT device")
	}

	iaddr, err := nmgr.natDevice.GetInternalAddress()
	if err != nil {
		return port, err
	}
	logger.Infof("Internal address: %s", iaddr)

	eaddr, err := nmgr.natDevice.GetExternalAddress()
	if err != nil {
		return port, err
	}
	logger.Infof("External address: %s", eaddr)

	eport, err = nmgr.natDevice.AddPortMapping("tcp", port, "tcp", 60*time.Second)
	if err != nil {
		return port, err
	}
	logger.Infof("External port for %v is %v", port, eport)

	return eport, nil
}

// GetChannelIDs implements the p2p.MessageHandler interface
func (nmgr *NATManager) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDNATMapping,
	}
}

// EncodeMessage implements the p2p.MessageHandler interface
func (nmgr *NATManager) EncodeMessage(message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

// ParseMessage implements the p2p.MessageHandler interface
func (nmgr *NATManager) ParseMessage(peerID string,
	channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error) {
	mappingMsg, err := decodeNATMappingMessage(rawMessageBytes)
	message := types.Message{
		PeerID:    peerID,
		ChannelID: channelID,
		Content:   mappingMsg,
	}
	if err != nil {
		logger.Errorf("Error decoding NATMappingMessage: %v", err)
		return message, err
	}

	return message, nil
}

// HandleMessage implements the p2p.MessageHandler interface
func (nmgr *NATManager) HandleMessage(msg types.Message) error {
	if msg.ChannelID != common.ChannelIDNATMapping {
		errMsg := fmt.Sprintf("Invalid channelID for the NATMappingMessageHandler: %v", msg.ChannelID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	peerID := msg.PeerID
	peer := nmgr.peerTable.GetPeer(peerID)
	if peer == nil {
		errMsg := fmt.Sprintf("Cannot find peer %v in the peer table", peerID)
		logger.Errorf(errMsg)
		return errors.New(errMsg)
	}

	natMsg := (msg.Content).(NATMappingMessage)
	peerAddr := peer.NetAddress()
	peerAddr.Port = natMsg.EPort
	peer.SetNetAddress(peerAddr)

	logger.Debugf("Update peer address for %v - external port: %v, peerAddr: %v", peer.ID(), peerAddr.Port, peerAddr.String())

	return nil
}

func decodeNATMappingMessage(msgBytes common.Bytes) (message NATMappingMessage, err error) {
	err = rlp.DecodeBytes(msgBytes, &message)
	return
}
