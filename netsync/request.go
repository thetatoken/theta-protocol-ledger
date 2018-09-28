package netsync

import (
	"bytes"
	"container/list"
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/dispatcher"

	log "github.com/sirupsen/logrus"
)

const RequestOutputChannelSize = 128
const WorkerPoolSize = 4
const RequestTimeout = 10 * time.Second
const RequestQuotaPerSecond = 100

type RequestState uint8

const (
	RequestCreated = iota
	RequestToSendInvReq
	RequestWaitingInvResp
	RequestToSendDataReq
	RequestWaitingDataResp
	RequestWaitingParent
)

type PendingBlock struct {
	hash       common.Bytes
	block      *core.Block
	peers      []string
	lastUpdate time.Time
	status     RequestState
}

func NewPendingBlock(x common.Bytes, peerIds []string) *PendingBlock {
	return &PendingBlock{
		hash:       x,
		lastUpdate: time.Now(),
		peers:      peerIds,
		status:     RequestCreated,
	}
}

func (pb *PendingBlock) HasTimedOut() bool {
	return time.Since(pb.lastUpdate) > RequestTimeout
}

func (pb *PendingBlock) UpdateTimestamp() {
	pb.lastUpdate = time.Now()
}

// type RequestWorker struct {
// 	work chan *PendingBlock

// 	wg      *sync.WaitGroup
// 	ctx     context.Context
// 	cancel  context.CancelFunc
// 	stopped bool
// }

// func NewRequestWorker(work chan *PendingBlock) *RequestWorker {
// 	return &RequestWorker{
// 		work: work,
// 		wg:   &sync.WaitGroup{},
// 	}
// }

// func (w *RequestWorker) mainLoop() {
// 	defer w.wg.Done()

// 	for {
// 		select {
// 		case <-w.ctx.Done():
// 			w.stopped = true
// 			return
// 		case pb := <-w.work:
// 			w.consensus.AddMessage(block)
// 		}
// 	}
// }

// func (w *RequestWorker) Start(ctx context.Context) {
// 	c, cancel := context.WithCancel(ctx)
// 	w.ctx = c
// 	w.cancel = cancel

// 	w.wg.Add(1)
// 	go w.mainLoop()
// }

// func (w *RequestWorker) Stop() {
// 	w.cancel()
// }

// func (w *RequestWorker) Wait() {
// 	w.wg.Wait()
// }

type RequestManager struct {
	logger *log.Entry

	C chan *core.Block

	ticker   *time.Ticker
	quota    int
	workBell chan struct{}
	work     chan *PendingBlock

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	syncMgr    *SyncManager
	chain      *blockchain.Chain
	dispatcher *dispatcher.Dispatcher

	pendingBlocks         *list.List
	pendingBlocksByHash   map[string]*list.Element
	pendingBlocksByParent map[string][]*list.Element

	endHashCache      []common.Bytes
	blockRequestCache []common.Bytes
}

func NewRequestManager(syncMgr *SyncManager) *RequestManager {
	rm := &RequestManager{
		C: make(chan *core.Block, RequestOutputChannelSize),

		ticker: time.NewTicker(1 * time.Second),
		quota:  RequestQuotaPerSecond,
		// workBell: make(chan struct{}),
		work: make(chan *PendingBlock, WorkerPoolSize),

		wg: &sync.WaitGroup{},

		syncMgr:    syncMgr,
		chain:      syncMgr.chain,
		dispatcher: syncMgr.dispatcher,

		pendingBlocks:         list.New(),
		pendingBlocksByHash:   make(map[string]*list.Element),
		pendingBlocksByParent: make(map[string][]*list.Element),
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
	for curr := rm.pendingBlocks.Front(); rm.quota != 0 && curr != nil; curr = curr.Next() {
		pendingBlock := curr.Value.(*PendingBlock)
		if pendingBlock.status == RequestToSendInvReq ||
			(pendingBlock.status == RequestWaitingInvResp && pendingBlock.HasTimedOut()) {
			tip := rm.syncMgr.consensus.GetTip()
			req := dispatcher.InventoryRequest{ChannelID: common.ChannelIDBlock, Start: tip.Hash.String()}
			rm.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
				"startHash": req.Start,
				"endHash":   req.End,
			}).Debug("Sending inventory request")
			rm.syncMgr.dispatcher.GetInventory([]string{}, req)
			pendingBlock.status = RequestWaitingInvResp
			rm.quota--
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
			pendingBlock.status = RequestWaitingDataResp
			rm.quota--
			continue
		}
	}
}

func (rm *RequestManager) AddHash(x common.Bytes, peerIDs []string) {
	rm.processHash(x, nil, peerIDs)
}

func (rm *RequestManager) AddBlock(b *core.Block) {
	rm.processHash(b.Hash, b, []string{})
}

func (rm *RequestManager) processHash(x common.Bytes, block *core.Block, peerIDs []string) (isAdded bool) {
	if _, err := rm.chain.FindBlock(x); err == nil {
		return true
	}

	var pendingBlockEl *list.Element
	var pendingBlock *PendingBlock
	pendingBlockEl, ok := rm.pendingBlocksByHash[x.String()]
	if !ok {
		pendingBlock = NewPendingBlock(x, peerIDs)
		pendingBlockEl = rm.pendingBlocks.PushBack(pendingBlock)
		rm.pendingBlocksByHash[x.String()] = pendingBlockEl
	} else {
		// Add peerIDs to pendingBlock.peers
		pendingBlock = pendingBlockEl.Value.(*PendingBlock)
		if len(peerIDs) > 0 && pendingBlock.status == RequestWaitingInvResp {
			pendingBlock.status = RequestToSendDataReq
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

	if pendingBlock.block == nil {
		if block == nil {
			if len(pendingBlock.peers) == 0 {
				if pendingBlock.status != RequestWaitingInvResp {
					pendingBlock.status = RequestToSendInvReq
				}
				return false
			}

			if pendingBlock.status != RequestWaitingDataResp {
				pendingBlock.status = RequestToSendDataReq
			}
			return false
		} else {
			pendingBlock.block = block
		}
	}
	block = pendingBlock.block
	parent := block.ParentHash
	if !rm.processHash(parent, nil, []string{}) {
		pendingBlock.status = RequestWaitingParent
		byParents, ok := rm.pendingBlocksByParent[parent.String()]
		if !ok {
			byParents = []*list.Element{}
		}
		found := false
		for _, child := range byParents {
			if 0 == bytes.Compare(child.Value.(*PendingBlock).hash, pendingBlock.hash) {
				found = true
				break
			}
		}
		if !found {
			byParents = append(byParents, pendingBlockEl)
		}
		rm.pendingBlocksByParent[parent.String()] = byParents
		return false
	}

	rm.dumpReadyBlocks(pendingBlockEl)
	return true
}

func (rm *RequestManager) dumpReadyBlocks(x *list.Element) {
	queue := []*list.Element{x}
	for len(queue) > 0 {
		pendingBlockEl := queue[0]
		pendingBlock := pendingBlockEl.Value.(*PendingBlock)
		hash := pendingBlock.hash
		queue = queue[1:]
		if pendingBlock.block != nil {
			if children, ok := rm.pendingBlocksByParent[hash.String()]; ok {
				queue = append(queue, children...)
			}
		}

		rm.chain.AddBlock(pendingBlock.block)
		delete(rm.pendingBlocksByHash, hash.String())
		rm.pendingBlocks.Remove(pendingBlockEl)
		rm.C <- pendingBlock.block
	}
}
