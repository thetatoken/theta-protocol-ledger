package types

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitRuleJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	splitRule := &SplitRule{
		EndBlockHeight: math.MaxUint64,
	}

	s, err := json.Marshal(splitRule)
	require.Nil(err)

	var d SplitRule
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.EndBlockHeight)
}
