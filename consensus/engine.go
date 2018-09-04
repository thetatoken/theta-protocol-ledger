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

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/p2p"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

var _ Engine = (*DefaultEngine)(nil)

// DefaultEngine is the default implementation of the Engine interface.
type DefaultEngine struct {
	chain   *blockchain.Chain
	network p2p.Network

	incoming        chan interface{}
	finalizedBlocks chan *blockchain.Block

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool

	// TODO: persist state
	// Consensus state
	mu                 *sync.Mutex
	highestCCBlock     *blockchain.ExtendedBlock
	lastFinalizedBlock *blockchain.ExtendedBlock
	tip                *blockchain.ExtendedBlock
	lastVoteHeight     uint32
	voteLog            map[uint32]blockchain.Vote     // level -> vote
	collectedVotes     map[string]*blockchain.VoteSet // block hash -> votes
	epochManager       *EpochManager
	epoch              uint32
	validatorManager   ValidatorManager
	rand               *rand.Rand
}

// NewEngine creates a instance of DefaultEngine.
func NewEngine(chain *blockchain.Chain, network p2p.Network, validators *ValidatorSet) *DefaultEngine {
	e := &DefaultEngine{
		chain:   chain,
		network: network,

		incoming:        make(chan interface{}, viper.GetInt(common.CfgConsensusMessageQueueSize)),
		finalizedBlocks: make(chan *blockchain.Block, viper.GetInt(common.CfgConsensusMessageQueueSize)),

		wg:   &sync.WaitGroup{},
		quit: make(chan struct{}),

		mu:                 &sync.Mutex{},
		highestCCBlock:     chain.Root,
		lastFinalizedBlock: chain.Root,
		tip:                chain.Root,
		voteLog:            make(map[uint32]blockchain.Vote),
		collectedVotes:     make(map[string]*blockchain.VoteSet),
		validatorManager:   NewRotatingValidatorManager(validators),
		epochManager:       NewEpochManager(),
		epoch:              0,
	}
	e.epochManager.Init(e)

	h := md5.New()
	io.WriteString(h, network.ID())
	seed := binary.BigEndian.Uint64(h.Sum(nil))
	e.rand = rand.New(rand.NewSource(int64(seed)))
	return e
}

// ID returns the identifier of current node.
func (e *DefaultEngine) ID() string {
	return e.network.ID()
}

// Chain return a pointer to the underlying chain store.
func (e *DefaultEngine) Chain() *blockchain.Chain {
	return e.chain
}

// Network returns a pointer to the underlying network.
func (e *DefaultEngine) Network() p2p.Network {
	return e.network
}

// Start starts sub components and kick off the main loop.
func (e *DefaultEngine) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	e.ctx = c
	e.cancel = cancel

	e.epochManager.Start(e.ctx)

	go e.mainLoop()
}

// Stop notifies all goroutines to stop without blocking.
func (e *DefaultEngine) Stop() {
	e.cancel()
}

// Wait blocks until all goroutines stop.
func (e *DefaultEngine) Wait() {
	e.epochManager.Wait()
	e.wg.Wait()
}

func (e *DefaultEngine) mainLoop() {
	e.wg.Add(1)
	defer e.wg.Done()

	e.enterNewEpoch(e.epochManager.GetEpoch())

	for {
		select {
		case <-e.ctx.Done():
			e.stopped = true
			return
		case msg := <-e.incoming:
			e.processMessage(msg)
		case newEpoch := <-e.epochManager.C:
			newEpoch = e.epochManager.GetEpoch()
			e.enterNewEpoch(newEpoch)
		}
	}
}

func (e *DefaultEngine) enterNewEpoch(newEpoch uint32) {
	e.epoch = newEpoch
	if e.shouldPropose(newEpoch) {
		e.propose()
	}
}

// GetChannelIDs implements the p2p.MessageHandler interface.
func (e *DefaultEngine) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDVote,
	}
}

func (e *DefaultEngine) AddMessage(msg interface{}) {
	e.incoming <- msg
}

