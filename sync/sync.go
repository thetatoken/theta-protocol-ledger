package sync

import (
	"context"
	"sync"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/dispatcher"
)

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

	mu              *sync.Mutex
	incomingMsgChan chan interface{}
	epoch           uint32
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
		case msg := <-sm.incomingMsgChan:
			sm.handleMessage(msg)
		}
	}
}

func (sm *SyncManager) handleMessage(msg interface{}) {
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

	nextBlock := sm.orphanBlockPool.TryGetNextBlock(block.Hash)
	if nextBlock != nil {
		sm.handleMessage(nextBlock)
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
