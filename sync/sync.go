package sync

import (
	"context"
	"sync"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/p2p"

	p2ptypes "github.com/thetatoken/ukulele/p2p/types"

	"github.com/spf13/viper"
)

var _ p2p.MessageHandler = (*SyncManager)(nil)

// SyncManager is an intermediate layer between consensus engine and p2p network. Its main responsibilities are to manage
// fast blocks sync among peers and buffer orphaned block/CC. Otherwise messages are passed through to consensus engine.
type SyncManager struct {
	chain           *blockchain.Chain
	consensus       consensus.Engine
	requestMgr      *RequestManager
	orphanBlockPool *OrphanBlockPool
	orphanCCPool    *OrphanCCPool

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	mu       *sync.Mutex
	incoming chan interface{}
	epoch    uint32
}

func NewSyncManager(chain *blockchain.Chain, consensus consensus.Engine) *SyncManager {
	return &SyncManager{
		chain:           chain,
		consensus:       consensus,
		requestMgr:      NewRequestManager(),
		orphanBlockPool: NewOrphanBlockPool(),
		orphanCCPool:    NewOrphanCCPool(),

		wg: &sync.WaitGroup{},

		mu:       &sync.Mutex{},
		incoming: make(chan interface{}, viper.GetInt(common.CfgSyncMessageQueueSize)),
	}

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
func (sm *SyncManager) HandleMessage(peerID string, msg p2ptypes.Message) {
	sm.incoming <- msg.Content
}

func (sm *SyncManager) processMessage(msg interface{}) {
	switch m := msg.(type) {
	case blockchain.Block:
		sm.handleBlock(&m)
	case blockchain.CommitCertificate:
		sm.handleCC(&m)
	case dispatcher.InventoryResponse:
		sm.handleInvResponse(&m)
	case dispatcher.InventoryRequest:
		sm.handleInvRequest(&m)
	}
}

func (sm *SyncManager) handleBlock(block *blockchain.Block) {
	if sm.chain.IsOrphan(block) {
		sm.orphanBlockPool.Add(block)
		sm.requestMgr.EnqueueBlocks(block.Hash)
		return
	}

	sm.consensus.AddMessage(block)

	cc := sm.orphanCCPool.TryGetCCByBlockHash(block.Hash)
	if cc != nil {
		sm.processMessage(cc)
	}

	nextBlock := sm.orphanBlockPool.TryGetNextBlock(block.Hash)
	if nextBlock != nil {
		sm.processMessage(nextBlock)
	}
}

func (sm *SyncManager) handleCC(cc *blockchain.CommitCertificate) {
	if block, _ := sm.chain.FindBlock(cc.BlockHash); block == nil {
		sm.orphanCCPool.Add(cc)
		sm.requestMgr.EnqueueBlocks(cc.BlockHash)
		return
	}

	sm.consensus.AddMessage(cc)
}

func (sm *SyncManager) handleInvResponse(invResp *dispatcher.InventoryResponse) {
	sm.requestMgr.handleInvResponse(invResp)
}

func (sm *SyncManager) handleInvRequest(invResp *dispatcher.InventoryRequest) {}
