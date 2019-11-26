package util

import (
	"math/rand"
	"time"
)

// Sample returns a sample of the given entries
func Sample(entries []string, sampleSize int) []string {
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})
	if sampleSize > len(entries) {
		sampleSize = len(entries)
	}
	return entries[0:sampleSize]
}
