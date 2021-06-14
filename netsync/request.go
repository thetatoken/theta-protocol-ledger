package netsync

import (
	"container/heap"
	"container/list"
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/dispatcher"
	rp "github.com/thetatoken/theta/report"

	log "github.com/sirupsen/logrus"
)

const DumpBlockCacheLimit = 32
const RequestTimeout = 10 * time.Second
const Expiration = 300 * time.Second
const MinInventoryRequestInterval = 6 * time.Second
const MaxInventoryRequestInterval = 6 * time.Second

const FastsyncRequestQuota = 8 // Max number of outstanding block requests
const GossipRequestQuotaPerSecond = 10
const MaxNumPeersToSendRequests = 4
const RefreshCounterLimit = 4
const MaxBlocksPerRequest = 4
const MaxPeerActiveScore = 16

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
	blockNotify          chan *core.ExtendedBlock
	tip                  atomic.Value

	mu                      *sync.RWMutex
	pendingBlocks           *list.List
	pendingBlocksByHash     map[string]*list.Element
	pendingBlocksWithHeader *HeaderHeap
	gossipQuota             uint
	fastsyncQuota           uint
	ifDownloadByHash        bool
	ifDownloadByHeader      bool

	dumpBlockCache *lru.Cache

	endHashCache      []common.Bytes
	blockRequestCache []common.Bytes

	activePeers    map[string]int
	refreshCounter int
	aplock         *sync.RWMutex

	reporter *rp.Reporter
}

func NewRequestManager(syncMgr *SyncManager, reporter *rp.Reporter) *RequestManager {
	dumpBlockCache, err := lru.New(DumpBlockCacheLimit)
	if err != nil {
		log.Panic(err)
	}

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
		pendingBlocksWithHeader: &HeaderHeap{},
		ifDownloadByHash:        viper.GetBool(common.CfgSyncDownloadByHash),
		ifDownloadByHeader:      viper.GetBool(common.CfgSyncDownloadByHeader),

		blockNotify:    make(chan *core.ExtendedBlock, 1),
		dumpBlockCache: dumpBlockCache,

		activePeers:    make(map[string]int),
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

	rm.wg.Add(1)
	go rm.mainLoop()

	rm.wg.Add(1)
	go rm.passReadyBlocks()
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

	for pid, score := range rm.activePeers {
		if pid == activePeerID {
			if score < MaxPeerActiveScore {
				rm.activePeers[pid] = MaxPeerActiveScore
			}
			rm.logger.Debugf("Active peer boosted: %v", activePeerID)
			return
		}
	}

	if len(rm.activePeers) >= MaxNumPeersToSendRequests {
		minScore := MaxPeerActiveScore
		minPID := ""
		for pid, score := range rm.activePeers {
			if score <= minScore {
				minScore = score
				minPID = pid
			}
		}
		delete(rm.activePeers, minPID)
	}

	rm.activePeers[activePeerID] = MaxPeerActiveScore
	rm.logger.Debugf("Active peer added: %v", activePeerID)
}

func (rm *RequestManager) buildInventoryRequest() dispatcher.InventoryRequest {
	tip, ok := rm.tip.Load().(*core.ExtendedBlock)
	if !ok || tip == nil {
		tip = rm.syncMgr.consensus.GetTip(true)
	}
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
			// Exclude orphan blocks and pending blocks
			if b.Status != core.BlockStatusPending && b.Status != core.BlockStatusInvalid {
				starts = append(starts, b.Hash().Hex())
			}
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

	rm.gossipQuota = GossipRequestQuotaPerSecond
	rm.fastsyncQuota = FastsyncRequestQuota

	hasUndownloadedBlocks := rm.pendingBlocks.Len() > 0 || len(rm.pendingBlocksByHash) > 0 || rm.pendingBlocksWithHeader.Len() > 0

	minIntervalPassed := time.Since(rm.lastInventoryRequest) >= MinInventoryRequestInterval
	maxIntervalPassed := time.Since(rm.lastInventoryRequest) >= MaxInventoryRequestInterval

	if maxIntervalPassed || (hasUndownloadedBlocks && minIntervalPassed) {
		if hasUndownloadedBlocks && rm.pendingBlocks.Len() > 1 {
			fastSyncHeight := uint64(0)
			if fastSyncTip, ok := rm.tip.Load().(*core.ExtendedBlock); ok {
				fastSyncHeight = fastSyncTip.Height
			}
			rm.logger.WithFields(log.Fields{
				"pending block hashes": rm.pendingBlocks.Len(),
				"current chain tip":    rm.syncMgr.consensus.GetTip(true).Hash().Hex(),
				"fast sync tip":        fastSyncHeight,
			}).Info("Sync progress")
		}

		rm.lastInventoryRequest = time.Now()
		req := rm.buildInventoryRequest()
		rm.getInventory(req)
	}
	if rm.ifDownloadByHeader {
		rm.downloadBlockFromHeader()
	}
	if rm.ifDownloadByHash {
		rm.downloadBlockFromHash()
	}

	// Remove downloaded blocks from header queue
	// newQ := []*PendingBlock{}
	newQ := &HeaderHeap{}
	for _, header := range *rm.pendingBlocksWithHeader {
		if _, ok := rm.pendingBlocksByHash[header.hash.Hex()]; ok {
			heap.Push(newQ, header)
		}
	}
	rm.pendingBlocksWithHeader = newQ
}

