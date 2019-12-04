package netsync

import (
	"container/heap"
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
	rp "github.com/thetatoken/theta/report"

	log "github.com/sirupsen/logrus"
)

const RequestTimeout = 5 * time.Second
const Expiration = 300 * time.Second
const MinInventoryRequestInterval = 3 * time.Second
const MaxInventoryRequestInterval = 3 * time.Second
const FastsyncRequestQuotaPerSecond = 50
const GossipRequestQuotaPerSecond = 50
const MaxNumPeersToSendRequests = 4
const RefreshCounterLimit = 4
const MaxBlocksPerRequest = 8

type RequestState uint8

const (
	RequestToSendDataReq = iota
	RequestWaitingDataResp
	RequestToSendBodyReq
	RequestWaitingBodyResp
)

type PendingBlock struct {
	hash       common.Hash
	block      *core.Block
	header     *core.BlockHeader
	peers      []string
	lastUpdate time.Time
	createdAt  time.Time
	status     RequestState
	fromGossip bool
}

func NewPendingBlock(x common.Hash, peerIds []string, fromGossip bool) *PendingBlock {
	return &PendingBlock{
		hash:       x,
		lastUpdate: time.Now(),
		createdAt:  time.Now(),
		peers:      peerIds,
		status:     RequestToSendDataReq,
		fromGossip: fromGossip,
	}
}

func (pb *PendingBlock) HasTimedOut() bool {
	return time.Since(pb.lastUpdate) > RequestTimeout
}

func (pb *PendingBlock) HasExpired() bool {
	return time.Since(pb.createdAt) > Expiration
}

func (pb *PendingBlock) UpdateTimestamp() {
	pb.lastUpdate = time.Now()
}

type HeaderHeap []*PendingBlock

func (h HeaderHeap) Len() int { return len(h) }
func (h HeaderHeap) Less(i, j int) bool {
	if h[i].header != nil && h[j].header != nil {
		return h[i].header.Height < h[j].header.Height
	}
	return i < j
}

func (h HeaderHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *HeaderHeap) Push(x interface{}) {
	*h = append(*h, x.(*PendingBlock))
}

func (h *HeaderHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	return x
}

type RequestManager struct {
	logger *log.Entry

	ticker *time.Ticker

	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	syncMgr    *SyncManager
	chain      *blockchain.Chain
	dispatcher *dispatcher.Dispatcher

	lastInventoryRequest time.Time

	mu                      *sync.RWMutex
	pendingBlocks           *list.List
	pendingBlocksByHash     map[string]*list.Element
	pendingBlocksByParent   map[string][]*core.Block
	pendingBlocksWithHeader *HeaderHeap

	endHashCache      []common.Bytes
	blockRequestCache []common.Bytes

	activePeers    []string
	refreshCounter int
	aplock         *sync.RWMutex

	reporter *rp.Reporter
}

func NewRequestManager(syncMgr *SyncManager, reporter *rp.Reporter) *RequestManager {
	rm := &RequestManager{
		ticker: time.NewTicker(1 * time.Second),

		wg: &sync.WaitGroup{},

		lastInventoryRequest: time.Unix(0, 0),

		syncMgr:    syncMgr,
		chain:      syncMgr.chain,
		dispatcher: syncMgr.dispatcher,

		mu:                      &sync.RWMutex{},
		pendingBlocks:           list.New(),
		pendingBlocksByHash:     make(map[string]*list.Element),
		pendingBlocksByParent:   make(map[string][]*core.Block),
		pendingBlocksWithHeader: &HeaderHeap{},

		activePeers:    []string{},
		refreshCounter: 0,
		aplock:         &sync.RWMutex{},

		reporter: reporter,
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
			rm.tryToDownload()
		}
	}
}

func (rm *RequestManager) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	rm.ctx = c
	rm.cancel = cancel

	rm.resumePendingBlocks()

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

func (rm *RequestManager) AddActivePeer(activePeerID string) {
	rm.aplock.Lock()
	defer rm.aplock.Unlock()

	if len(rm.activePeers) >= MaxNumPeersToSendRequests {
		return
	}

	for _, pid := range rm.activePeers {
		if pid == activePeerID {
			return
		}
	}

	rm.activePeers = append(rm.activePeers, activePeerID)
	rm.logger.Debugf("Active peer added: %v", activePeerID)
}

