package types

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlashIntentJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	intent := SlashIntent{
		ReserveSequence: math.MaxUint64,
	}

	s, err := json.Marshal(intent)
	require.Nil(err)

	var d SlashIntent
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.ReserveSequence)
}

func TestOverspendingProofJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	p := OverspendingProof{
		ReserveSequence: math.MaxUint64,
	}

	s, err := json.Marshal(p)
	require.Nil(err)

	var d OverspendingProof
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.ReserveSequence)
}