// ParseMessage implements p2p.MessageHandler interface.
func (e *DefaultEngine) ParseMessage(peerID string, channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	// To be implemented..
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
	}
	return message, nil
}

func (e *DefaultEngine) processMessage(msg interface{}) error {
	switch m := msg.(type) {
	case Proposal:
		e.handleProposal(m)
	case blockchain.Vote:
		e.handleVote(m)
	case *blockchain.Block:
		e.handleBlock(m)
	case *blockchain.CommitCertificate:
		e.handleCC(m)
	default:
		log.Errorf("Unknown message type: %v", m)
	}

	return nil
}

func (e *DefaultEngine) handleProposal(p Proposal) {
	log.WithFields(log.Fields{"proposal": p, "id": e.ID()}).Debug("Received proposal")

	if expectedProposer := e.validatorManager.GetProposerForEpoch(e.epoch).ID(); p.ProposerID != expectedProposer {
		log.WithFields(log.Fields{"proposal": p, "id": e.ID(), "p.proposerID": p.ProposerID, "expected proposer": expectedProposer}).Debug("Ignoring proposed block since proposer shouldn't propose in epoch")
		return
	}

	e.handleBlock(&p.Block)
	e.handleCC(p.CommitCertificate)
	e.tryVote()
}

func (e *DefaultEngine) handleBlock(block *blockchain.Block) {
	var err error
	if block.Epoch != e.epoch {
		log.WithFields(log.Fields{"id": e.ID(), "block.Epoch": block.Epoch, "e.epoch": e.epoch}).Debug("Ignoring block from another epoch")
	} else {
		_, err = e.chain.AddBlock(block)
		if err != nil {
			log.WithFields(log.Fields{"id": e.ID(), "block": block}).Error(err)
			panic(err)
		}
	}

	e.tryVote()
}

func (e *DefaultEngine) tryVote() {
	previousTip := e.GetTip()
	tip := e.setTip()

	if bytes.Compare(previousTip.Hash, tip.Hash) == 0 || e.lastVoteHeight >= tip.Height {
		log.WithFields(log.Fields{"id": e.ID(), "lastVoteHeight": e.lastVoteHeight, "tip.Hash": tip.Hash}).Debug("Skip voting since has already voted at height")
		return
	}

	vote := blockchain.Vote{Block: &tip.BlockHeader, ID: e.ID()}
	e.lastVoteHeight = tip.Height

	log.WithFields(log.Fields{"vote.block.hash": vote.Block.Hash, "id": e.ID()}).Debug("Sending vote")

	voteMsg := p2ptypes.Message{
		ChannelID: common.ChannelIDVote,
		Content:   vote,
	}
	e.network.Broadcast(voteMsg)
}

func (e *DefaultEngine) handleCC(cc *blockchain.CommitCertificate) {
	if cc == nil {
		return
	}
	ccBlock, err := e.chain.FindBlock(cc.BlockHash)
	if err != nil {
		log.WithFields(log.Fields{"blockhash": fmt.Sprintf("%v", cc.BlockHash)}).Error("Blockhash in commit certificate is not found")
		return
	}
	ccBlock.CommitCertificate = cc

	e.chain.SaveBlock(ccBlock)
	log.WithFields(log.Fields{"id": e.ID(), "error": err, "block": ccBlock, "commitCertificate": cc}).Debug("Update block with commit certificate")

	e.processCCBlock(ccBlock)
}

func (e *DefaultEngine) handleVote(vote blockchain.Vote) {
	log.WithFields(log.Fields{"vote": vote, "id": e.ID()}).Debug("Received vote")

	hs := hex.EncodeToString(vote.Block.Hash)
	block, err := e.Chain().FindBlock(vote.Block.Hash)
	if err != nil {
		log.WithFields(log.Fields{"id": e.ID(), "vote.block.hash": vote.Block.Hash}).Warn("Block hash in vote is not found")
		return
	}
	votes, ok := e.collectedVotes[hs]
	if !ok {
		votes = blockchain.NewVoteSet()
		e.collectedVotes[hs] = votes
	}
	votes.AddVote(vote)

	validators := e.validatorManager.GetValidatorSetForEpoch(e.epoch)
	if validators.HasMajority(votes) {
		cc := &blockchain.CommitCertificate{Votes: votes, BlockHash: vote.Block.Hash}
		block.CommitCertificate = cc

		e.chain.SaveBlock(block)
		e.processCCBlock(block)
	}
}

