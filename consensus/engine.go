package consensus

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "consensus"})

var _ core.ConsensusEngine = (*ConsensusEngine)(nil)

// ConsensusEngine is the default implementation of the Engine interface.
type ConsensusEngine struct {
	logger *log.Entry

	privateKey *crypto.PrivateKey

	chain            *blockchain.Chain
	network          p2p.Network
	validatorManager core.ValidatorManager
	ledger           core.Ledger

	incoming        chan interface{}
	finalizedBlocks chan *core.Block

	// Life cycle
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	mu         *sync.Mutex
	epochTimer *time.Timer

	state *State

	rand *rand.Rand
}

// NewConsensusEngine creates a instance of ConsensusEngine.
func NewConsensusEngine(privateKey *crypto.PrivateKey, db store.Store, chain *blockchain.Chain, network p2p.Network, validatorManager core.ValidatorManager) *ConsensusEngine {
	e := &ConsensusEngine{
		chain:   chain,
		network: network,

		privateKey: privateKey,

		incoming:        make(chan interface{}, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		finalizedBlocks: make(chan *core.Block, viper.GetInt(common.CfgConsensusMessageQueueSize)),

		wg: &sync.WaitGroup{},

		mu:    &sync.Mutex{},
		state: NewState(db, chain),

		validatorManager: validatorManager,
	}

	logger = util.GetLoggerForModule("consensus")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": network.ID()})
	}
	e.logger = logger

	e.logger.WithFields(log.Fields{"state": e.state}).Info("Starting state")

	e.rand = rand.New(rand.NewSource(time.Now().Unix()))

	return e
}

func (e *ConsensusEngine) SetLedger(ledger core.Ledger) {
	e.ledger = ledger
}

// ID returns the identifier of current node.
func (e *ConsensusEngine) ID() string {
	return e.privateKey.PublicKey().Address().Hex()
}

// PrivateKey returns the private key
func (e *ConsensusEngine) PrivateKey() *crypto.PrivateKey {
	return e.privateKey
}

// Chain return a pointer to the underlying chain store.
func (e *ConsensusEngine) Chain() *blockchain.Chain {
	return e.chain
}

// Network returns a pointer to the underlying network.
func (e *ConsensusEngine) Network() p2p.Network {
	return e.network
}

// GetEpoch returns the current epoch
func (e *ConsensusEngine) GetEpoch() uint64 {
	return e.state.GetEpoch()
}

// GetValidatorManager returns a pointer to the valiator manager.
func (e *ConsensusEngine) GetValidatorManager() core.ValidatorManager {
	return e.validatorManager
}

// Start starts sub components and kick off the main loop.
func (e *ConsensusEngine) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	e.ctx = c
	e.cancel = cancel

	e.wg.Add(1)
	go e.mainLoop()
}

// Stop notifies all goroutines to stop without blocking.
func (e *ConsensusEngine) Stop() {
	e.cancel()
}

// Wait blocks until all goroutines stop.
func (e *ConsensusEngine) Wait() {
	e.wg.Wait()
}

func (e *ConsensusEngine) mainLoop() {
	defer e.wg.Done()

	for {
		e.enterEpoch()
	Epoch:
		for {
			select {
			case <-e.ctx.Done():
				e.stopped = true
				return
			case msg := <-e.incoming:
				endEpoch := e.processMessage(msg)
				if endEpoch {
					break Epoch
				}
			case <-e.epochTimer.C:
				e.logger.WithFields(log.Fields{"e.epoch": e.GetEpoch()}).Debug("Epoch timeout. Repeating epoch")
				e.vote()
				break Epoch
			}
		}
	}
}

func (e *ConsensusEngine) enterEpoch() {
	// Reset timer.
	if e.epochTimer != nil {
		e.epochTimer.Stop()
	}
	e.epochTimer = time.NewTimer(time.Duration(viper.GetInt(common.CfgConsensusMaxEpochLength)) * time.Second)

	if e.shouldPropose(e.GetEpoch()) {
		e.propose()
	}
}

