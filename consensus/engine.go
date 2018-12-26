package consensus

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/dispatcher"
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
	dispatcher       *dispatcher.Dispatcher
	validatorManager core.ValidatorManager
	ledger           core.Ledger

	incoming        chan interface{}
	finalizedBlocks chan *core.Block

	// Life cycle
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	mu            *sync.Mutex
	epochTimer    *time.Timer
	proposalTimer *time.Timer

	state *State

	rand *rand.Rand
}

// NewConsensusEngine creates a instance of ConsensusEngine.
func NewConsensusEngine(privateKey *crypto.PrivateKey, db store.Store, chain *blockchain.Chain, dispatcher *dispatcher.Dispatcher, validatorManager core.ValidatorManager) *ConsensusEngine {
	e := &ConsensusEngine{
		chain:      chain,
		dispatcher: dispatcher,

		privateKey: privateKey,

		incoming:        make(chan interface{}, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		finalizedBlocks: make(chan *core.Block, viper.GetInt(common.CfgConsensusMessageQueueSize)),

		wg: &sync.WaitGroup{},

		mu:    &sync.Mutex{},
		state: NewState(db, chain),

		validatorManager: validatorManager,
	}

	logger = util.GetLoggerForModule("consensus")
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

	// Verify configurations
	if viper.GetInt(common.CfgConsensusMaxEpochLength) <= viper.GetInt(common.CfgConsensusMinProposalWait) {
		log.WithFields(log.Fields{
			"CfgConsensusMaxEpochLength":  viper.GetInt(common.CfgConsensusMaxEpochLength),
			"CfgConsensusMinProposalWait": viper.GetInt(common.CfgConsensusMinProposalWait),
		}).Fatal("Invalid configuration: max epoch length must be larger than minimal proposal wait")
	}

	// Set ledger state pointer to intial state.
	lastCC := e.state.GetHighestCCBlock()
	e.ledger.ResetState(lastCC.Height, lastCC.StateHash)

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
				if e.shouldVote(e.GetEpoch()) {
					e.vote()
				}
				break Epoch
			case <-e.proposalTimer.C:
				e.propose()
			}
		}
	}
}

func (e *ConsensusEngine) enterEpoch() {
	// Reset timers.
	if e.epochTimer != nil {
		e.epochTimer.Stop()
	}
	e.epochTimer = time.NewTimer(time.Duration(viper.GetInt(common.CfgConsensusMaxEpochLength)) * time.Second)

	if e.proposalTimer != nil {
		e.proposalTimer.Stop()
	}
	if e.shouldPropose(e.GetEpoch()) {
		e.proposalTimer = time.NewTimer(time.Duration(viper.GetInt(common.CfgConsensusMinProposalWait)) * time.Second)
	} else {
		e.proposalTimer = time.NewTimer(math.MaxInt64)
		e.proposalTimer.Stop()
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
	case core.Vote:
		e.logger.WithFields(log.Fields{"vote": m}).Debug("Received vote")
		return e.handleVote(m)
	case *core.Block:
		e.logger.WithFields(log.Fields{"block": m}).Debug("Received block")
		e.handleBlock(m)
	default:
		log.Errorf("Unknown message type: %v", m)
		panic(fmt.Sprintf("Unknown message type: %v", m))
	}

	return false
}

func (e *ConsensusEngine) handleBlock(block *core.Block) {
	parent, err := e.chain.FindBlock(block.Parent)
	if err != nil {
		e.logger.WithFields(log.Fields{
			"error":  err,
			"parent": block.Parent.Hex(),
			"block":  block.Hash().Hex(),
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
			"parent":          block.Parent.Hex(),
			"block":           block.Hash().Hex(),
			"block.StateHash": block.StateHash.Hex(),
		}).Error("Failed to apply block Txs")
		return
	}

	if !e.shouldVote(e.GetEpoch()) {
		return
	}

	// Skip voting for block older than current best known epoch.
	if block.Epoch < e.GetEpoch() {
		e.logger.WithFields(log.Fields{
			"block.Epoch": block.Epoch,
			"block.Hash":  block.Hash().Hex(),
			"e.epoch":     e.GetEpoch(),
		}).Debug("Skipping voting for block from previous epoch")
		return
	}

	e.vote()
}

