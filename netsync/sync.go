package netsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/p2p"
	"github.com/thetatoken/ukulele/rlp"

	p2ptypes "github.com/thetatoken/ukulele/p2p/types"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type MessageIDEnum uint8

const (
	MessageIDBlock MessageIDEnum = iota
	MessageIDVote
	MessageIDProposal
	MessageIDCommitCertificate
	MessageIDInvRequest
	MessageIDInvResponse
	MessageIDDataRequest
	MessageIDDataResponse
)

var _ p2p.MessageHandler = (*SyncManager)(nil)

// SyncManager is an intermediate layer between consensus engine and p2p network. Its main responsibilities are to manage
// fast blocks sync among peers and buffer orphaned block/CC. Otherwise messages are passed through to consensus engine.
type SyncManager struct {
	chain           *blockchain.Chain
	consensus       consensus.Engine
	dispatcher      *dispatcher.Dispatcher
	requestMgr      *RequestManager
	orphanBlockPool *OrphanBlockPool
	orphanCCPool    *OrphanCCPool

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	mu       *sync.Mutex
	incoming chan *p2ptypes.Message
	epoch    uint32
}

func NewSyncManager(chain *blockchain.Chain, cons consensus.Engine, network p2p.Network, disp *dispatcher.Dispatcher) *SyncManager {
	sm := &SyncManager{
		chain:           chain,
		consensus:       cons,
		dispatcher:      disp,
		orphanBlockPool: NewOrphanBlockPool(),
		orphanCCPool:    NewOrphanCCPool(),

		wg: &sync.WaitGroup{},

		mu:       &sync.Mutex{},
		incoming: make(chan *p2ptypes.Message, viper.GetInt(common.CfgSyncMessageQueueSize)),
	}
	sm.requestMgr = NewRequestManager(sm)
	network.RegisterMessageHandler(sm)
	return sm
}

func (sm *SyncManager) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	sm.ctx = c
	sm.cancel = cancel

	sm.wg.Add(1)
	go sm.incomingMsgLoop()
}

func (sm *SyncManager) Stop() {
	sm.cancel()
}

func (sm *SyncManager) Wait() {
	sm.wg.Wait()
}

func (sm *SyncManager) incomingMsgLoop() {
	defer sm.wg.Done()

	for {
		select {
		case <-sm.ctx.Done():
			sm.stopped = true
			return
		case msg := <-sm.incoming:
			sm.processMessage(msg)
		}
	}
}

// GetChannelIDs implements the p2p.MessageHandler interface.
func (sm *SyncManager) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDCC,
		common.ChannelIDVote,
	}
}

// ParseMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) ParseMessage(peerID string, channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
	}

	var msgID MessageIDEnum
	err := rlp.DecodeBytes(rawMessageBytes[:1], &msgID)
	if err != nil {
		return message, err
	}

	if msgID == MessageIDInvRequest {
		data := dispatcher.InventoryRequest{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDInvResponse {
		data := dispatcher.InventoryResponse{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDDataRequest {
		data := dispatcher.DataRequest{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDDataResponse {
		data := dispatcher.DataResponse{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDBlock {
		data := blockchain.Block{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDProposal {
		data := consensus.Proposal{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDVote {
		data := blockchain.Vote{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else if msgID == MessageIDCommitCertificate {
		data := blockchain.CommitCertificate{}
		err = rlp.DecodeBytes(rawMessageBytes[1:], &data)
		message.Content = data
		return message, err
	} else {
		return message, fmt.Errorf("Unknown message ID: %v", msgID)
	}
}

// EncodeMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) EncodeMessage(message interface{}) (common.Bytes, error) {
	var buf bytes.Buffer
	var msgID MessageIDEnum
	switch message.(type) {
	case dispatcher.InventoryRequest:
		msgID = MessageIDInvRequest
	case dispatcher.InventoryResponse:
		msgID = MessageIDInvResponse
	case dispatcher.DataRequest:
		msgID = MessageIDDataRequest
	case dispatcher.DataResponse:
		msgID = MessageIDDataResponse
	case consensus.Proposal:
		msgID = MessageIDProposal
	case blockchain.Block:
		msgID = MessageIDBlock
	case blockchain.CommitCertificate:
		msgID = MessageIDCommitCertificate
	default:
		return nil, errors.New("Unsupported message type")
	}
	err := rlp.Encode(&buf, msgID)
	if err != nil {
		return nil, err
	}
	err = rlp.Encode(&buf, message)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// HandleMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) HandleMessage(msg p2ptypes.Message) error {
	sm.AddMessage(&msg)
	return nil
}

func (sm *SyncManager) AddMessage(msg *p2ptypes.Message) {
	sm.incoming <- msg
}

func (sm *SyncManager) processMessage(message *p2ptypes.Message) {
	switch content := message.Content.(type) {
	// Messages needed for fast-sync.
	case dispatcher.InventoryRequest:
		sm.requestMgr.handleInvRequest(message.PeerID, &content)
	case dispatcher.InventoryResponse:
		sm.requestMgr.handleInvResponse(message.PeerID, &content)
	case dispatcher.DataRequest:
		sm.requestMgr.handleDataRequest(message.PeerID, &content)
	case dispatcher.DataResponse:
		sm.requestMgr.handleDataResponse(message.PeerID, &content)
	default:
		sm.processData(message.Content)
	}
}

func (sm *SyncManager) processData(data interface{}) {
	switch d := data.(type) {
	// Messages need to be preprocessed.
	case consensus.Proposal:
		sm.handleProposal(&d)
	case blockchain.Block:
		sm.handleBlock(&d)
	case blockchain.CommitCertificate:
		sm.handleCC(&d)
	default:
		// Other messages are passed through to consensus engine.
		sm.consensus.AddMessage(d)
	}
}

func (sm *SyncManager) handleProposal(p *consensus.Proposal) {
	if p.CommitCertificate != nil {
		sm.handleCC(p.CommitCertificate)
	}
	sm.handleBlock(&p.Block)
}

func (sm *SyncManager) handleBlock(block *blockchain.Block) {
	if sm.chain.IsOrphan(block) {
		sm.orphanBlockPool.Add(block)
		sm.requestMgr.enqueueBlocks(block.Hash)
		log.WithFields(log.Fields{"id": sm.consensus.ID(), "block.Hash": block.Hash}).Debug("Received orphaned block")
		return
	}

	sm.consensus.AddMessage(block)

	cc := sm.orphanCCPool.TryGetCCByBlockHash(block.Hash)
	if cc != nil {
		sm.processData(cc)
	}

	nextBlock := sm.orphanBlockPool.TryGetNextBlock(block.Hash)
	if nextBlock != nil {
		sm.processData(nextBlock)
	}
}

func (sm *SyncManager) handleCC(cc *blockchain.CommitCertificate) {
	if block, _ := sm.chain.FindBlock(cc.BlockHash); block == nil {
		log.WithFields(log.Fields{"id": sm.consensus.ID(), "cc.BlockHash": cc.BlockHash}).Debug("Received orphaned CC")
		sm.orphanCCPool.Add(cc)
		sm.requestMgr.enqueueBlocks(cc.BlockHash)
		return
	}

	sm.consensus.AddMessage(cc)
}