// GetChannelIDs implements the p2p.MessageHandler interface.
func (e *ConsensusEngine) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDVote,
	}
}

func (e *ConsensusEngine) AddMessage(msg interface{}) {
	e.incoming <- msg
}

func (e *ConsensusEngine) processMessage(msg interface{}) (endEpoch bool) {
	switch m := msg.(type) {
	case *core.Vote:
		return e.handleVote(*m)
	case *core.Block:
		e.handleBlock(m)
	default:
		log.Errorf("Unknown message type: %v", m)
		panic(fmt.Sprintf("Unknown message type: %v", m))
	}

	return false
}

func (e *ConsensusEngine) handleProposal(p core.Proposal) {
	e.logger.WithFields(log.Fields{"proposal": p}).Debug("Received proposal")

	if expectedProposer := e.validatorManager.GetProposerForEpoch(e.GetEpoch()).ID(); p.ProposerID != expectedProposer {
		e.logger.WithFields(log.Fields{"proposal": p, "p.proposerID": p.ProposerID, "expected proposer": expectedProposer}).Debug("Ignoring proposed block since proposer shouldn't propose in epoch")
		return
	}

	e.chain.AddBlock(p.Block)
	e.handleBlock(p.Block)
	e.handleCC(p.CommitCertificate)
	return
}

func (e *ConsensusEngine) handleBlock(block *core.Block) {
	e.logger.WithFields(log.Fields{"block": block}).Debug("Received block")

	if block.Epoch != e.GetEpoch() {
		e.logger.WithFields(log.Fields{
			"block.Epoch": block.Epoch,
			"block.Hash":  block.Hash,
			"e.epoch":     e.GetEpoch(),
		}).Debug("Received block from another epoch")
	}

	parent, err := e.chain.FindBlock(block.Parent)
	if err != nil {
		e.logger.WithFields(log.Fields{
			"error":  err,
			"parent": block.Parent,
			"block":  block.Hash,
		}).Error("Failed to find parent block")
		return
	}
	result := e.ledger.ResetState(parent.Height, parent.StateHash)
	if result.IsError() {
		e.logger.WithFields(log.Fields{
			"error":            result.Message,
			"parent.StateHash": parent.StateHash,
		}).Error("Failed to reset state to parent.StateHash")
		return
	}
	result = e.ledger.ApplyBlockTxs(block.Txs, block.StateHash)
	if result.IsError() {
		e.logger.WithFields(log.Fields{
			"error":           result.String(),
			"parent":          block.Parent,
			"block":           block.Hash,
			"block.StateHash": block.StateHash,
		}).Error("Failed to apply block Txs")
		return
	}

	e.vote()
}

func (e *ConsensusEngine) vote() {
	previousTip := e.state.GetTip()
	tip := e.state.SetTip()

	var header *core.BlockHeader
	if previousTip.Hash() == tip.Hash() || e.state.GetLastVoteHeight() >= tip.Height {
		e.logger.WithFields(log.Fields{
			"lastVoteHeight": e.state.GetLastVoteHeight(),
			"tip.Hash":       tip.Hash,
		}).Debug("Voting nil since already voted at height")
	} else {
		header = tip.BlockHeader
		e.state.SetLastVoteHeight(tip.Height)
	}

	vote := core.Vote{
		Block: header.Hash(),
		ID:    e.privateKey.PublicKey().Address(),
		Epoch: e.GetEpoch(),
	}

	e.logger.WithFields(log.Fields{"vote.block": vote.Block}).Debug("Sending vote")

	payload, err := rlp.EncodeToBytes(vote)
	if err != nil {
		e.logger.WithFields(log.Fields{"vote": vote}).Error("Failed to encode vote")
		return
	}
	data := dispatcher.DataResponse{
		ChannelID: common.ChannelIDVote,
		Payload:   payload,
	}
	voteMsg := p2ptypes.Message{
		ChannelID: common.ChannelIDVote,
		Content:   data,
	}
	e.AddMessage(&vote)
	e.network.Broadcast(voteMsg)
}

