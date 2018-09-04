package netsync

import (
	"context"
	"sync"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/p2p"

	p2ptypes "github.com/thetatoken/ukulele/p2p/types"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	incoming chan *Message
	epoch    uint32
}

// Message represents an item to be process in queue. It allows us to save peerID with the message to be processed later.
type Message struct {
	peerID string
	data   interface{}
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
		incoming: make(chan *Message, viper.GetInt(common.CfgSyncMessageQueueSize)),
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
func (sm *SyncManager) ParseMessage(channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	// To be implemented..
	message := p2ptypes.Message{
		ChannelID: channelID,
	}
	return message, nil
}

// HandleMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) HandleMessage(peerID string, msg p2ptypes.Message) error {
	sm.AddMessage(&Message{peerID: peerID, data: msg.Content})
	return nil
}

func (sm *SyncManager) AddMessage(msg *Message) {
	sm.incoming <- msg
}

func (sm *SyncManager) AddData(data interface{}) {
	sm.AddMessage(&Message{data: data})
}

func (sm *SyncManager) processMessage(message *Message) {
	switch m := message.data.(type) {
	// Messages needed for fast-sync.
	case dispatcher.InventoryRequest:
		sm.requestMgr.handleInvRequest(message.peerID, &m)
	case dispatcher.InventoryResponse:
		sm.requestMgr.handleInvResponse(message.peerID, &m)
	case dispatcher.DataRequest:
		sm.requestMgr.handleDataRequest(message.peerID, &m)
	case dispatcher.DataResponse:
		sm.requestMgr.handleDataResponse(message.peerID, &m)
	default:
		sm.processData(message.data)
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
