package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
)

func TestValidateSplitBasisPointBeforeFork(t *testing.T) {
	assert := assert.New(t)
	height := common.HeightRemoveRewardSplitCap - 1

	assert.True(validateSplitBasisPoint(0, height).IsOK())
	assert.True(validateSplitBasisPoint(1000, height).IsOK())
	assert.True(validateSplitBasisPoint(1001, height).IsError())
	assert.True(validateSplitBasisPoint(10000, height).IsError())
}

func TestValidateSplitBasisPointAfterFork(t *testing.T) {
	assert := assert.New(t)
	height := common.HeightRemoveRewardSplitCap

	assert.True(validateSplitBasisPoint(0, height).IsOK())
	assert.True(validateSplitBasisPoint(1000, height).IsOK())
	assert.True(validateSplitBasisPoint(1001, height).IsOK())
	assert.True(validateSplitBasisPoint(10000, height).IsOK())
	assert.True(validateSplitBasisPoint(10001, height).IsError())
}
