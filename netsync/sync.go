package netsync

import (
	"context"
	"reflect"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/dispatcher"
	"github.com/thetatoken/theta/p2p"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/p2pl"
	rp "github.com/thetatoken/theta/report"
	"github.com/thetatoken/theta/rlp"
)

const voteCacheLimit = 512

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "netsync"})

type MessageConsumer interface {
	AddMessage(interface{})
}

type Headers struct {
	HeaderArray []*core.BlockHeader
}

type Blocks struct {
	BlockArray []*core.Block
}

var _ p2p.MessageHandler = (*SyncManager)(nil)

// SyncManager is an intermediate layer between consensus engine and p2p network. Its main responsibilities are to manage
// fast blocks sync among peers and buffer orphaned block/CC. Otherwise messages are passed through to consensus engine.
type SyncManager struct {
	chain      *blockchain.Chain
	consensus  core.ConsensusEngine
	consumer   MessageConsumer
	dispatcher *dispatcher.Dispatcher
	requestMgr *RequestManager

	wg       *sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
	stopped  bool
	incoming chan p2ptypes.Message

	whitelist []string

	logger *log.Entry

	voteCache *lru.Cache // Cache for votes
}

func NewSyncManager(chain *blockchain.Chain, cons core.ConsensusEngine, networkOld p2p.Network, network p2pl.Network, disp *dispatcher.Dispatcher, consumer MessageConsumer, reporter *rp.Reporter) *SyncManager {
	voteCache, _ := lru.New(voteCacheLimit)
	sm := &SyncManager{
		chain:      chain,
		consensus:  cons,
		consumer:   consumer,
		dispatcher: disp,
		wg:         &sync.WaitGroup{},
		incoming:   make(chan p2ptypes.Message, viper.GetInt(common.CfgSyncMessageQueueSize)),

		voteCache: voteCache,
	}
	sm.requestMgr = NewRequestManager(sm, reporter)

	if !reflect.ValueOf(networkOld).IsNil() {
		networkOld.RegisterMessageHandler(sm)
	}
	if !reflect.ValueOf(network).IsNil() {
		network.RegisterMessageHandler(sm)
	}

	if viper.GetString(common.CfgSyncInboundResponseWhitelist) != "" {
		sm.whitelist = strings.Split(viper.GetString(common.CfgSyncInboundResponseWhitelist), ",")
	}

	logger := util.GetLoggerForModule("sync")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": sm.consensus.ID()})
	}
	sm.logger = logger

	return sm
}

func (sm *SyncManager) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	sm.ctx = c
	sm.cancel = cancel

	sm.requestMgr.Start(c)

	sm.wg.Add(1)
	go sm.mainLoop()
}

func (sm *SyncManager) Stop() {
	sm.cancel()
}

func (sm *SyncManager) Wait() {
	sm.requestMgr.Wait()
	sm.wg.Wait()
}

func (sm *SyncManager) mainLoop() {
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
		common.ChannelIDProposal,
		common.ChannelIDCC,
		common.ChannelIDVote,
		common.ChannelIDGuardian,
		common.ChannelIDEliteEdgeNodeVote,
		common.ChannelIDAggregatedEliteEdgeNodeVotes,
	}
}

// ParseMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) ParseMessage(peerID string, channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
	}
	data, err := decodeMessage(rawMessageBytes)
	message.Content = data
	return message, err
}

// EncodeMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) EncodeMessage(message interface{}) (common.Bytes, error) {
	return encodeMessage(message)
}

// HandleMessage implements p2p.MessageHandler interface.
func (sm *SyncManager) HandleMessage(msg p2ptypes.Message) (err error) {
	sm.incoming <- msg
	return
}

