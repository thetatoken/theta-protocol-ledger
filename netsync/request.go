package netsync

import (
	"container/list"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/dispatcher"

	log "github.com/sirupsen/logrus"
)

const RequestTimeout = 10 * time.Second
const MinInventoryRequestInterval = 3 * time.Second
const RequestQuotaPerSecond = 1000

type RequestState uint8

const (
	RequestToSendDataReq = iota
	RequestWaitingDataResp
)

type PendingBlock struct {
	hash       common.Hash
	block      *core.Block
	peers      []string
	lastUpdate time.Time
	status     RequestState
}

func NewPendingBlock(x common.Hash, peerIds []string) *PendingBlock {
	return &PendingBlock{
		hash:       x,
		lastUpdate: time.Now(),
		peers:      peerIds,
		status:     RequestToSendDataReq,
	}
}

func (pb *PendingBlock) HasTimedOut() bool {
	return time.Since(pb.lastUpdate) > RequestTimeout
}

func (pb *PendingBlock) UpdateTimestamp() {
	pb.lastUpdate = time.Now()
}

type RequestManager struct {
	logger *log.Entry

	ticker *time.Ticker
	quota  int

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	syncMgr    *SyncManager
	chain      *blockchain.Chain
	dispatcher *dispatcher.Dispatcher

	lastInventoryRequest time.Time

	pendingBlocks         *list.List
	pendingBlocksByHash   map[string]*list.Element
	pendingBlocksByParent map[string][]*core.Block

	endHashCache      []common.Bytes
	blockRequestCache []common.Bytes
}

func NewRequestManager(syncMgr *SyncManager) *RequestManager {
	rm := &RequestManager{
		ticker: time.NewTicker(1 * time.Second),
		quota:  RequestQuotaPerSecond,

		wg: &sync.WaitGroup{},

		lastInventoryRequest: time.Now(),

		syncMgr:    syncMgr,
		chain:      syncMgr.chain,
		dispatcher: syncMgr.dispatcher,

		pendingBlocks:         list.New(),
		pendingBlocksByHash:   make(map[string]*list.Element),
		pendingBlocksByParent: make(map[string][]*core.Block),
	}

	logger := util.GetLoggerForModule("request")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": rm.syncMgr.consensus.ID()})
	}
	rm.logger = logger

	return rm
}

func (rm *RequestManager) mainLoop() {
	defer rm.wg.Done()

	for {
		select {
		case <-rm.ctx.Done():
			rm.stopped = true
			return
		case <-rm.ticker.C:
			rm.quota = RequestQuotaPerSecond
			rm.tryToDownload()
		}
	}
}

func (rm *RequestManager) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	rm.ctx = c
	rm.cancel = cancel

	rm.wg.Add(1)
	go rm.mainLoop()
}

func (rm *RequestManager) Stop() {
	rm.ticker.Stop()
	rm.cancel()
}

func (rm *RequestManager) Wait() {
	rm.wg.Wait()
}