func (rm *RequestManager) buildInventoryRequest() dispatcher.InventoryRequest {
	tip := rm.syncMgr.consensus.GetTip(true)
	lfb := rm.syncMgr.consensus.GetLastFinalizedBlock()

	// Build expontially backoff starting hashes:
	// https://en.bitcoin.it/wiki/Protocol_documentation#getblocks
	starts := []string{}
	step := 1

	// Start at the top of the chain and work backwards.
	for index := tip.Height; index > lfb.Height; index -= uint64(step) {
		// Push top 10 indexes first, then back off exponentially.
		if tip.Height-index >= 10 {
			step *= 2
		}
		// Check overflow
		if uint64(step) > index || step <= 0 {
			break
		}

		blocks := rm.syncMgr.chain.FindBlocksByHeight(index)
		for _, b := range blocks {
			starts = append(starts, b.Hash().Hex())
		}
	}

	//  Push last finalized block.
	starts = append(starts, lfb.Hash().Hex())

	return dispatcher.InventoryRequest{
		ChannelID: common.ChannelIDBlock,
		Starts:    starts,
	}
}

func (rm *RequestManager) tryToDownload() {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	hasUndownloadedBlocks := rm.pendingBlocks.Len() > 0 || len(rm.pendingBlocksByHash) > 0 || len(rm.pendingBlocksByParent) > 0 || rm.pendingBlocksWithHeader.Len() > 0
	rm.reporter.SetInSync(!hasUndownloadedBlocks)

	minIntervalPassed := time.Since(rm.lastInventoryRequest) >= MinInventoryRequestInterval
	maxIntervalPassed := time.Since(rm.lastInventoryRequest) >= MaxInventoryRequestInterval

	if maxIntervalPassed || (hasUndownloadedBlocks && minIntervalPassed) {
		if hasUndownloadedBlocks && rm.pendingBlocks.Len() > 1 {
			rm.logger.WithFields(log.Fields{
				"pending block hashes": rm.pendingBlocks.Len() - len(rm.pendingBlocksByParent),
				"orphan blocks":        len(rm.pendingBlocksByParent),
				"current chain tip":    rm.syncMgr.consensus.GetTip(true).Hash().Hex(),
			}).Info("Sync progress")
		}

		rm.lastInventoryRequest = time.Now()
		req := rm.buildInventoryRequest()

		rm.logger.WithFields(log.Fields{
			"channelID": req.ChannelID,
			"starts":    req.Starts,
			"end":       req.End,
		}).Debug("Sending inventory request")

		rm.getInventory(req)
	}
	if viper.GetBool(common.CfgSyncDownloadByHeader) {
		rm.downloadBlockFromHeader()
	}
	if viper.GetBool(common.CfgSyncDownloadByHash) {
		rm.downloadBlockFromHash()
	}
}

//compatible with older version, download block from hash
func (rm *RequestManager) downloadBlockFromHash() {
	gossipQuota := GossipRequestQuotaPerSecond
	fastsyncQuota := FastsyncRequestQuotaPerSecond

	//loop over downloaded hash
	var curr *list.Element
	elToRemove := []*list.Element{}
	for curr = rm.pendingBlocks.Front(); (gossipQuota > 0 || fastsyncQuota > 0) && curr != nil; curr = curr.Next() {
		pendingBlock := curr.Value.(*PendingBlock)
		if pendingBlock.HasExpired() || pendingBlock.HasTimedOut() {
			elToRemove = append(elToRemove, curr)
			continue
		}
		if pendingBlock.header != nil || pendingBlock.block != nil {
			continue
		}
		if len(pendingBlock.peers) == 0 {
			continue
		}
		if pendingBlock.fromGossip && gossipQuota <= 0 {
			continue
		}
		if !pendingBlock.fromGossip && fastsyncQuota <= 0 {
			continue
		}
		// if pendingBlock.status == RequestWaitingDataResp {
		// 	if pendingBlock.fromGossip {
		// 		gossipQuota--
		// 	} else {
		// 		fastsyncQuota--
		// 	}
		// 	continue
		// }
		if pendingBlock.status == RequestToSendDataReq {
			randomPeerID := pendingBlock.peers[rand.Intn(len(pendingBlock.peers))]
			request := dispatcher.DataRequest{
				ChannelID: common.ChannelIDBlock,
				Entries:   []string{pendingBlock.hash.String()},
			}
			rm.logger.WithFields(log.Fields{
				"channelID":       request.ChannelID,
				"request.Entries": request.Entries,
				"peer":            randomPeerID,
			}).Debug("Sending data request from hash")
			rm.syncMgr.dispatcher.GetData([]string{randomPeerID}, request)
			pendingBlock.UpdateTimestamp()
			pendingBlock.status = RequestWaitingDataResp

			if pendingBlock.fromGossip {
				gossipQuota--
			} else {
				fastsyncQuota--
			}

			continue
		}
	}
	for _, el := range elToRemove {
		pendingBlock := el.Value.(*PendingBlock)
		hash := pendingBlock.hash.Hex()
		height := uint64(0)
		if pendingBlock.block != nil {
			height = pendingBlock.block.Height
		}
		rm.logger.WithFields(log.Fields{
			"block":        hash,
			"block.Height": height,
		}).Debug("Removing outdated block")
		rm.removeEl(el)
	}
}