func (sm *SyncManager) processMessage(message p2ptypes.Message) {
	inboundAllowed := true
	// If whitelist is set, only process message from peers in the whitelist.
	if len(sm.whitelist) > 0 {
		inboundAllowed = false
		for _, peerID := range sm.whitelist {
			if strings.ToLower(peerID) == strings.ToLower(message.PeerID) {
				inboundAllowed = true
				break
			}
		}
	}

	switch content := message.Content.(type) {
	case dispatcher.InventoryRequest:
		sm.handleInvRequest(message.PeerID, &content)
	case dispatcher.InventoryResponse:
		if !inboundAllowed {
			return
		}
		sm.handleInvResponse(message.PeerID, &content)
	case dispatcher.DataRequest:
		sm.handleDataRequest(message.PeerID, &content)
	case dispatcher.DataResponse:
		if !inboundAllowed {
			return
		}
		sm.handleDataResponse(message.PeerID, &content)
	default:
		sm.logger.WithFields(log.Fields{
			"message": message,
		}).Warn("Received unknown message")
	}
}

// PassdownMessage passes message through to the consumer.
func (sm *SyncManager) PassdownMessage(msg interface{}) {
	sm.consumer.AddMessage(msg)
}

// locateStart finds first start hash that exists in local chain.
func (m *SyncManager) locateStart(starts []string) common.Hash {
	var start common.Hash
	for i := 0; i < len(starts); i++ {
		curr := common.HexToHash(starts[i])
		b, err := m.chain.FindBlock(curr)
		if err == nil &&
			b.Status != core.BlockStatusPending &&
			b.Status != core.BlockStatusInvalid {
			start = curr
			break
		}
	}
	return start
}

// Dump blocks from start until end or MaxInventorySize is reached.
func (m *SyncManager) collectBlocks(start common.Hash, end common.Hash) []string {
	ret := []string{}

	lfbHeight := m.consensus.GetLastFinalizedBlock().Height
	q := []common.Hash{start}
	for len(q) > 0 && len(ret) < dispatcher.MaxInventorySize-1 {
		curr := q[0]
		q = q[1:]
		block, err := m.chain.FindBlock(curr)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"hash": curr.Hex(),
			}).Debug("Failed to find block with given hash")
			return ret
		}
		ret = append(ret, curr.Hex())
		if curr == end {
			break
		}

		if block.Height < lfbHeight {
			// Enqueue finalized child.
			for _, child := range block.Children {
				block, err := m.chain.FindBlock(child)
				if err != nil {
					m.logger.WithFields(log.Fields{
						"err":  err,
						"hash": curr.Hex(),
					}).Debug("Failed to load block")
					return ret
				}
				if block.Status.IsFinalized() {
					q = append(q, block.Hash())
					break
				}
			}
		} else {
			// Enqueue all children.
			q = append(q, block.Children...)
		}
	}

	// Make sure response is in size limit.
	if len(ret) > dispatcher.MaxInventorySize {
		ret = ret[:dispatcher.MaxInventorySize-1]
	}

	// Add last finalized block in the end so that receiver is aware of latest network state.
	ret = append(ret, m.consensus.GetLastFinalizedBlock().Hash().Hex())

	return ret
}

func (m *SyncManager) collectHeaders(start common.Hash, end common.Hash) Headers {
	ret := []*core.BlockHeader{}

	lfbHeight := m.consensus.GetLastFinalizedBlock().Height
	q := []common.Hash{start}
	for len(q) > 0 && len(ret) < dispatcher.MaxInventorySize-1 {
		curr := q[0]
		q = q[1:]
		block, err := m.chain.FindBlock(curr)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"hash": curr.Hex(),
			}).Debug("Failed to find block with given hash")
			return Headers{
				HeaderArray: ret,
			}
		}
		ret = append(ret, block.BlockHeader)
		if curr == end {
			break
		}

		if block.Height < lfbHeight {
			// Enqueue finalized child.
			for _, child := range block.Children {
				block, err := m.chain.FindBlock(child)
				if err != nil {
					m.logger.WithFields(log.Fields{
						"err":  err,
						"hash": curr.Hex(),
					}).Debug("Failed to load block")
					return Headers{
						HeaderArray: ret,
					}
				}
				if block.Status.IsFinalized() {
					q = append(q, block.Hash())
					break
				}
			}
		} else {
			// Enqueue all children.
			q = append(q, block.Children...)
		}
	}

	// Make sure response is in size limit.
	if len(ret) > dispatcher.MaxInventorySize {
		ret = ret[:dispatcher.MaxInventorySize-1]
	}

	// Add last finalized block in the end so that receiver is aware of latest network state.
	ret = append(ret, m.consensus.GetLastFinalizedBlock().BlockHeader)

	return Headers{
		HeaderArray: ret,
	}
}