//compatible with older version, download block from hash
func (rm *RequestManager) downloadBlockFromHash() {
	//loop over downloaded hash
	var curr *list.Element
	elToRemove := []*list.Element{}
	for curr = rm.pendingBlocks.Front(); (rm.gossipQuota > 0 || rm.fastsyncQuota > 0) && curr != nil; curr = curr.Next() {
		pendingBlock := curr.Value.(*PendingBlock)
		if pendingBlock.HasExpired() || pendingBlock.HasTimedOut() {
			elToRemove = append(elToRemove, curr)
			continue
		}
		if pendingBlock.block != nil {
			continue
		}
		if len(pendingBlock.peers) == 0 {
			continue
		}
		if pendingBlock.fromGossip && rm.gossipQuota <= 0 {
			continue
		}
		if !pendingBlock.fromGossip && rm.fastsyncQuota <= 0 {
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
		if pendingBlock.status == RequestToSendDataReq ||
			(!rm.ifDownloadByHeader && pendingBlock.status == RequestToSendBodyReq) {
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
				rm.gossipQuota--
			} else {
				rm.fastsyncQuota--
			}

			continue
		}
	}
	for _, el := range elToRemove {
		pendingBlock := el.Value.(*PendingBlock)
		hash := pendingBlock.hash.Hex()
		height := uint64(0)
		if pendingBlock.header != nil {
			height = pendingBlock.header.Height
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
	addBack := HeaderHeap{}
	elToRemove := []*list.Element{}
	peerMap := make(map[string][]string)
	var blockBuffer []string
	var ok bool
	for rm.pendingBlocksWithHeader.Len() > 0 && rm.fastsyncQuota > 0 {
		pendingBlock := heap.Pop(rm.pendingBlocksWithHeader).(*PendingBlock)

		// Remove expired header from queue
		if pendingBlock.HasExpired() {
			if el, ok := rm.pendingBlocksByHash[pendingBlock.hash.String()]; ok {
				elToRemove = append(elToRemove, el)
			}
			continue
		}
		// Remove header for downloaded blocks from queue
		isDownloaded := false
		if rm.dumpBlockCache.Contains(pendingBlock.hash) {
			isDownloaded = true
		}
		if !isDownloaded {
			if _, err := rm.chain.FindBlock(pendingBlock.hash); err == nil {
				isDownloaded = true
			}
		}
		if isDownloaded {
			if el, ok := rm.pendingBlocksByHash[pendingBlock.hash.String()]; ok {
				elToRemove = append(elToRemove, el)
			}
			continue
		}

		// Otherwise the header should be added back to queue
		addBack = append(addBack, pendingBlock)
		if len(pendingBlock.peers) == 0 {
			rm.logger.WithFields(log.Fields{
				"block": pendingBlock.hash.String(),
			}).Debug("Skip block with no peer")
			continue
		}
		if pendingBlock.status == RequestWaitingBodyResp && !pendingBlock.HasTimedOut() {
			rm.fastsyncQuota--
			continue
		}
		if pendingBlock.status == RequestToSendBodyReq ||
			(pendingBlock.status == RequestWaitingBodyResp && pendingBlock.HasTimedOut()) {

			peersWithBlock := util.Shuffle(pendingBlock.peers)
			var randomPeerID string
			for i := 0; i < len(peersWithBlock); i++ {
				if rm.dispatcher.PeerExists(peersWithBlock[i]) { // the peer may have been purged
					randomPeerID = peersWithBlock[i]
					break
				}

				rm.logger.WithFields(log.Fields{
					"pendingBlock": pendingBlock.hash.String(),
					"peer":         peersWithBlock[i],
				}).Debug("Skipped peer that may have been purged")

			}
			if len(randomPeerID) == 0 {
				rm.logger.WithFields(log.Fields{
					"pendingBlock": pendingBlock.hash.String(),
				}).Debug("All peers skipped")
				continue
			}

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
			rm.fastsyncQuota--
		}
	}
	// send block requests for every peer in map
	for k, v := range peerMap {
		if len(v) > 0 {
			rm.sendBlocksRequest(k, v)
		}
	}
	for _, header := range addBack {
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

	for pid := range rm.activePeers {
		if !rm.dispatcher.PeerExists(pid) { // the peer may have been purged
			rm.logger.Debugf("Removing disconnected peer from active list: %v", pid)
			delete(rm.activePeers, pid)
		} else {
			rm.activePeers[pid]--
		}

	}
	if rm.refreshCounter >= RefreshCounterLimit {
		rm.refreshCounter = 0

		rm.logger.Debugf("Reset refreshCounter")
	}
	if len(rm.activePeers) != 0 {
		peersToRequest = []string{}
		for pid, score := range rm.activePeers {
			if score > 0 {
				peersToRequest = append(peersToRequest, pid)
			} else {
				rm.logger.WithFields(log.Fields{
					"peer":  pid,
					"score": score,
				}).Debugf("Skipping low score peer from active list")
			}
		}
		rm.logger.Debugf("Reuse activePeers: %v", peersToRequest)
	}
	rm.aplock.Unlock()

	targetSize := MaxNumPeersToSendRequests
	if rm.refreshCounter == 0 {
		// Query extra random peers
		targetSize += 2
	}
	if len(peersToRequest) < targetSize { // resample
		allPeers := rm.syncMgr.dispatcher.Peers(true) // skip edge nodes
		samples := util.Sample(allPeers, targetSize)
		for _, sample := range samples {
			duplicate := false
			for _, pid := range peersToRequest {
				if pid == sample {
					duplicate = true
					break
				}
			}
			if !duplicate {
				peersToRequest = append(peersToRequest, sample)
			}

			if len(peersToRequest) >= targetSize {
				break
			}
		}
		rm.logger.Debugf("Resampled peers to send requests: %v", peersToRequest)
	}

	rm.logger.WithFields(log.Fields{
		"channelID": req.ChannelID,
		"starts":    req.Starts,
		"end":       req.End,
		"peers":     peersToRequest,
	}).Debug("Sending inventory request")

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
		rm.logger.WithFields(log.Fields{
			"hash": header.Hash().String(),
		}).Debug("Skipping header: this block is already downloaded")
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
		for _, idToAdd := range peerIDs {
			found := false
			for _, id := range pendingBlock.peers {
				if id == idToAdd {
					found = true
					break
				}
			}
			if !found {
				pendingBlock.peers = append(pendingBlock.peers, idToAdd)
			}
		}
	}
}

// AddBlock process an incoming block.
func (rm *RequestManager) AddBlock(block *core.Block) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	eb, err := rm.chain.AddBlock(block)
	if err != nil {
		log.Debugf("failed to add block, err=%v", err)
		return
	}

	hash := block.Hash().String()

	if pendingBlockEl, ok := rm.pendingBlocksByHash[hash]; ok {
		rm.pendingBlocks.Remove(pendingBlockEl)
		delete(rm.pendingBlocksByHash, hash)
	}

	select {
	case rm.blockNotify <- eb:
	default:
	}
}

func (rm *RequestManager) passReadyBlocks() {
	defer rm.wg.Done()

	timer := time.NewTicker(time.Second)
	defer timer.Stop()

	for {
		lfb := rm.syncMgr.consensus.GetLastFinalizedBlock()
		height := lfb.Height + 1
		parents := []*core.ExtendedBlock{lfb}

		for {
			blocks := rm.chain.FindBlocksByHeight(height)

			if len(blocks) == 0 {
				break
			}

			for _, block := range blocks {
				if rm.dumpBlockCache.Contains(block.Hash()) {
					continue
				}

				// Check if block's parent has already been added to chain. If not, skip block
				found := false
				for _, parent := range parents {
					if parent.Hash() == block.Parent && parent.Status.IsValid() {
						found = true
						break
					}
				}
				if !found {
					continue
				}

				rm.dumpBlockCache.Add(block.Hash(), struct{}{})
				if block.Status.IsPending() {
					rm.syncMgr.PassdownMessage(block.Block)
					rm.tip.Store(block)
				}
			}

			height++
			parents = blocks
		}

		select {
		case <-rm.ctx.Done():
			return
		case <-rm.blockNotify:
		case <-timer.C:
		}
	}

}