func (e *ConsensusEngine) shouldVote(epoch uint64) bool {
	return e.shouldVoteByID(epoch, e.privateKey.PublicKey().Address())
}

func (e *ConsensusEngine) shouldVoteByID(epoch uint64, id common.Address) bool {
	validators := e.validatorManager.GetValidatorSetForEpoch(epoch)
	_, err := validators.GetValidator(id)
	return err == nil
}

func (e *ConsensusEngine) vote() {
	tip := e.state.GetTip()

	var vote core.Vote
	lastVote := e.state.GetLastVote()
	if lastVote.Height != 0 && lastVote.Height >= tip.Height {
		// Voting height should be monotonically increasing.
		e.logger.WithFields(log.Fields{
			"vote": lastVote,
		}).Debug("Repeating vote at height")
		vote = lastVote
		vote.Epoch = e.GetEpoch()
	} else if localHCC := e.state.GetHighestCCBlock().Hash(); lastVote.Height != 0 && tip.HCC != localHCC {
		// HCC in candidate block must equal local highest CC.
		e.logger.WithFields(log.Fields{
			"vote":      lastVote,
			"tip.HCC":   tip.HCC.Hex(),
			"local.HCC": localHCC.Hex(),
		}).Debug("Repeating vote due to mismatched HCC")
		vote = lastVote
		vote.Epoch = e.GetEpoch()
	} else {
		vote = e.createVote(tip.Block)
		e.state.SetLastVote(vote)
		e.logger.WithFields(log.Fields{
			"vote": vote,
		}).Debug("Sending vote")
	}

	payload, err := rlp.EncodeToBytes(vote)
	if err != nil {
		e.logger.WithFields(log.Fields{"vote": vote}).Error("Failed to encode vote")
		return
	}
	voteMsg := dispatcher.DataResponse{
		ChannelID: common.ChannelIDVote,
		Payload:   payload,
	}
	e.handleVote(vote)
	e.dispatcher.SendData([]string{}, voteMsg)
}

func (e *ConsensusEngine) createVote(block *core.Block) core.Vote {
	vote := core.Vote{
		Block:  block.Hash(),
		Height: block.Height,
		ID:     e.privateKey.PublicKey().Address(),
		Epoch:  e.GetEpoch(),
	}
	sig, err := e.privateKey.Sign(vote.SignBytes())
	if err != nil {
		e.logger.WithFields(log.Fields{"error": err}).Panic("Failed to sign vote")
	}
	vote.SetSignature(sig)
	return vote
}

func (e *ConsensusEngine) validateVote(vote core.Vote) bool {
	if res := vote.Validate(); res.IsError() {
		e.logger.WithFields(log.Fields{
			"err": res.String(),
		}).Warn("Ignoring invalid vote")
		return false
	}
	if !e.shouldVoteByID(vote.Epoch, vote.ID) {
		e.logger.WithFields(log.Fields{
			"vote.Epoch": vote.Epoch,
			"vote.ID":    vote.ID,
		}).Warn("Ignoring invalid vote from non-validator")
		return false
	}
	return true
}

func (e *ConsensusEngine) handleVote(vote core.Vote) (endEpoch bool) {
	if !e.validateVote(vote) {
		return
	}

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
		return
	}
	block, err := e.Chain().FindBlock(vote.Block)
	if err != nil {
		e.logger.WithFields(log.Fields{"vote.block": vote.Block.Hex()}).Warn("Block hash in vote is not found")
		return
	}

	// Ingore outdated votes.
	highestCCBlockHeight := e.state.GetHighestCCBlock().Height
	if block.Height < highestCCBlockHeight {
		e.logger.WithFields(log.Fields{
			"vote":                 vote,
			"vote.Block.Height":    block.Height,
			"HeightCCBlock.Height": highestCCBlockHeight,
		}).Debug("Ignoring outdated vote")
		return
	}

	votes, err := e.state.GetVoteSetByBlock(vote.Block)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err}).Panic("Failed to retrieve vote set by block")
	}
	if validators.HasMajority(votes) {
		e.processCCBlock(block)
	}

	return
}

// GetTip return the block to be extended from.
func (e *ConsensusEngine) GetTip() *core.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.state.GetTip()
}

