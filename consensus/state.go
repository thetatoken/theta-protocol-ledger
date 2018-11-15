package consensus

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/store"
)

type StateStub struct {
	Root               common.Hash
	HighestCCBlock     common.Hash
	LastFinalizedBlock common.Hash
	LastVoteHeight     uint64
	Epoch              uint64
}

const (
	DBStateStubKey      = "cs/ss"
	DBVoteByBlockPrefix = "cs/vbb/"
	DBEpochVotesKey     = "cs/ev"
)

type State struct {
	db    store.Store
	chain *blockchain.Chain

	highestCCBlock     *core.ExtendedBlock
	lastFinalizedBlock *core.ExtendedBlock
	tip                *core.ExtendedBlock
	lastVoteHeight     uint64
	epoch              uint64
}

func NewState(db store.Store, chain *blockchain.Chain) *State {
	s := &State{
		db:                 db,
		chain:              chain,
		highestCCBlock:     chain.Root,
		lastFinalizedBlock: chain.Root,
		tip:                chain.Root,
		epoch:              chain.Root.Epoch,
	}
	err := s.Load()
	if err != nil {
		panic(err)
	}
	return s
}

func (s *State) String() string {
	highestCCBlockStr := "nil"
	if s.highestCCBlock != nil {
		highestCCBlockStr = s.highestCCBlock.Hash().Hex()
	}

	lastFinalizedBlockStr := "nil"
	if s.lastFinalizedBlock != nil {
		lastFinalizedBlockStr = s.lastFinalizedBlock.Hash().Hex()
	}

	tipStr := "nil"
	if s.tip != nil {
		tipStr = s.tip.Hash().Hex()
	}
	return fmt.Sprintf("State{highestCCBlock: %v, lastFinalizedBlock: %v, tip: %v, lastVoteHeight: %d, epoch: %d}",
		highestCCBlockStr, lastFinalizedBlockStr, tipStr, s.lastVoteHeight, s.epoch)
}

func (s *State) commit() error {
	stub := &StateStub{
		LastVoteHeight: s.lastVoteHeight,
		Epoch:          s.epoch,
		Root:           s.chain.Root.Hash(),
	}
	if s.highestCCBlock != nil {
		stub.HighestCCBlock = s.highestCCBlock.Hash()
	}
	if s.lastFinalizedBlock != nil {
		stub.LastFinalizedBlock = s.lastFinalizedBlock.Hash()
	}
	key := []byte(DBStateStubKey)

	return s.db.Put(key, stub)
}

func (s *State) Load() (err error) {
	key := []byte(DBStateStubKey)
	stub := &StateStub{}
	s.db.Get(key, stub)

	if stub.Root != s.chain.Root.Hash() {
		logger.WithFields(log.Fields{
			"stub.Root":  stub.Root,
			"chain.Root": s.chain.Root.Hash,
		}).Warn("Ignoring previous consensus state since it is on a different root")
		s.SetTip()
		return
	}

	s.lastVoteHeight = stub.LastVoteHeight
	s.epoch = stub.Epoch
	if !stub.LastFinalizedBlock.IsEmpty() {
		lastFinalizedBlock, err := s.chain.FindBlock(stub.LastFinalizedBlock)
		if err == nil {
			s.lastFinalizedBlock = lastFinalizedBlock
		}
	}
	if !stub.HighestCCBlock.IsEmpty() {
		highestCCBlock, err := s.chain.FindBlock(stub.HighestCCBlock)
		if err == nil {
			s.highestCCBlock = highestCCBlock
		}
	}
	s.SetTip()
	return
}

func (s *State) GetEpoch() uint64 {
	return s.epoch
}

func (s *State) SetEpoch(epoch uint64) error {
	s.epoch = epoch
	return s.commit()
}

func (s *State) GetLastVoteHeight() uint64 {
	return s.lastVoteHeight
}

func (s *State) SetLastVoteHeight(height uint64) error {
	s.lastVoteHeight = height
	return s.commit()
}

func (s *State) GetHighestCCBlock() *core.ExtendedBlock {
	return s.highestCCBlock
}

func (s *State) SetHighestCCBlock(block *core.ExtendedBlock) error {
	s.highestCCBlock = block
	return s.commit()
}

func (s *State) GetLastFinalizedBlock() *core.ExtendedBlock {
	return s.lastFinalizedBlock
}

func (s *State) SetLastFinalizedBlock(block *core.ExtendedBlock) error {
	s.lastFinalizedBlock = block
	return s.commit()
}

// SetTip sets the block to extended from by next proposal. Currently we use the highest block among highestCCBlock's
// descendants as the fork-choice rule.
func (s *State) SetTip() *core.ExtendedBlock {
	ret, _ := s.chain.FindDeepestDescendant(s.highestCCBlock.Hash())
	s.tip = ret
	return ret
}

// GetTip return the block to be extended from.
func (s *State) GetTip() *core.ExtendedBlock {
	return s.tip
}

func (s *State) AddVote(vote *core.Vote) error {
	if err := s.AddEpochVote(vote); err != nil {
		return err
	}
	if err := s.AddVoteByBlock(vote); err != nil {
		return err
	}
	return nil
}

func (s *State) GetVoteSetByBlock(hash common.Hash) (*core.VoteSet, error) {
	key := append([]byte(DBVoteByBlockPrefix), hash[:]...)
	ret := core.NewVoteSet()
	err := s.db.Get(key, ret)
	return ret, err
}

func (s *State) AddVoteByBlock(vote *core.Vote) error {
	if vote.Block.IsEmpty() {
		return nil
	}
	voteset, err := s.GetVoteSetByBlock(vote.Block)
	if err != nil {
		voteset = core.NewVoteSet()
	}
	voteset.AddVote(*vote)
	key := append([]byte(DBVoteByBlockPrefix), vote.Block[:]...)
	return s.db.Put(key, voteset)
}

func (s *State) GetEpochVotes() (*core.VoteSet, error) {
	key := []byte(DBEpochVotesKey)
	ret := core.NewVoteSet()
	err := s.db.Get(key, ret)
	return ret, err
}

func (s *State) AddEpochVote(vote *core.Vote) error {
	voteset, err := s.GetEpochVotes()
	if err != nil {
		voteset = core.NewVoteSet()
	}
	voteset.AddVote(*vote)
	key := []byte(DBEpochVotesKey)
	return s.db.Put(key, voteset)
}
