package util

import (
	"math/rand"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

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