//download block from header
func (rm *RequestManager) downloadBlockFromHeader() {
	quota := GossipRequestQuotaPerSecond

	backup := HeaderHeap{}
	elToRemove := []*list.Element{}
	peerMap := make(map[string][]string)
	var blockBuffer []string
	var ok bool
	for rm.pendingBlocksWithHeader.Len() > 0 && quota > 0 {
		pendingBlock := heap.Pop(rm.pendingBlocksWithHeader).(*PendingBlock)
		if pendingBlock.HasExpired() {
			if el, ok := rm.pendingBlocksByHash[pendingBlock.hash.String()]; ok {
				elToRemove = append(elToRemove, el)
			}
			continue
		}
		if _, ok := rm.pendingBlocksByHash[pendingBlock.hash.String()]; !ok || pendingBlock.block != nil {
			continue
		}
		if len(pendingBlock.peers) == 0 {
			continue
		}
		if pendingBlock.status == RequestToSendBodyReq ||
			(pendingBlock.status == RequestWaitingBodyResp && pendingBlock.HasTimedOut()) {
			randomPeerID := pendingBlock.peers[rand.Intn(len(pendingBlock.peers))]
			if blockBuffer, ok = peerMap[randomPeerID]; !ok {
				blockBuffer = []string{}
			}
			blockBuffer := append(blockBuffer, pendingBlock.hash.String())
			if len(blockBuffer) == MaxBlocksPerRequest {
				rm.sendBlocksRequest(randomPeerID, blockBuffer)
				blockBuffer = []string{}
			}
			peerMap[randomPeerID] = blockBuffer
			pendingBlock.UpdateTimestamp()
			pendingBlock.status = RequestWaitingBodyResp
			quota--
		}
		backup = append(backup, pendingBlock)
	}
	// send block requests for every peer in map
	for k, v := range peerMap {
		if len(v) > 0 {
			rm.sendBlocksRequest(k, v)
		}
	}
	for _, header := range backup {
		heap.Push(rm.pendingBlocksWithHeader, header)
	}
	for _, el := range elToRemove {
		pendingBlock := el.Value.(*PendingBlock)
		hash := pendingBlock.hash.Hex()
		height := uint64(0)
		if pendingBlock.block != nil {
			height = pendingBlock.block.Height
		}
		rm.logger.WithFields(log.Fields{
			"block":        hash,
			"block.Height": height,
		}).Debug("Removing outdated block")
		rm.removeEl(el)
	}
}

func (rm *RequestManager) getInventory(req dispatcher.InventoryRequest) {
	var peersToRequest []string

	rm.logger.Debugf("refreshCounter: %v", rm.refreshCounter)

	rm.aplock.Lock()
	rm.refreshCounter++
	if rm.refreshCounter >= RefreshCounterLimit {
		rm.activePeers = []string{}
		rm.refreshCounter = 0

		rm.logger.Debugf("Reset refreshCounter")
	}
	if len(rm.activePeers) != 0 {
		peersToRequest = make([]string, len(rm.activePeers))
		copy(peersToRequest, rm.activePeers)

		rm.logger.Debugf("Reuse activePeers: %v", peersToRequest)
	}
	rm.aplock.Unlock()

	if len(peersToRequest) == 0 { // resample
		allPeers := rm.syncMgr.dispatcher.Peers()
		peersToRequest = util.Sample(allPeers, MaxNumPeersToSendRequests)

		rm.logger.Debugf("Resampled peers to send requests: %v", peersToRequest)
	}

	rm.syncMgr.dispatcher.GetInventory(peersToRequest, req)
}