func (rm *RequestManager) tryToDownload() {
	hasUndownloadedBlocks := rm.pendingBlocks.Len() > 0 || len(rm.pendingBlocksByHash) > 0 || len(rm.pendingBlocksByParent) > 0
	inventoryRequestIntervalPassed := time.Since(rm.lastInventoryRequest) >= MinInventoryRequestInterval
	if hasUndownloadedBlocks && inventoryRequestIntervalPassed {
		rm.logger.WithFields(log.Fields{
			"pendingBlocks":     rm.pendingBlocks.Len(),
			"orphan blocks":     len(rm.pendingBlocksByParent),
			"current chain tip": rm.syncMgr.consensus.GetTip(true).Hash().Hex(),
		}).Info("Fast sync in progress")

		rm.lastInventoryRequest = time.Now()
		tip := rm.syncMgr.consensus.GetTip(true)
		req := dispatcher.InventoryRequest{ChannelID: common.ChannelIDBlock, Start: tip.Hash().String()}
		rm.logger.WithFields(log.Fields{
			"channelID": req.ChannelID,
			"startHash": req.Start,
			"endHash":   req.End,
		}).Debug("Sending inventory request")
		rm.syncMgr.dispatcher.GetInventory([]string{}, req)
	}

	for curr := rm.pendingBlocks.Front(); rm.quota != 0 && curr != nil; curr = curr.Next() {
		pendingBlock := curr.Value.(*PendingBlock)
		if pendingBlock.block != nil {
			continue
		}
		if len(pendingBlock.peers) == 0 {
			continue
		}
		if pendingBlock.status == RequestToSendDataReq ||
			(pendingBlock.status == RequestWaitingDataResp && pendingBlock.HasTimedOut()) {
			randomPeerID := pendingBlock.peers[rand.Intn(len(pendingBlock.peers))]
			request := dispatcher.DataRequest{
				ChannelID: common.ChannelIDBlock,
				Entries:   []string{pendingBlock.hash.String()},
			}
			rm.logger.WithFields(log.Fields{
				"channelID":       request.ChannelID,
				"request.Entries": request.Entries,
				"peer":            randomPeerID,
			}).Debug("Sending data request")
			rm.syncMgr.dispatcher.GetData([]string{randomPeerID}, request)
			pendingBlock.UpdateTimestamp()
			pendingBlock.status = RequestWaitingDataResp
			rm.quota--
			continue
		}
	}
}

func (rm *RequestManager) AddHash(x common.Hash, peerIDs []string) {
	if _, err := rm.chain.FindBlock(x); err == nil {
		return
	}

	var pendingBlockEl *list.Element
	var pendingBlock *PendingBlock
	pendingBlockEl, ok := rm.pendingBlocksByHash[x.String()]
	if !ok {
		pendingBlock = NewPendingBlock(x, peerIDs)
		pendingBlockEl = rm.pendingBlocks.PushBack(pendingBlock)
		rm.pendingBlocksByHash[x.String()] = pendingBlockEl
	}
	// Add peerIDs to pendingBlock.peers
	pendingBlock = pendingBlockEl.Value.(*PendingBlock)
	if pendingBlock.block != nil {
		return
	}
	for _, xid := range peerIDs {
		found := false
		for _, id := range pendingBlock.peers {
			if id == xid {
				found = true
				break
			}
		}
		if !found {
			pendingBlock.peers = append(pendingBlock.peers, xid)
		}
	}
}

func (rm *RequestManager) AddBlock(block *core.Block) {
	if _, err := rm.chain.FindBlock(block.Hash()); err == nil {
		return
	}
	if pendingBlockEl, ok := rm.pendingBlocksByHash[block.Hash().String()]; ok {
		pendingBlock := pendingBlockEl.Value.(*PendingBlock)
		pendingBlock.block = block
	}
	parent := block.Parent
	if _, err := rm.chain.FindBlock(parent); err == nil {
		rm.dumpReadyBlocks(block)
		return
	}
	byParents, ok := rm.pendingBlocksByParent[parent.String()]
	if !ok {
		byParents = []*core.Block{}
	}
	found := false
	for _, child := range byParents {
		if child.Hash() == block.Hash() {
			found = true
			break
		}
	}
	if !found {
		byParents = append(byParents, block)
	}
	rm.pendingBlocksByParent[parent.String()] = byParents
}

func (rm *RequestManager) dumpReadyBlocks(block *core.Block) {
	queue := []*core.Block{block}
	for len(queue) > 0 {
		block := queue[0]
		hash := block.Hash().String()
		queue = queue[1:]

		if children, ok := rm.pendingBlocksByParent[hash]; ok {
			queue = append(queue, children...)
			delete(rm.pendingBlocksByParent, hash)
		}

		if pendingBlockEl, ok := rm.pendingBlocksByHash[hash]; ok {
			rm.pendingBlocks.Remove(pendingBlockEl)
			delete(rm.pendingBlocksByHash, hash)
		}

		_, err := rm.chain.AddBlock(block)
		if err != nil {
			rm.logger.Panic(err)
		}
		rm.syncMgr.PassdownMessage(block)
	}
}