// GetSummary returns a summary of consensus state.
func (e *ConsensusEngine) GetSummary() *StateStub {
	return e.state.GetSummary()
}

// FinalizedBlocks returns a channel that will be published with finalized blocks by the engine.
func (e *ConsensusEngine) FinalizedBlocks() chan *core.Block {
	return e.finalizedBlocks
}

func (e *ConsensusEngine) processCCBlock(ccBlock *core.ExtendedBlock) {
	if ccBlock.Height <= e.state.GetHighestCCBlock().Height {
		return
	}

	e.logger.WithFields(log.Fields{"ccBlock.Hash": ccBlock.Hash().Hex(), "c.epoch": e.state.GetEpoch()}).Debug("Updating highestCCBlock")

	e.chain.CommitBlock(ccBlock.Hash())

	parent, err := e.Chain().FindBlock(ccBlock.Parent)
	if err != nil {
		e.logger.WithFields(log.Fields{"err": err, "hash": ccBlock.Parent}).Error("Failed to load block")
		return
	}
	if parent.Status == core.BlockStatusCommitted {
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

	e.logger.WithFields(log.Fields{"block.Hash": block.Hash().Hex()}).Info("Finalizing block")

	e.state.SetLastFinalizedBlock(block)
	e.ledger.FinalizeState(block.Height, block.StateHash)

	// Mark block and its ancestors as finalized.
	e.chain.FinalizePreviousBlocks(block.Hash())

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
	if proposer.ID().Hex() != e.ID() {
		return false
	}
	if e.GetEpoch() == 0 {
		return false
	}
	return true
}

func (e *ConsensusEngine) createProposal() (core.Proposal, error) {
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
	block.HCC = e.state.GetHighestCCBlock().Hash()

	newRoot, txs, result := e.ledger.ProposeBlockTxs()
	if result.IsError() {
		err := fmt.Errorf("Failed to collect Txs for block proposal: %v", result.String())
		return core.Proposal{}, err
	}
	block.Txs = txs
	block.StateHash = newRoot

	proposal := core.Proposal{
		Block:      block,
		ProposerID: common.HexToAddress(e.ID()),
	}

	// Add votes that might help peers progress, e.g. votes on last CC block and latest epoch
	// votes.
	lastCC := e.state.GetHighestCCBlock()
	lastCCVotes, err := e.state.GetVoteSetByBlock(lastCC.Hash())
	if err != nil {
		e.logger.WithFields(log.Fields{"error": err, "block": lastCC.Hash().Hex()}).Warn("Failed to load votes for last CC block")
	}
	epochVotes, err := e.state.GetEpochVotes()
	if err != nil {
		e.logger.WithFields(log.Fields{"error": err}).Warn("Failed to load epoch votes")
	}
	proposal.Votes = lastCCVotes.Merge(epochVotes).UniqueVoterAndBlock()
	selfVote := e.createVote(block)
	proposal.Votes.AddVote(selfVote)

	_, err = e.chain.AddBlock(block)
	if err != nil {
		return core.Proposal{}, errors.Wrap(err, "Failed to add proposed block to chain")
	}

	e.handleBlock(block)

	return proposal, nil
}

func (e *ConsensusEngine) propose() {
	var proposal core.Proposal
	var err error
	lastProposal := e.state.GetLastProposal()
	if lastProposal.Block != nil && e.GetEpoch() == lastProposal.Block.Epoch {
		proposal = lastProposal
		e.logger.WithFields(log.Fields{"proposal": proposal}).Info("Repeating proposal")
	} else {
		proposal, err = e.createProposal()
		if err != nil {
			e.logger.WithFields(log.Fields{"error": err}).Error("Failed to create proposal")
			return
		}
		e.state.LastProposal = proposal

		e.logger.WithFields(log.Fields{"proposal": proposal}).Info("Making proposal")
	}

	payload, err := rlp.EncodeToBytes(proposal)
	if err != nil {
		e.logger.WithFields(log.Fields{"proposal": proposal}).Error("Failed to encode proposal")
		return
	}
	proposalMsg := dispatcher.DataResponse{
		ChannelID: common.ChannelIDProposal,
		Payload:   payload,
	}
	e.dispatcher.SendData([]string{}, proposalMsg)
}