func (e *ConsensusEngine) handleCC(cc *core.CommitCertificate) {
	e.logger.WithFields(log.Fields{"cc": cc}).Debug("Received CC")

	if cc == nil {
		return
	}
	ccBlock, err := e.chain.FindBlock(cc.BlockHash)
	if err != nil {
		e.logger.WithFields(log.Fields{"blockhash": fmt.Sprintf("%v", cc.BlockHash)}).Error("Blockhash in commit certificate is not found")
		return
	}
	ccBlock.CommitCertificate = cc

	e.chain.SaveBlock(ccBlock)
	e.logger.WithFields(log.Fields{
		"error":             err,
		"block":             ccBlock,
		"commitCertificate": cc,
	}).Debug("Update block with commit certificate")

	e.processCCBlock(ccBlock)
}

func (e *ConsensusEngine) handleVote(vote core.Vote) (endEpoch bool) {
	e.logger.WithFields(log.Fields{"vote": vote}).Debug("Received vote")

	validators := e.validatorManager.GetValidatorSetForEpoch(e.state.GetEpoch())
	err := e.state.AddVote(&vote)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err}).Panic("Failed to add vote")
	}

	if vote.Epoch >= e.GetEpoch() {
		currentEpochVotes := core.NewVoteSet()
		allEpochVotes, err := e.state.GetEpochVotes()
		if err != nil {
			e.logger.WithFields(log.Fields{"err": err}).Panic("Failed to retrieve epoch votes")
		}
		for _, v := range allEpochVotes.Votes() {
			if v.Epoch >= vote.Epoch {
				currentEpochVotes.AddVote(v)
			}
		}

		if validators.HasMajority(currentEpochVotes) {
			nextEpoch := vote.Epoch + 1
			endEpoch = true

			e.logger.WithFields(log.Fields{
				"e.epoch":      e.GetEpoch,
				"nextEpoch":    nextEpoch,
				"epochVoteSet": currentEpochVotes,
			}).Debug("Majority votes for current epoch. Moving to new epoch")
			e.state.SetEpoch(nextEpoch)
		}
	}

	if vote.Block.IsEmpty() {
		e.logger.WithFields(log.Fields{"vote": vote}).Debug("Vote with empty block hash received")
		return
	}
	block, err := e.Chain().FindBlock(vote.Block)
	if err != nil {
		e.logger.WithFields(log.Fields{"vote.block": vote.Block}).Warn("Block hash in vote is not found")
		return
	}
	votes, err := e.state.GetVoteSetByBlock(vote.Block)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err}).Panic("Failed to retrieve vote set by block")
	}
	if validators.HasMajority(votes) {
		cc := &core.CommitCertificate{Votes: votes, BlockHash: vote.Block}
		block.CommitCertificate = cc

		e.chain.SaveBlock(block)
		e.processCCBlock(block)
	}

	return
}

// setTip sets the block to extended from by next proposal. Currently we use the highest block among highestCCBlock's
// descendants as the fork-choice rule.
func (e *ConsensusEngine) setTip() *core.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.state.SetTip()
}

// GetTip return the block to be extended from.
func (e *ConsensusEngine) GetTip() *core.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.state.GetTip()
}

// FinalizedBlocks returns a channel that will be published with finalized blocks by the engine.
func (e *ConsensusEngine) FinalizedBlocks() chan *core.Block {
	return e.finalizedBlocks
}