func (m *SyncManager) handleInvRequest(peerID string, req *dispatcher.InventoryRequest) {
	m.logger.WithFields(log.Fields{
		"channelID":   req.ChannelID,
		"startHashes": req.Starts,
		"endHash":     req.End,
		"peerID":      peerID,
	}).Debug("Received inventory request")

	switch req.ChannelID {
	case common.ChannelIDBlock:

		start := m.locateStart(req.Starts)
		if start.IsEmpty() {
			m.logger.WithFields(log.Fields{
				"channelID": req.ChannelID,
				"peerID":    peerID,
			}).Debug("No start hash can be found in local chain")
			return
		}

		end := common.HexToHash(req.End)
		blocks := m.collectBlocks(start, end)

		// Send header response
		headers := m.collectHeaders(start, end)
		payload, err := rlp.EncodeToBytes(headers)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"err":          err,
				"headerHashes": blocks,
				"peerID":       peerID,
			}).Error("Failed to encode headers")
		} else {
			hresp := dispatcher.DataResponse{ChannelID: common.ChannelIDHeader, Payload: payload}
			m.dispatcher.SendData([]string{peerID}, hresp)
		}
		// Send Inventory response. compatible with outdated nodes
		resp := dispatcher.InventoryResponse{ChannelID: common.ChannelIDBlock, Entries: blocks}
		m.logger.WithFields(log.Fields{
			"channelID":         resp.ChannelID,
			"len(resp.Entries)": len(resp.Entries),
			"peerID":            peerID,
		}).Debug("Sending inventory response")
		m.dispatcher.SendInventory([]string{peerID}, resp)
	default:
		m.logger.WithFields(log.Fields{"channelID": req.ChannelID}).Warn("Unsupported channelID in received InvRequest")
	}

}

func (m *SyncManager) handleInvResponse(peerID string, resp *dispatcher.InventoryResponse) {
	m.logger.WithFields(log.Fields{
		"channelID":   resp.ChannelID,
		"InvResponse": resp,
		"peerID":      peerID,
	}).Debug("Received Inventory Response")

	switch resp.ChannelID {
	case common.ChannelIDBlock:
		fromGossip := len(resp.Entries) == 1
		for idx, hashStr := range resp.Entries {
			if idx > dispatcher.MaxInventorySize-1 {
				break
			}
			hash := common.HexToHash(hashStr)
			m.requestMgr.AddHash(hash, []string{peerID}, fromGossip)
		}
		if !fromGossip {
			m.requestMgr.AddActivePeer(peerID)
		}
	default:
		m.logger.WithFields(log.Fields{
			"channelID": resp.ChannelID,
			"peerID":    peerID,
		}).Warn("Unsupported channelID in received Inventory Request")
	}
}

func (m *SyncManager) handleDataRequest(peerID string, data *dispatcher.DataRequest) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		if len(data.Entries) == 1 { // compatible with old version
			m.sendSingleBlock(peerID, data.Entries[0], data.ChannelID)
		} else {
			blocks := &Blocks{}
			for _, hashStr := range data.Entries {
				hash := common.HexToHash(hashStr)
				block, err := m.chain.FindBlock(hash)
				if err != nil {
					m.logger.WithFields(log.Fields{
						"channelID": data.ChannelID,
						"hashStr":   hashStr,
						"err":       err,
						"peerID":    peerID,
					}).Debug("Failed to find hash string locally")
					return
				}
				blocks.BlockArray = append(blocks.BlockArray, block.Block)
			}
			payload, err := rlp.EncodeToBytes(blocks)
			if err != nil {
				m.logger.WithFields(log.Fields{
					"blocks": len(blocks.BlockArray),
					"peerID": peerID,
				}).Error("Failed to encode block")
				return
			}
			sendData := dispatcher.DataResponse{
				ChannelID: common.ChannelIDBlock,
				Payload:   payload,
			}
			m.logger.WithFields(log.Fields{
				"channelID":     sendData.ChannelID,
				"start hashStr": blocks.BlockArray[0].Hash().String(),
				"amount":        len(blocks.BlockArray),
				"peerID":        peerID,
			}).Debug("Sending requested block")
			m.dispatcher.SendData([]string{peerID}, sendData)
		}
	default:
		m.logger.WithFields(log.Fields{
			"channelID": data.ChannelID,
		}).Warn("Unsupported channelID in received DataRequest")
	}
}

