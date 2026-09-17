package buffer

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/require"
)

func rawChunk(seqID int32, payloadSize int32, isEOF byte, payload []byte) []byte {
	raw := make([]byte, headerSize+len(payload))
	binary.BigEndian.PutUint32(raw[seqIDOffset:seqIDOffset+4], uint32(seqID))
	binary.BigEndian.PutUint32(raw[payloadSizeOffset:payloadSizeOffset+4], uint32(payloadSize))
	raw[isEOFOffset] = isEOF
	copy(raw[payloadOffset:], payload)
	return raw
}

func TestChunkValidityMatchesSanityCheck(t *testing.T) {
	valid := NewChunk([]byte("hello"), 0, 5, markerEOF, 0)
	require.True(t, valid.IsValid())

	invalid := rawChunk(0, -1, markerEOF, nil)
	chunk := &Chunk{bytes: invalid}
	require.False(t, chunk.IsValid())
}

func TestNewChunkFromRawBytesRejectsInvalidHeaders(t *testing.T) {
	_, err := NewChunkFromRawBytes(rawChunk(0, -1, markerEOF, nil))
	require.Error(t, err)

	_, err = NewChunkFromRawBytes(rawChunk(0, maxChunkPayloadSize+1, markerEOF, nil))
	require.Error(t, err)

	_, err = NewChunkFromRawBytes(rawChunk(0, 0, byte(0xff), nil))
	require.Error(t, err)

	_, err = NewChunkFromRawBytes(append(rawChunk(0, 0, markerEOF, nil), byte(0x00)))
	require.Error(t, err)
}