// setTip sets the block to extended from by next proposal. Currently we use the highest block among highestCCBlock's
// descendants as the fork-choice rule.
func (e *DefaultEngine) setTip() *blockchain.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	ret, _ := e.Chain().FindDeepestDescendant(e.highestCCBlock.Hash)
	e.tip = ret
	return ret
}

// GetTip return the block to be extended from.
func (e *DefaultEngine) GetTip() *blockchain.ExtendedBlock {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.tip
}

// FinalizedBlocks returns a channel that will be published with finalized blocks by the engine.
func (e *DefaultEngine) FinalizedBlocks() chan *blockchain.Block {
	return e.finalizedBlocks
}

func (e *DefaultEngine) processCCBlock(ccBlock *blockchain.ExtendedBlock) {
	log.WithFields(log.Fields{"id": e.ID(), "ccBlock": ccBlock, "c.epoch": e.epoch}).Debug("Start processing ccBlock")
	defer log.WithFields(log.Fields{"id": e.ID(), "ccBlock": ccBlock, "c.epoch": e.epoch}).Debug("Done processing ccBlock")

	if ccBlock.Height > e.highestCCBlock.Height {
		log.WithFields(log.Fields{"id": e.ID(), "ccBlock": ccBlock}).Debug("Updating highestCCBlock since ccBlock.Height > e.highestCCBlock.Height")
		e.highestCCBlock = ccBlock
	}

	parent, err := e.Chain().FindBlock(ccBlock.Parent)
	if err != nil {
		log.WithFields(log.Fields{"id": e.ID(), "err": err, "hash": ccBlock.Parent}).Error("Failed to load block")
		return
	}
	if parent.CommitCertificate != nil {
		e.finalizeBlock(parent)
	}

	if ccBlock.Epoch >= e.epoch {
		log.WithFields(log.Fields{"id": e.ID(), "ccBlock": ccBlock, "e.epoch": e.epoch}).Debug("Advancing epoch")
		newEpoch := ccBlock.Epoch + 1
		e.enterNewEpoch(newEpoch)
		e.epochManager.SetEpoch(newEpoch)
	}
}

func (e *DefaultEngine) finalizeBlock(block *blockchain.ExtendedBlock) {
	if e.stopped {
		return
	}

	// Skip blocks that have already published.
	if bytes.Compare(block.Hash, e.lastFinalizedBlock.Hash) == 0 {
		return
	}

	log.WithFields(log.Fields{"id": e.ID(), "block.Hash": block.Hash}).Info("Finalizing block")
	defer log.WithFields(log.Fields{"id": e.ID(), "block.Hash": block.Hash}).Info("Done Finalized block")

	e.lastFinalizedBlock = block

	select {
	case e.finalizedBlocks <- block.Block:
	default:
	}
}

func (e *DefaultEngine) randHex() []byte {
	bytes := make([]byte, 10)
	e.rand.Read(bytes)
	return bytes
}

func (e *DefaultEngine) shouldPropose(epoch uint32) bool {
	proposer := e.validatorManager.GetProposerForEpoch(epoch)
	return proposer.ID() == e.ID()
}

func (e *DefaultEngine) propose() {
	tip := e.GetTip()

	block := blockchain.Block{}
	block.ChainID = e.chain.ChainID
	block.Hash = e.randHex()
	block.Epoch = e.epoch
	block.ParentHash = tip.Hash

	lastCC := e.highestCCBlock
	proposal := Proposal{Block: block, ProposerID: e.ID()}
	if lastCC.CommitCertificate != nil {
		proposal.CommitCertificate = lastCC.CommitCertificate.Copy()
	}

	log.WithFields(log.Fields{"proposal": proposal, "id": e.ID()}).Info("Making proposal")

	proposalMsg := p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   proposal,
	}
	e.network.Broadcast(proposalMsg)
}