func (m *SyncManager) sendSingleBlock(peerID string, hashStr string, channelID common.ChannelIDEnum) {
	hash := common.HexToHash(hashStr)
	block, err := m.chain.FindBlock(hash)
	if err != nil {
		m.logger.WithFields(log.Fields{
			"channelID": channelID,
			"hashStr":   hashStr,
			"err":       err,
			"peerID":    peerID,
		}).Debug("Failed to find hash string locally")
		return
	}

	payload, err := rlp.EncodeToBytes(block.Block)
	if err != nil {
		m.logger.WithFields(log.Fields{
			"block":  block,
			"peerID": peerID,
		}).Error("Failed to encode block")
		return
	}
	data := dispatcher.DataResponse{
		ChannelID: common.ChannelIDBlock,
		Payload:   payload,
	}
	m.logger.WithFields(log.Fields{
		"channelID": data.ChannelID,
		"hashStr":   hashStr,
		"peerID":    peerID,
	}).Debug("Sending requested block")
	m.dispatcher.SendData([]string{peerID}, data)
}

func Fuzz(data []byte) int {
	if len(data) == 0 {
		return -1
	}
	if data[0]%4 == 0 {
		block := core.NewBlock()
		err := rlp.DecodeBytes(data[1:], block)
		if err != nil {
			return 1
		}
		return 0
	}
	if data[0]%4 == 1 {
		vote := core.Vote{}
		err := rlp.DecodeBytes(data[1:], &vote)
		if err != nil {
			return 1
		}
		return 0
	}
	if data[0]%4 == 2 {
		proposal := &core.Proposal{}
		err := rlp.DecodeBytes(data[1:], proposal)
		if err != nil {
			return 1
		}
		return 0
	}
	if _, err := decodeMessage(data); err != nil {
		return 1
	}
	return 0
}

