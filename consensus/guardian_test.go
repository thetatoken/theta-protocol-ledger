package consensus

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMaxMultiplies(t *testing.T) {
	require := require.New(t)
	g := NewGuardianEngine(nil, nil)
	maxNeighbors := uint32(1 << maxLogNeighbors)
	require.Equal(maxNeighbors, g.maxMultiply(1))
	require.Equal(maxNeighbors*maxNeighbors, g.maxMultiply(2))

}
