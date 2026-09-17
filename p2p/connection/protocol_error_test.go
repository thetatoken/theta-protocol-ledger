package connection

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProtocolErrorClassification(t *testing.T) {
	cause := errors.New("bad frame")
	marked := markProtocolError(cause)

	require.True(t, IsProtocolError(marked))
	require.True(t, errors.Is(marked, cause))
	require.Same(t, marked, markProtocolError(marked))
	require.False(t, IsProtocolError(cause))
	require.False(t, IsProtocolError("bad frame"))
	require.Contains(t, fmt.Sprint(marked), "p2p protocol violation")
}