func (m *SyncManager) handleDataResponse(peerID string, data *dispatcher.DataResponse) {
	switch data.ChannelID {
	case common.ChannelIDBlock:
		maxReceivedHeight := uint64(0)
		block := core.NewBlock()
		err := rlp.DecodeBytes(data.Payload, block)
		if err != nil {
			//check if payload is blocks
			blocks := &Blocks{}
			err = rlp.DecodeBytes(data.Payload, blocks)
			if err != nil {
				m.logger.WithFields(log.Fields{
					"channelID": data.ChannelID,
					"payload":   data.Payload,
					"error":     err,
					"peerID":    peerID,
				}).Warn("Failed to decode DataResponse payload")
				return
			}
			for _, block = range blocks.BlockArray {
				m.logger.WithFields(log.Fields{
					"block.Hash":   block.Hash().Hex(),
					"block.Parent": block.Parent.Hex(),
					"block.Height": block.Height,
					"peer":         peerID,
				}).Debug("Received block")
				m.handleBlock(block)
				if block.Height > maxReceivedHeight {
					maxReceivedHeight = block.Height
				}
			}
		} else {
			m.logger.WithFields(log.Fields{
				"block.Hash":   block.Hash().Hex(),
				"block.Parent": block.Parent.Hex(),
				"block.Height": block.Height,
				"peer":         peerID,
			}).Debug("Received block")
			m.handleBlock(block)
			maxReceivedHeight = block.Height
		}
	case common.ChannelIDVote:
		vote := core.Vote{}
		err := rlp.DecodeBytes(data.Payload, &vote)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.logger.WithFields(log.Fields{
			"vote.Hash":  vote.Block.Hex(),
			"vote.ID":    vote.ID.Hex(),
			"vote.Epoch": vote.Epoch,
			"peer":       peerID,
		}).Debug("Received vote")
		m.handleVote(vote)
	case common.ChannelIDProposal:
		proposal := &core.Proposal{}
		err := rlp.DecodeBytes(data.Payload, proposal)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.logger.WithFields(log.Fields{
			"proposal": proposal,
			"peer":     peerID,
		}).Debug("Received proposal")
		m.handleProposal(proposal)
	case common.ChannelIDGuardian:
		vote := &core.AggregatedVotes{}
		err := rlp.DecodeBytes(data.Payload, vote)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.logger.WithFields(log.Fields{
			"vote.Hash":       vote.Block.Hex(),
			"vote.GCP":        vote.Gcp.Hex(),
			"vote.Multiplies": vote.Multiplies,
			"peer":            peerID,
		}).Debug("Received guardian vote")
		m.handleGuardianVote(vote)
	case common.ChannelIDEliteEdgeNodeVote:
		vote := &core.EENVote{}
		err := rlp.DecodeBytes(data.Payload, vote)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		// m.logger.WithFields(log.Fields{
		// 	"vote.Block": vote.Block.Hex(),
		// 	"peer":       peerID,
		// }).Debug("Received elite edge node vote")
		m.handleEliteEdgeNodeVote(vote)
	case common.ChannelIDAggregatedEliteEdgeNodeVotes:
		vote := &core.AggregatedEENVotes{}
		err := rlp.DecodeBytes(data.Payload, vote)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Warn("Failed to decode DataResponse payload")
			return
		}
		m.logger.WithFields(log.Fields{
			"vote.Block":      vote.Block.Hex(),
			"vote.Multiplies": vote.Multiplies,
			"peer":            peerID,
		}).Debug("Received aggregated elite edge node vote")
		m.handleAggregatedEliteEdgeNodeVotes(vote)
	case common.ChannelIDHeader:
		headers := &Headers{}
		err := rlp.DecodeBytes(data.Payload, headers)
		if err != nil {
			m.logger.WithFields(log.Fields{
				"channelID": data.ChannelID,
				"payload":   data.Payload,
				"error":     err,
				"peerID":    peerID,
			}).Debug("Failed to decode HeaderResponse payload")
			return
		}
		for _, header := range headers.HeaderArray {
			m.logger.WithFields(log.Fields{
				"header.Hash":   header.Hash().Hex(),
				"header.Parent": header.Parent.Hex(),
				"header.Height": header.Height,
				"peer":          peerID,
			}).Debug("Received header")
			m.handleHeader(header, []string{peerID})
		}
	default:
		m.logger.WithFields(log.Fields{
			"channelID": data.ChannelID,
		}).Warn("Unsupported channelID in received DataResponse")
	}
}

func (sm *SyncManager) handleProposal(p *core.Proposal) {
	if p.Votes != nil {
		for _, vote := range p.Votes.Votes() {
			sm.handleVote(vote)
		}
	}
	sm.handleBlock(p.Block)
}

func (sm *SyncManager) handleHeader(header *core.BlockHeader, peerID []string) {
	if eb, err := sm.chain.FindBlock(header.Hash()); err == nil && !eb.Status.IsPending() {
		sm.logger.WithFields(log.Fields{
			"block hash":   header.Hash().String(),
			"block height": header.Height,
		}).Debug("Header/Block is already in chain")
		return
	}

	if hash, ok := core.HardcodeBlockHashes[header.Height]; ok {
		if hash != header.Hash().Hex() {
			sm.logger.WithFields(log.Fields{
				"block hash":   header.Hash().String(),
				"block height": header.Height,
			}).Debug("hardcoded block")
			return
		}
	}

	lfbHeight := sm.consensus.GetLastFinalizedBlock().Height
	tipHeight := sm.consensus.GetTip(true).Height
	if header.Height > lfbHeight && header.Height <= tipHeight+dispatcher.MaxInventorySize+1 {
		sm.requestMgr.AddHeader(header, peerID)
	}
}

