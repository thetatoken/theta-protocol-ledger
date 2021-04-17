package util

import (
	crand "crypto/rand"
	"encoding/binary"
	"log"
	"math/big"
	"math/rand"
	"sort"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
)

var TfuelRewardN = 400 // Reward receiver sampling params

// Sample returns a sample of the given entries
func Sample(entries []string, sampleSize int) []string {
	if sampleSize < 0 {
		return []string{}
	}
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
	if sampleSize > len(entries) {
		sampleSize = len(entries)
	}
	return entries[0:sampleSize]
}

// Shuffle shuffles the given entries
func Shuffle(entries []string) []string {
	numEntries := len(entries)
	return Sample(entries, numEntries)
}

// StakeSample samples stakers and returns a mapping from sampled source address to its weight.
func StakeSample(height uint64, hash common.Hash, stakeSourceList []common.Address, stakeSourceMap map[common.Address]*big.Int, totalStake *big.Int) (sourceWeightMap map[common.Address]int) {
	ret := make(map[common.Address]int)

	samples := make([]*big.Int, TfuelRewardN)
	for i := 0; i < TfuelRewardN; i++ {
		// Set random seed to (block_height||sampling_index||checkpoint_hash)
		seed := make([]byte, 2*binary.MaxVarintLen64+common.HashLength)
		binary.PutUvarint(seed[:], height)
		binary.PutUvarint(seed[binary.MaxVarintLen64:], uint64(i))
		copy(seed[2*binary.MaxVarintLen64:], hash[:])

		var err error
		samples[i], err = crand.Int(NewHashRand(seed), totalStake)
		if err != nil {
			// Should not reach here
			log.Panic(err)
		}
	}

	sort.Sort(BigIntSort(samples))

	curr := 0
	currSum := big.NewInt(0)

	for i := 0; i < len(stakeSourceList); i++ {
		stakeSourceAddr := stakeSourceList[i]
		stakeAmountSum := stakeSourceMap[stakeSourceAddr]

		if curr >= TfuelRewardN {
			break
		}

		count := 0
		lower := currSum
		upper := new(big.Int).Add(currSum, stakeAmountSum)
		for curr < TfuelRewardN && samples[curr].Cmp(lower) >= 0 && samples[curr].Cmp(upper) < 0 {
			count++
			curr++
		}
		currSum = upper

		if count > 0 {
			if _, ok := ret[stakeSourceAddr]; !ok {
				ret[stakeSourceAddr] = 0
			}
			ret[stakeSourceAddr] += count
		}
	}
	return ret
}

// SampledEENVotesVector samples the EliteEdgeNodePool and returns mapping from holders with sampled stakes to their indices in the vector.
func SampledEENVotesVector(eenp *core.EliteEdgeNodePool, height uint64, block common.Hash) (holderIndexMap map[common.Address]int) {
	stakeSourceMap := map[common.Address]*big.Int{}
	stakeSourceList := []common.Address{}
	totalStake := big.NewInt(0)

	for _, e := range eenp.SortedEliteEdgeNodes {
		stakes := e.Stakes
		for _, stake := range stakes {
			if stake.Withdrawn {
				continue
			}
			stakeAmount := stake.Amount
			stakeSource := stake.Source
			if stakeAmountSum, exists := stakeSourceMap[stakeSource]; exists {
				stakeAmountSum.Add(stakeAmountSum, stakeAmount)
			} else {
				stakeSourceMap[stakeSource] = stakeAmount
				stakeSourceList = append(stakeSourceList, stakeSource)
			}
		}

		totalStake.Add(totalStake, e.TotalStake())
	}

	sourceWeightMap := StakeSample(height, block, stakeSourceList, stakeSourceMap, totalStake)
	holderIndexMap = make(map[common.Address]int)
	i := 0
	for _, en := range eenp.SortedEliteEdgeNodes {
		stakes := en.Stakes
		for _, stake := range stakes {
			if _, ok := sourceWeightMap[stake.Source]; ok {
				holderIndexMap[en.Holder] = i
				i++
				break
			}
		}
	}

	return holderIndexMap
}

// HashRand generate infinite number of random bytes by repeatedly hashing the seed
type HashRand struct {
	remaining []byte
	curr      common.Hash
}

func NewHashRand(seed []byte) *HashRand {
	return &HashRand{
		remaining: []byte{},
		curr:      crypto.Keccak256Hash(seed),
	}
}

func (r *HashRand) Read(buf []byte) (int, error) {
	if len(r.remaining) != 0 {
		n := copy(buf, r.remaining)
		r.remaining = r.remaining[n:]
		return n, nil
	}
	r.curr = crypto.Keccak256Hash(r.curr[:])
	n := copy(buf, r.curr[:])
	r.remaining = r.curr[n:]
	return n, nil
}

type BigIntSort []*big.Int

func (s BigIntSort) Len() int           { return len(s) }
func (s BigIntSort) Less(i, j int) bool { return s[i].Cmp(s[j]) < 0 }
func (s BigIntSort) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
