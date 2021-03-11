package consensus

import (
	"container/list"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	tcore "github.com/thetatoken/theta/core"
	tcrypto "github.com/thetatoken/theta/crypto"
)

const DefaultMaxNumVotesCached = uint(200000)

const maxVoteLife = 5 * time.Minute

//
// EENVoteBookkeeper keeps tracks of recently seen elite edge node votes
//
type EENVoteBookkeeper struct {
	mutex *sync.Mutex

	voteMap  map[string]*EENVoteRecord // map: vote hash -> record
	voteList list.List                 // FIFO list of vote hashes

	maxNumVotes uint
}

type EENVoteRecord struct {
	Hash      string
	CreatedAt time.Time
}

func (r *EENVoteRecord) IsOutdated() bool {
	return time.Since(r.CreatedAt) > maxVoteLife
}

type TxStatus int

const (
	TxStatusPending TxStatus = iota
	TxStatusAbandoned
)

func CreateEENVoteBookkeeper(maxNumTxs uint) *EENVoteBookkeeper {
	return &EENVoteBookkeeper{
		mutex:       &sync.Mutex{},
		voteMap:     make(map[string]*EENVoteRecord),
		maxNumVotes: maxNumTxs,
	}
}

func (vb *EENVoteBookkeeper) reset() {
	vb.mutex.Lock()
	defer vb.mutex.Unlock()
	vb.voteMap = make(map[string]*EENVoteRecord)
	vb.voteList.Init()
}

func (vb *EENVoteBookkeeper) HasSeen(vote *tcore.EENVote) bool {
	vb.mutex.Lock()
	defer vb.mutex.Unlock()

	// Remove outdated Tx records
	vb.removeOutdatedVotesUnsafe()

	voteHash := getVoteHash(vote)
	_, exists := vb.voteMap[voteHash]
	return exists
}

func (vb *EENVoteBookkeeper) removeOutdatedVotesUnsafe() {
	// Loop and remove all outdated Tx records
	for {
		el := vb.voteList.Front()
		if el == nil {
			return
		}
		voteRecord := el.Value.(*EENVoteRecord)
		if !voteRecord.IsOutdated() {
			return
		}

		if _, exists := vb.voteMap[voteRecord.Hash]; exists {
			delete(vb.voteMap, voteRecord.Hash)
		}
		vb.voteList.Remove(el)
	}
}

func (vb *EENVoteBookkeeper) Record(vote *tcore.EENVote) bool {
	vb.mutex.Lock()
	defer vb.mutex.Unlock()
	voteHash := getVoteHash(vote)

	// Remove outdated vote records
	vb.removeOutdatedVotesUnsafe()

	if _, exists := vb.voteMap[voteHash]; exists {
		return false
	}

	if uint(vb.voteList.Len()) >= vb.maxNumVotes { // remove the oldest votes
		popped := vb.voteList.Front()
		poppedVoteHash := popped.Value.(*EENVoteRecord).Hash
		delete(vb.voteMap, poppedVoteHash)
		vb.voteList.Remove(popped)
	}

	record := &EENVoteRecord{
		Hash:      voteHash,
		CreatedAt: time.Now(),
	}
	vb.voteMap[voteHash] = record

	vb.voteList.PushBack(record)

	return true
}

func getVoteHash(vote *tcore.EENVote) string {
	voteStr := fmt.Sprintf("%v:%v", vote.Address, vote.Block)
	txhash := tcrypto.Keccak256Hash([]byte(voteStr))
	txhashStr := hex.EncodeToString(txhash[:])
	return txhashStr
}