func (sm *SyncManager) handleBlock(block *core.Block) {
	if eb, err := sm.chain.FindBlock(block.Hash()); err == nil && !eb.Status.IsPending() {
		sm.logger.WithFields(log.Fields{
			"block hash":   block.Hash().String(),
			"block height": block.Height,
		}).Debug("block is already in chain")
		return
	}

	if hash, ok := core.HardcodeBlockHashes[block.Height]; ok {
		if hash != block.Hash().Hex() {
			sm.logger.WithFields(log.Fields{
				"block hash":   block.Hash().String(),
				"block height": block.Height,
			}).Debug("hardcoded block")
			return
		}
	} else if res := block.Validate(sm.chain.ChainID); res.IsError() {
		sm.logger.WithFields(log.Fields{
			"block hash":   block.Hash().String(),
			"block height": block.Height,
		}).Debug("chain ID is invalid")
		return
	}

	sm.requestMgr.AddBlock(block)

	p2pOpt := common.P2POptEnum(viper.GetInt(common.CfgP2POpt))
	if sm.requestMgr.IsGossipBlock(block.Hash()) && p2pOpt != common.P2POptLibp2p {
		// Gossip the block out using hash
		sm.dispatcher.SendInventory([]string{}, dispatcher.InventoryResponse{
			ChannelID: common.ChannelIDBlock,
			Entries:   []string{block.Hash().Hex()},
		})

		// Gossip the block out using header
		headers := Headers{
			HeaderArray: []*core.BlockHeader{block.BlockHeader},
		}
		payload, err := rlp.EncodeToBytes(headers)
		if err != nil {
			sm.logger.WithFields(log.Fields{
				"block hash":   block.Hash().String(),
				"block height": block.Height,
				"err":          err.Error(),
			}).Debug("failed to encode header")
			return
		}
		hresp := dispatcher.DataResponse{ChannelID: common.ChannelIDHeader, Payload: payload}
		sm.dispatcher.SendData([]string{}, hresp)
	}
}

func (sm *SyncManager) handleVote(vote core.Vote) {
	votes := sm.chain.FindVotesByHash(vote.Block).Votes()
	for _, v := range votes {
		// Check if vote already processed.
		if v.Block == vote.Block && v.Epoch == vote.Epoch && v.Height == vote.Height && v.ID == vote.ID {
			return
		}
	}
	// Ignore vote for disposed blocks.
	if b, err := sm.chain.FindBlock(vote.Block); err == nil {
		if b.Status == core.BlockStatusDisposed {
			return
		}
	}

	sm.PassdownMessage(vote)

	p2pOpt := common.P2POptEnum(viper.GetInt(common.CfgP2POpt))
	if p2pOpt != common.P2POptLibp2p {
		// Need to manually gossip if not using Libp2p
		hash := vote.Hash()
		if sm.voteCache.Contains(hash) {
			return
		}
		sm.voteCache.Add(hash, struct{}{})

		payload, err := rlp.EncodeToBytes(vote)
		if err != nil {
			sm.logger.WithFields(log.Fields{"vote": vote}).Error("Failed to encode vote")
			return
		}
		msg := dispatcher.DataResponse{
			ChannelID: common.ChannelIDVote,
			Payload:   payload,
		}
		sm.dispatcher.SendData([]string{}, msg)
	}
}

func (sm *SyncManager) handleGuardianVote(vote *core.AggregatedVotes) {
	sm.PassdownMessage(vote)
}

func (sm *SyncManager) handleEliteEdgeNodeVote(vote *core.EENVote) {
	sm.PassdownMessage(vote)
}

func (sm *SyncManager) handleAggregatedEliteEdgeNodeVotes(vote *core.AggregatedEENVotes) {
	sm.PassdownMessage(vote)
}