func (rm *RequestManager) sendBlocksRequest(peerID string, entries []string) {
	request := dispatcher.DataRequest{
		ChannelID: common.ChannelIDBlock,
		Entries:   entries,
	}
	rm.logger.WithFields(log.Fields{
		"channelID":       request.ChannelID,
		"request.Entries": request.Entries,
		"peer":            peerID,
	}).Debug("Sending data request from header")
	rm.syncMgr.dispatcher.GetData([]string{peerID}, request)
}

func (rm *RequestManager) removeEl(el *list.Element) {
	pendingBlock := el.Value.(*PendingBlock)
	hash := pendingBlock.hash.Hex()

	delete(rm.pendingBlocksByHash, hash)

	if pendingBlock.block != nil {
		parent := pendingBlock.block.Parent.Hex()
		if blocks, ok := rm.pendingBlocksByParent[parent]; ok {
			found := -1
			for idx, block := range blocks {
				if block.Hash() == pendingBlock.block.Hash() {
					found = idx
					break
				}
			}
			if found != -1 {
				blocks = append(blocks[:found], blocks[found+1:]...)
				rm.pendingBlocksByParent[parent] = blocks
			}
			if len(rm.pendingBlocksByParent[parent]) == 0 {
				delete(rm.pendingBlocksByParent, parent)
			}
		}
	}
	rm.pendingBlocks.Remove(el)
}

func (rm *RequestManager) AddHash(x common.Hash, peerIDs []string, fromGossip bool) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.addHash(x, peerIDs, fromGossip)
}

