package consensus

import (
	"fmt"

	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/store"
)

type StateStub struct {
	HighestCCBlock     common.Bytes
	LastFinalizedBlock common.Bytes
	LastVoteHeight     uint32
	Epoch              uint32
}

const (
	DBStateStubKey       = "cs/ss"
	DBVoteByHeightPrefix = "cs/vbh/"
	DBVoteByBlockPrefix  = "cs/vbb/"
	DBEpochVotesKey      = "cs/ev"
)

type State struct {
	db    store.Store
	chain *blockchain.Chain

	highestCCBlock     *core.ExtendedBlock
	lastFinalizedBlock *core.ExtendedBlock
	tip                *core.ExtendedBlock
	lastVoteHeight     uint32
	epoch              uint32
}

func NewState(db store.Store, chain *blockchain.Chain) *State {
	s := &State{
		db:                 db,
		chain:              chain,
		highestCCBlock:     chain.Root,
		lastFinalizedBlock: chain.Root,
		tip:                chain.Root,
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
		highestCCBlockStr = s.highestCCBlock.Hash.String()
	}

	lastFinalizedBlockStr := "nil"
	if s.lastFinalizedBlock != nil {
		lastFinalizedBlockStr = s.lastFinalizedBlock.Hash.String()
	}

	tipStr := "nil"
	if s.tip != nil {
		tipStr = s.tip.Hash.String()
	}
	return fmt.Sprintf("State{highestCCBlock: %v, lastFinalizedBlock: %v, tip: %v, lastVoteHeight: %d, epoch: %d}",
		highestCCBlockStr, lastFinalizedBlockStr, tipStr, s.lastVoteHeight, s.epoch)
}

func (s *State) commit() error {
	stub := &StateStub{
		LastVoteHeight: s.lastVoteHeight,
		Epoch:          s.epoch,
	}
	if s.highestCCBlock != nil {
		stub.HighestCCBlock = s.highestCCBlock.Hash
	}
	if s.lastFinalizedBlock != nil {
		stub.LastFinalizedBlock = s.lastFinalizedBlock.Hash
	}
	key := []byte(DBStateStubKey)

	return s.db.Put(key, stub)
}

func (s *State) Load() (err error) {
	key := []byte(DBStateStubKey)
	stub := &StateStub{}
	s.db.Get(key, stub)

	s.lastVoteHeight = stub.LastVoteHeight
	s.epoch = stub.Epoch
	if stub.LastFinalizedBlock != nil {
		lastFinalizedBlock, err := s.chain.FindBlock(stub.LastFinalizedBlock)
		if err == nil {
			s.lastFinalizedBlock = lastFinalizedBlock
		}
	}
	if stub.HighestCCBlock != nil {
		highestCCBlock, err := s.chain.FindBlock(stub.HighestCCBlock)
		if err == nil {
			s.highestCCBlock = highestCCBlock
		}
	}
	s.SetTip()
	return
}

func (s *State) GetEpoch() uint32 {
	return s.epoch
}

func (s *State) SetEpoch(epoch uint32) error {
	s.epoch = epoch
	return s.commit()
}

func (s *State) GetLastVoteHeight() uint32 {
	return s.lastVoteHeight
}

func (s *State) SetLastVoteHeight(height uint32) error {
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
	ret, _ := s.chain.FindDeepestDescendant(s.highestCCBlock.Hash)
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
	if err := s.AddVoteByHeight(vote); err != nil {
		return err
	}
	return nil
}

func (s *State) GetVoteSetByHeight(height uint32) (*core.VoteSet, error) {
	key := []byte(fmt.Sprintf("%s:%d", DBVoteByHeightPrefix, height))
	ret := core.NewVoteSet()
	err := s.db.Get(key, ret)
	return ret, err
}

func (s *State) AddVoteByHeight(vote *core.Vote) error {
	if vote.Block == nil {
		return nil
	}
	height := vote.Block.Height
	voteset, err := s.GetVoteSetByHeight(height)
	if err != nil {
		voteset = core.NewVoteSet()
	}
	voteset.AddVote(*vote)
	key := []byte(fmt.Sprintf("%s:%d", DBVoteByHeightPrefix, height))
	return s.db.Put(key, voteset)
}

func (s *State) GetVoteSetByBlock(hash common.Bytes) (*core.VoteSet, error) {
	key := append([]byte(DBVoteByBlockPrefix), hash...)
	ret := core.NewVoteSet()
	err := s.db.Get(key, ret)
	return ret, err
}

func (s *State) AddVoteByBlock(vote *core.Vote) error {
	if vote.Block == nil {
		return nil
	}
	hash := vote.Block.Hash
	voteset, err := s.GetVoteSetByBlock(hash)
	if err != nil {
		voteset = core.NewVoteSet()
	}
	voteset.AddVote(*vote)
	key := append([]byte(DBVoteByBlockPrefix), hash...)
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
