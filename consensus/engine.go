package consensus

import (
	"bytes"
	"context"
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
	highestCCBlock     *blockchain.ExtendedBlock
	lastFinalizedBlock *blockchain.ExtendedBlock
	tip                *blockchain.ExtendedBlock
	lastVoteHeight     uint32
	voteLog            map[uint32]blockchain.Vote     // level -> vote
	collectedVotes     map[string]*blockchain.VoteSet // block hash -> votes
	epochManager       *EpochManager
	epoch              uint32
	validatorManager   ValidatorManager

	// Strategies
	proposerStrategy ProposerStrategy
	replicaStrategy  ReplicaStrategy
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

		highestCCBlock:     chain.Root,
		lastFinalizedBlock: chain.Root,
		tip:                chain.Root,
		voteLog:            make(map[uint32]blockchain.Vote),
		collectedVotes:     make(map[string]*blockchain.VoteSet),
		validatorManager:   NewRotatingValidatorManager(validators),
		epochManager:       NewEpochManager(),
		epoch:              0,

		proposerStrategy: &DefaultProposerStrategy{},
		replicaStrategy:  &DefaultReplicaStrategy{},
	}
	e.proposerStrategy.Init(e)
	e.replicaStrategy.Init(e)
	e.epochManager.Init(e)
	network.AddMessageHandler(e)
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

// SetProposerStrategy allows us to change proposerStrategy.
func (e *DefaultEngine) SetProposerStrategy(s ProposerStrategy) {
	s.Init(e)
	e.proposerStrategy = s
}

// SetReplicaStrategy allows us to change replicaStrategy.
func (e *DefaultEngine) SetReplicaStrategy(s ReplicaStrategy) {
	s.Init(e)
	e.replicaStrategy = s
}

// Start is the main event loop.
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
		case msg, ok := <-e.incoming:
			if !ok {
				continue
			}
			switch m := msg.(type) {
			case Proposal:
				e.handleProposal(m)
			case blockchain.Vote:
				e.handleVote(m)
			default:
				log.Errorf("Unknown message type: %v", m)
			}
		case newEpoch := <-e.epochManager.C:
			newEpoch = e.epochManager.GetEpoch()
			e.enterNewEpoch(newEpoch)
		}
	}
}

func (e *DefaultEngine) enterNewEpoch(newEpoch uint32) {
	e.epoch = newEpoch
	e.proposerStrategy.EnterNewEpoch(newEpoch)
	e.replicaStrategy.EnterNewEpoch(newEpoch)
}

// GetChannelIDs implements the p2p.MessageHandler interface.
func (e *DefaultEngine) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDVote,
	}
}

// ParseMessage implements p2p.MessageHandler interface.
func (e *DefaultEngine) ParseMessage(channelID common.ChannelIDEnum,
	rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	// To be implemented..
	message := p2ptypes.Message{
		ChannelID: channelID,
	}
	return message, nil
}

// HandleMessage implements p2p.MessageHandler interface.
func (e *DefaultEngine) HandleMessage(peerID string, msg p2ptypes.Message) {
	e.incoming <- msg.Content
}

func (e *DefaultEngine) handleProposal(proposal Proposal) {
	e.replicaStrategy.HandleProposal(proposal)
}

func (e *DefaultEngine) handleVote(vote blockchain.Vote) {
	e.proposerStrategy.HandleVote(vote)
}

// setTip sets the block to extended from by next proposal. Currently we use the highest block among highestCCBlock's
// descendants as the fork-choice rule.
func (e *DefaultEngine) setTip() *blockchain.ExtendedBlock {
	ret, _ := e.highestCCBlock.FindDeepestDescendant()
	e.tip = ret
	return ret
}

// getTip return the block to be extended from.
func (e *DefaultEngine) getTip() *blockchain.ExtendedBlock {
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

	if ccBlock.Parent.CommitCertificate != nil {
		e.finalizeBlock(ccBlock.Parent)
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
