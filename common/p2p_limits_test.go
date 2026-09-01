package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestP2PReassemblyDeadlineUsesProgressAndAbsoluteBounds(t *testing.T) {
	start := time.Unix(100, 0)
	inactivity := 10 * time.Second

	deadline := P2PReassemblyDeadline(start, start.Add(5*time.Second),
		MaxBlockMessageSize, inactivity)
	require.Equal(t, start.Add(15*time.Second), deadline)

	maxDuration := MaxP2PReassemblyDuration(MaxNormalMessageSize, inactivity)
	deadline = P2PReassemblyDeadline(start, start.Add(maxDuration),
		MaxNormalMessageSize, inactivity)
	require.Equal(t, start.Add(maxDuration), deadline)
}

func TestP2PReassemblyDeadlineDisabledWithoutActiveMessage(t *testing.T) {
	require.True(t, P2PReassemblyDeadline(time.Time{}, time.Now(),
		MaxNormalMessageSize, time.Second).IsZero())
	require.True(t, P2PReassemblyDeadline(time.Now(), time.Now(),
		MaxNormalMessageSize, 0).IsZero())
}
