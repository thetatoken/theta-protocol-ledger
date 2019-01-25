package consensus

import (
	"math/big"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/core"
)

// GetFinalizedBlocks drains the FinalizedBlocks channel and return a slice of block hashes.
func GetFinalizedBlocks(ch chan *core.Block) []string {
	res := []string{}
loop:
	for {
		select {
		case block := <-ch:
			res = append(res, block.Hash().String())
		default:
			break loop
		}
	}
	return res
}

// AssertFinalizedBlocks asserts finalized blocks are as expected.
func AssertFinalizedBlocks(assert *assert.Assertions, expected []string, ch chan *core.Block) {
	assert.Equal(expected, GetFinalizedBlocks(ch))
}

// AssertFinalizedBlocksNotConflicting asserts two chains are not conflicting.
func AssertFinalizedBlocksNotConflicting(assert *assert.Assertions, c1 []string, c2 []string, msg string) {
	length := len(c2)
	if len(c1) < len(c2) {
		length = len(c1)
	}
	for i := 0; i < length; i++ {
		if c1[i] != c2[i] {
			assert.Failf(msg, "Conflicts found: %v, %v", c1, c2)
		}
	}
}

func NewTestValidatorSet(addressStrs []string) *core.ValidatorSet {
	s := core.NewValidatorSet()
	for _, addressStr := range addressStrs {
		stake := new(big.Int).SetUint64(1)
		v := core.NewValidator(addressStr, stake)
		s.AddValidator(v)
	}
	return s
}
