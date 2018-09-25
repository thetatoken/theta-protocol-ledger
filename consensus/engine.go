package consensus

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/thetatoken/ukulele/dispatcher"
	"github.com/thetatoken/ukulele/rlp"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

var _ core.ConsensusEngine = (*ConsensusEngine)(nil)

// ConsensusEngine is the default implementation of the Engine interface.
type ConsensusEngine struct {
	logger *log.Entry

	chain   *blockchain.Chain
	network p2p.Network

	incoming        chan interface{}
	finalizedBlocks chan *core.Block

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	// TODO: persist state
	// Consensus state
	mu                 *sync.Mutex
	highestCCBlock     *core.ExtendedBlock
	lastFinalizedBlock *core.ExtendedBlock
	tip                *core.ExtendedBlock
	lastVoteHeight     uint32
	voteLog            map[uint32]core.Vote     // level -> vote
	collectedVotes     map[string]*core.VoteSet // block hash -> votes
	epochVotes         map[string]core.Vote     // Validator ID -> latest vote from this validator
	epochTimer         *time.Timer
	epoch              uint32
	validatorManager   core.ValidatorManager
	rand               *rand.Rand
}

// NewConsensusEngine creates a instance of ConsensusEngine.
func NewConsensusEngine(chain *blockchain.Chain, network p2p.Network, validators *core.ValidatorSet) *ConsensusEngine {
	e := &ConsensusEngine{
		chain:   chain,
		network: network,

		incoming:        make(chan interface{}, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		finalizedBlocks: make(chan *core.Block, viper.GetInt(common.CfgConsensusMessageQueueSize)),

		wg:   &sync.WaitGroup{},
		quit: make(chan struct{}),

		mu:                 &sync.Mutex{},
		highestCCBlock:     chain.Root,
		lastFinalizedBlock: chain.Root,
		tip:                chain.Root,
		voteLog:            make(map[uint32]core.Vote),
		collectedVotes:     make(map[string]*core.VoteSet),
		epochVotes:         make(map[string]core.Vote),
		validatorManager:   NewRotatingValidatorManager(validators),
		epoch:              0,
	}

	logger := util.GetLoggerForModule("consensus")
	if viper.GetBool(common.CfgLogPrintSelfID) {
		logger = logger.WithFields(log.Fields{"id": network.ID()})
	}
	e.logger = logger

	h := md5.New()
	io.WriteString(h, network.ID())
	seed := binary.BigEndian.Uint64(h.Sum(nil))
	e.rand = rand.New(rand.NewSource(int64(seed)))
	return e
}

// ID returns the identifier of current node.
func (e *ConsensusEngine) ID() string {
	return e.network.ID()
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
func (e *ConsensusEngine) GetEpoch() uint32 {
	return e.epoch
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
	e.wg.Add(1)
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
				e.logger.WithFields(log.Fields{"e.epoch": e.epoch}).Debug("Epoch timeout. Repeating epoch")
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

	if e.shouldPropose(e.epoch) {
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
	case core.Proposal:
		e.handleProposal(m)
	case core.Vote:
		return e.handleVote(m)
	case *core.Block:
		e.handleBlock(m)
	case *core.CommitCertificate:
		e.handleCC(m)
	default:
		log.Errorf("Unknown message type: %v", m)
	}

	return false
}

func (e *ConsensusEngine) handleProposal(p core.Proposal) {
	e.logger.WithFields(log.Fields{"proposal": p}).Debug("Received proposal")

	if expectedProposer := e.validatorManager.GetProposerForEpoch(e.epoch).ID(); p.ProposerID != expectedProposer {
		e.logger.WithFields(log.Fields{"proposal": p, "p.proposerID": p.ProposerID, "expected proposer": expectedProposer}).Debug("Ignoring proposed block since proposer shouldn't propose in epoch")
		return
	}

	e.handleBlock(&p.Block)
	e.handleCC(p.CommitCertificate)
	e.vote()
}

func (e *ConsensusEngine) handleBlock(block *core.Block) {
	e.logger.WithFields(log.Fields{"block": block}).Debug("Received block")

	var err error
	if block.Epoch != e.epoch {
		e.logger.WithFields(log.Fields{
			"block.Epoch": block.Epoch,
			"block.Hash":  block.Hash,
			"e.epoch":     e.epoch,
		}).Debug("Received block from another epoch")
	}
	_, err = e.chain.AddBlock(block)
	if err != nil {
		e.logger.WithFields(log.Fields{"block": block}).Error(err)
	}
}

func (e *ConsensusEngine) vote() {
	previousTip := e.GetTip()
	tip := e.setTip()

	var header *core.BlockHeader
	if bytes.Compare(previousTip.Hash, tip.Hash) == 0 || e.lastVoteHeight >= tip.Height {
		e.logger.WithFields(log.Fields{
			"lastVoteHeight": e.lastVoteHeight,
			"tip.Hash":       tip.Hash,
		}).Debug("Voting nil since already voted at height")
	} else {
		header = &tip.BlockHeader
		e.lastVoteHeight = tip.Height
	}

	vote := core.Vote{
		Block: header,
		ID:    e.ID(),
		Epoch: e.epoch,
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
	e.AddMessage(vote)
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

	validators := e.validatorManager.GetValidatorSetForEpoch(0)
	e.epochVotes[vote.ID] = vote

	if vote.Epoch >= e.epoch {
		epochVoteSet := core.NewVoteSet()
		for _, v := range e.epochVotes {
			if v.Epoch >= vote.Epoch {
				epochVoteSet.AddVote(v)
			}
		}
		if validators.HasMajority(epochVoteSet) {
			nextEpoch := vote.Epoch + 1
			endEpoch = true

			e.logger.WithFields(log.Fields{
				"e.epoch":      e.epoch,
				"nextEpoch":    nextEpoch,
				"epochVoteSet": epochVoteSet,
			}).Debug("Majority votes for current epoch. Moving to new epoch")
			e.epoch = nextEpoch
		}
	}

	if vote.Block == nil {
		e.logger.WithFields(log.Fields{"vote": vote}).Debug("Empty vote received")
		return
	}
	hs := hex.EncodeToString(vote.Block.Hash)
	block, err := e.Chain().FindBlock(vote.Block.Hash)
	if err != nil {
		e.logger.WithFields(log.Fields{"vote.block.hash": vote.Block.Hash}).Warn("Block hash in vote is not found")
		return
	}
	votes, ok := e.collectedVotes[hs]
	if !ok {
		votes = core.NewVoteSet()
		e.collectedVotes[hs] = votes
	}
	votes.AddVote(vote)
	if validators.HasMajority(votes) {
		cc := &core.CommitCertificate{Votes: votes, BlockHash: vote.Block.Hash}
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

	ret, _ := e.Chain().FindDeepestDescendant(e.highestCCBlock.Hash)
	e.tip = ret
	return ret
}

// GetTip return the block to be extended from.
func (e *ConsensusEngine) GetTip() *core.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.tip
}

// FinalizedBlocks returns a channel that will be published with finalized blocks by the engine.
func (e *ConsensusEngine) FinalizedBlocks() chan *core.Block {
	return e.finalizedBlocks
}

func (e *ConsensusEngine) processCCBlock(ccBlock *core.ExtendedBlock) {
	e.logger.WithFields(log.Fields{"ccBlock": ccBlock, "c.epoch": e.epoch}).Debug("Start processing ccBlock")
	defer e.logger.WithFields(log.Fields{"ccBlock": ccBlock, "c.epoch": e.epoch}).Debug("Done processing ccBlock")

	if ccBlock.Height > e.highestCCBlock.Height {
		e.logger.WithFields(log.Fields{"ccBlock": ccBlock}).Debug("Updating highestCCBlock since ccBlock.Height > e.highestCCBlock.Height")
		e.highestCCBlock = ccBlock
	}

	parent, err := e.Chain().FindBlock(ccBlock.Parent)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err, "hash": ccBlock.Parent}).Error("Failed to load block")
		return
	}
	if parent.CommitCertificate != nil {
		e.finalizeBlock(parent)
	}

	if ccBlock.Epoch >= e.epoch {
		e.logger.WithFields(log.Fields{"ccBlock": ccBlock, "e.epoch": e.epoch}).Debug("Advancing epoch")
		e.epoch = ccBlock.Epoch + 1
		e.enterEpoch()
	}
}

func (e *ConsensusEngine) finalizeBlock(block *core.ExtendedBlock) {
	if e.stopped {
		return
	}

	// Skip blocks that have already published.
	if bytes.Compare(block.Hash, e.lastFinalizedBlock.Hash) == 0 {
		return
	}

	e.logger.WithFields(log.Fields{"block.Hash": block.Hash}).Info("Finalizing block")
	defer e.logger.WithFields(log.Fields{"block.Hash": block.Hash}).Info("Done Finalized block")

	e.lastFinalizedBlock = block

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

func (e *ConsensusEngine) shouldPropose(epoch uint32) bool {
	proposer := e.validatorManager.GetProposerForEpoch(epoch)
	return proposer.ID() == e.ID()
}

func (e *ConsensusEngine) propose() {
	tip := e.GetTip()

	block := core.Block{}
	block.ChainID = e.chain.ChainID
	block.Hash = e.randHex()
	block.Epoch = e.epoch
	block.ParentHash = tip.Hash

	lastCC := e.highestCCBlock
	proposal := core.Proposal{Block: block, ProposerID: e.ID()}
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
	e.AddMessage(proposal)
	e.network.Broadcast(proposalMsg)
}