func (rm *RequestManager) addHash(x common.Hash, peerIDs []string, fromGossip bool) {
	if _, err := rm.chain.FindBlock(x); err == nil {
		return
	}

	var pendingBlockEl *list.Element
	var pendingBlock *PendingBlock
	pendingBlockEl, ok := rm.pendingBlocksByHash[x.String()]
	if !ok {
		pendingBlock = NewPendingBlock(x, peerIDs, fromGossip)
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

// shouldDumpBlock checks if a block and its decendant is descendant of genesis
func (rm *RequestManager) shouldDumpBlock(block *core.Block) bool {
	currHash := block.Parent
	for {
		currBlock, err := rm.chain.FindBlock(currHash)
		if err != nil {
			return false
		}
		// If a block has status other than pending, it has been processed by consensus engine hence
		// must be descendant of genesis.
		if !currBlock.Status.IsPending() {
			return true
		}
		currHash = currBlock.Parent
	}
}

func (rm *RequestManager) IsGossipBlock(hash common.Hash) bool {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	var pendingBlockEl *list.Element
	pendingBlockEl, ok := rm.pendingBlocksByHash[hash.String()]
	if !ok {
		return true // be more conservative here
	}

	pendingBlock := pendingBlockEl.Value.(*PendingBlock)
	return pendingBlock.fromGossip
}

func (rm *RequestManager) AddHeader(header *core.BlockHeader, peerIDs []string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if _, err := rm.chain.FindBlock(header.Hash()); err == nil {
		rm.logger.Debug("this block is already downloaded")
		return
	}
	if _, ok := rm.pendingBlocksByHash[header.Hash().String()]; !ok {
		rm.addHash(header.Hash(), peerIDs, true)
	}
	if pendingBlockEl, ok := rm.pendingBlocksByHash[header.Hash().String()]; ok {
		pendingBlock := pendingBlockEl.Value.(*PendingBlock)
		if pendingBlock.header == nil {
			pendingBlock.header = header
			pendingBlock.status = RequestToSendBodyReq
			heap.Push(rm.pendingBlocksWithHeader, pendingBlock)
		}
	}
}

func (rm *RequestManager) AddBlock(block *core.Block) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	_, err := rm.chain.AddBlock(block)
	if err != nil {
		rm.logger.WithFields(log.Fields{
			"err": err.Error(),
		}).Error("Failed to add block")
	}

	// TODO: should not need in-memory index anymore.
	if _, ok := rm.pendingBlocksByHash[block.Hash().String()]; !ok {
		rm.addHash(block.Hash(), []string{}, false)
	}
	if pendingBlockEl, ok := rm.pendingBlocksByHash[block.Hash().String()]; ok {
		pendingBlock := pendingBlockEl.Value.(*PendingBlock)
		//check txHash with header
		if pendingBlock.header != nil {
			if pendingBlock.header.Hash() != pendingBlock.hash {
				rm.logger.WithFields(log.Fields{
					"pending block hash":          pendingBlock.hash.String(),
					"pending block header hash":   pendingBlock.header.Hash().String(),
					"pending block header Height": pendingBlock.header.Height,
				}).Info("pendingblock.hash doesn't match with pendingblock.header.hash")
				rm.removeEl(pendingBlockEl)
				return
			}
			downloadBlockTxHash := core.CalculateRootHash(block.Txs)
			if downloadBlockTxHash != pendingBlock.header.TxHash {
				rm.logger.WithFields(log.Fields{
					"pending block hash":                 pendingBlock.hash.String(),
					"pending block header txHash":        pendingBlock.header.TxHash.String(),
					"pending block header Height":        pendingBlock.header.Height,
					"download block hash":                block.Hash().String(),
					"downloaded block Calculated txHash": downloadBlockTxHash.String(),
					"downloaded block Height":            block.Height,
				}).Info("TxHash doesn't match with header ")
				rm.removeEl(pendingBlockEl)
				return
			}
		}
		pendingBlock.block = block
	}

	if rm.shouldDumpBlock(block) {
		rm.dumpReadyBlocks(block)
		return
	}

	// TODO: remove this. We don't need in-memory index anymore.
	// byParents, ok := rm.pendingBlocksByParent[parent.String()]
	// if !ok {
	// 	byParents = []*core.Block{}
	// }
	// found := false
	// for _, child := range byParents {
	// 	if child.Hash() == block.Hash() {
	// 		found = true
	// 		break
	// 	}
	// }
	// if !found {
	// 	byParents = append(byParents, block)
	// }
	// rm.pendingBlocksByParent[parent.String()] = byParents
}

// resumePendingBlocks is called during process start to resume blocks that are already downloaded
// but are not yet processed by consensus engine.
func (rm *RequestManager) resumePendingBlocks() {
	lfb := rm.syncMgr.consensus.GetLastFinalizedBlock()
	queue := []*core.ExtendedBlock{lfb}
	for len(queue) > 0 {
		block := queue[0]
		queue = queue[1:]
		if block.Status.IsPending() {
			rm.AddBlock(block.Block)
		}
		for _, hash := range block.Children {
			child, err := rm.chain.FindBlock(hash)
			if err != nil {
				logger.Panic(err)
			}
			queue = append(queue, child)
		}
	}
}

func (rm *RequestManager) dumpReadyBlocks(block *core.Block) {
	queue := []*core.Block{block}
	for len(queue) > 0 {
		block := queue[0]
		hash := block.Hash().String()
		queue = queue[1:]

		// Add child blocks stored in the memory
		children, ok := rm.pendingBlocksByParent[hash]
		if ok {
			queue = append(queue, children...)
			delete(rm.pendingBlocksByParent, hash)
		}

		if pendingBlockEl, ok := rm.pendingBlocksByHash[hash]; ok {
			rm.pendingBlocks.Remove(pendingBlockEl)
			delete(rm.pendingBlocksByHash, hash)
		}

		// Add child blocks stored in the disk
		height := block.Height
		for _, child := range rm.chain.FindBlocksByHeight(height + 1) {
			if child.Parent.String() != hash {
				continue
			}

			duplicated := false
			for _, ch := range children {
				if ch.Hash() == child.Hash() {
					duplicated = true
					break
				}
			}

			if !duplicated {
				queue = append(queue, child.Block)
			}
		}

		rm.syncMgr.PassdownMessage(block)

		rm.syncMgr.dispatcher.SendInventory([]string{}, dispatcher.InventoryResponse{
			ChannelID: common.ChannelIDBlock,
			Entries:   []string{block.Hash().Hex()},
		})
	}
}