func (e *ConsensusEngine) processCCBlock(ccBlock *core.ExtendedBlock) {
	e.logger.WithFields(log.Fields{"ccBlock": ccBlock, "c.epoch": e.state.GetEpoch()}).Debug("Start processing ccBlock")
	defer e.logger.WithFields(log.Fields{"ccBlock": ccBlock, "c.epoch": e.state.GetEpoch()}).Debug("Done processing ccBlock")

	if ccBlock.Height > e.state.GetHighestCCBlock().Height {
		e.logger.WithFields(log.Fields{"ccBlock": ccBlock}).Debug("Updating highestCCBlock since ccBlock.Height > e.highestCCBlock.Height")
		e.state.SetHighestCCBlock(ccBlock)
	}

	parent, err := e.Chain().FindBlock(ccBlock.Parent)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err, "hash": ccBlock.Parent}).Error("Failed to load block")
		return
	}
	if parent.CommitCertificate != nil {
		e.finalizeBlock(parent)
	}
}

func (e *ConsensusEngine) finalizeBlock(block *core.ExtendedBlock) {
	if e.stopped {
		return
	}

	// Skip blocks that have already published.
	if block.Hash() == e.state.GetLastFinalizedBlock().Hash() {
		return
	}

	e.logger.WithFields(log.Fields{"block.Hash": block.Hash}).Info("Finalizing block")
	defer e.logger.WithFields(log.Fields{"block.Hash": block.Hash}).Info("Done Finalized block")

	e.state.SetLastFinalizedBlock(block)
	e.ledger.FinalizeState(block.Height, block.StateHash)

	// Mark block and its ancestors as finalized.
	e.chain.FinalizePreviousBlocks(block)

	// Force update TX index on block finalization so that the index doesn't point to
	// duplicate TX in fork.
	e.chain.AddTxsToIndex(block, true)

	select {
	case e.finalizedBlocks <- block.Block:
	default:
	}
}

func (e *ConsensusEngine) randHex() []byte {
	bytes := make([]byte, 10)
	e.rand.Read(bytes)
	return bytes
}

func (e *ConsensusEngine) shouldPropose(epoch uint64) bool {
	proposer := e.validatorManager.GetProposerForEpoch(epoch)
	return proposer.ID().Hex() == e.ID()
}

func (e *ConsensusEngine) propose() {
	tip := e.GetTip()
	result := e.ledger.ResetState(tip.Height, tip.StateHash)
	if result.IsError() {
		e.logger.WithFields(log.Fields{
			"error":         result.Message,
			"tip.StateHash": tip.StateHash,
		}).Panic("Failed to reset state to tip.StateHash")
	}

	block := core.NewBlock()
	block.ChainID = e.chain.ChainID
	block.Epoch = e.GetEpoch()
	block.Parent = tip.Hash()
	block.Height = tip.Height + 1
	block.Proposer = e.privateKey.PublicKey().Address()
	block.Timestamp = big.NewInt(time.Now().Unix())

	newRoot, txs, result := e.ledger.ProposeBlockTxs()
	if result.IsError() {
		e.logger.WithFields(log.Fields{"error": result.String()}).Error("Failed to collect Txs for block proposal")
		return
	}
	block.Txs = txs
	block.StateHash = newRoot

	lastCC := e.state.GetHighestCCBlock()
	proposal := core.Proposal{Block: block, ProposerID: common.HexToAddress(e.ID())}
	if lastCC.CommitCertificate != nil {
		proposal.CommitCertificate = lastCC.CommitCertificate.Copy()
	}

	e.logger.WithFields(log.Fields{"proposal": proposal}).Info("Making proposal")

	payload, err := rlp.EncodeToBytes(proposal)
	if err != nil {
		e.logger.WithFields(log.Fields{"proposal": proposal}).Error("Failed to encode proposal")
		return
	}
	data := dispatcher.DataResponse{
		ChannelID: common.ChannelIDProposal,
		Payload:   payload,
	}
	proposalMsg := p2ptypes.Message{
		ChannelID: common.ChannelIDProposal,
		Content:   data,
	}
	_, err = e.chain.AddBlock(block)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err}).Error("Failed to add proposed block to chain")
	}
	e.handleBlock(block)
	e.network.Broadcast(proposalMsg)
}
