package connection

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
)

func TestDefaultRecvBuffer(t *testing.T) {
	assert := assert.New(t)
	drb := newTestDefaultRecvBuffer()

	msgBytes := []byte("hello world")
	packet := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
	}

	recvBytes, success := drb.receivePacket(packet)
	assert.True(success)
	assert.Equal(msgBytes, recvBytes)
}

func TestRecvMultipleMessages(t *testing.T) {
	assert := assert.New(t)
	drb := newTestDefaultRecvBuffer()

	msgBytes1 := []byte("hello ")
	packet1 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes1,
		IsEOF:     byte(0x00),
		SeqID:     0,
	}

	msgBytes2 := []byte("world")
	packet2 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes2,
		IsEOF:     byte(0x01),
		SeqID:     1,
	}

	msgBytes3 := []byte("You've got ")
	packet3 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes3,
		IsEOF:     byte(0x00),
		SeqID:     0,
	}

	msgBytes4 := []byte("an ")
	packet4 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes4,
		IsEOF:     byte(0x00),
		SeqID:     1,
	}

	msgBytes5 := []byte("email")
	packet5 := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes5,
		IsEOF:     byte(0x01),
		SeqID:     2,
	}

	// ---------- Check messageA ----------

	messageA := string(msgBytes1) + string(msgBytes2)

	recvBytes1, success := drb.receivePacket(packet1)
	assert.True(success)
	assert.Nil(recvBytes1)

	recvBytes2, success := drb.receivePacket(packet2)
	assert.True(success)
	assert.NotNil(recvBytes2)

	assert.Equal(messageA, string(recvBytes2))

	// ---------- Check messageB ----------

	messageB := string(msgBytes3) + string(msgBytes4) + string(msgBytes5)

	recvBytes3, success := drb.receivePacket(packet3)
	assert.True(success)
	assert.Nil(recvBytes3)

	recvBytes4, success := drb.receivePacket(packet4)
	assert.True(success)
	assert.Nil(recvBytes4)

	recvBytes5, success := drb.receivePacket(packet5)
	assert.True(success)
	assert.NotNil(recvBytes5)

	assert.Equal(messageB, string(recvBytes5))
}

func TestRecvMessageAtDefaultLimit(t *testing.T) {
	assert := assert.New(t)
	drb := newTestDefaultRecvBuffer()

	expectedMsgBytes := []byte{}
	msgBytes := bytes.Repeat([]byte{0x42}, maxPayloadSize)
	packet := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x00),
	}

	var success bool
	var recvBytes []byte
	var i uint
	for ; i < common.MaxNormalMessageSize/maxPayloadSize-1; i++ {
		packet.SeqID = i
		recvBytes, success = drb.receivePacket(packet)
		assert.True(success)
		assert.Nil(recvBytes)

		expectedMsgBytes = append(expectedMsgBytes, packet.Bytes...)
	}

	endPacket := &Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
		SeqID:     i,
	}
	aggregatedBytes, success := drb.receivePacket(endPacket)
	assert.True(success)
	assert.NotNil(aggregatedBytes)
	expectedMsgBytes = append(expectedMsgBytes, endPacket.Bytes...)

	t.Logf("Length of the expectedMsgBytes: %v", len(expectedMsgBytes))
	t.Logf("Length of the aggregatedBytes:  %v", len(aggregatedBytes))

	assert.Equal(common.MaxNormalMessageSize, len(expectedMsgBytes))
	assert.Equal(common.MaxNormalMessageSize, len(aggregatedBytes))
	sameBytes := (bytes.Compare(expectedMsgBytes, aggregatedBytes) == 0)
	assert.True(sameBytes)
}

func TestRecvRejectsMessageAboveLimit(t *testing.T) {
	cfg := getDefaultRecvBufferConfig()
	cfg.maxMessageSize = 300
	drb := createRecvBuffer(cfg)
	msgBytes := bytes.Repeat([]byte{0x42}, 128)

	for seq := uint(0); seq < 2; seq++ {
		recvBytes, success, err := drb.receivePacketWithError(&Packet{
			ChannelID: common.ChannelIDTransaction,
			Bytes:     msgBytes,
			IsEOF:     byte(0x00),
			SeqID:     seq,
		})
		require.NoError(t, err)
		require.True(t, success)
		require.Nil(t, recvBytes)
	}

	recvBytes, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     msgBytes,
		IsEOF:     byte(0x01),
		SeqID:     2,
	})
	require.Error(t, err)
	require.False(t, success)
	require.Nil(t, recvBytes)
	require.Len(t, drb.workspace, 0)
	require.LessOrEqual(t, cap(drb.workspace), cfg.workspaceCapacity)
}

func TestRecvRejectsInvalidEOFMarker(t *testing.T) {
	drb := newTestDefaultRecvBuffer()

	recvBytes, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("invalid"),
		IsEOF:     byte(0x02),
		SeqID:     0,
	})
	require.Error(t, err)
	require.False(t, success)
	require.Nil(t, recvBytes)
	require.Len(t, drb.workspace, 0)
}

func TestRecvSequenceMismatchResetsBuffer(t *testing.T) {
	drb := newTestDefaultRecvBuffer()

	recvBytes, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("partial"),
		IsEOF:     byte(0x00),
		SeqID:     0,
	})
	require.NoError(t, err)
	require.True(t, success)
	require.Nil(t, recvBytes)

	recvBytes, success, err = drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("bad-seq"),
		IsEOF:     byte(0x01),
		SeqID:     2,
	})
	require.Error(t, err)
	require.False(t, success)
	require.Nil(t, recvBytes)
	require.Len(t, drb.workspace, 0)
	require.Equal(t, uint(0), drb.chanSeq)
}

func TestRecvRejectsOversizedPacketBeforeAppend(t *testing.T) {
	drb := newTestDefaultRecvBuffer()

	recvBytes, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     bytes.Repeat([]byte{0x42}, maxPayloadSize+1),
		IsEOF:     byte(0x01),
		SeqID:     0,
	})

	require.Error(t, err)
	require.False(t, success)
	require.Nil(t, recvBytes)
	require.Empty(t, drb.workspace)
}

func TestRecvRejectsExcessPacketCount(t *testing.T) {
	cfg := getDefaultRecvBufferConfig()
	cfg.maxMessageSize = 2 * maxPayloadSize
	drb := createRecvBuffer(cfg)

	maxPacketCount := cfg.maxMessageSize/maxPayloadSize + 2
	for seq := 0; seq < maxPacketCount; seq++ {
		_, success, err := drb.receivePacketWithError(&Packet{
			ChannelID: common.ChannelIDTransaction,
			IsEOF:     byte(0x00),
			SeqID:     uint(seq),
		})
		require.NoError(t, err)
		require.True(t, success)
	}

	_, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		IsEOF:     byte(0x00),
		SeqID:     uint(maxPacketCount),
	})
	require.Error(t, err)
	require.False(t, success)
	require.Empty(t, drb.workspace)
	require.Zero(t, drb.packetCount)
}

func TestRecvRejectsExpiredReassembly(t *testing.T) {
	cfg := getDefaultRecvBufferConfig()
	cfg.messageTimeout = 10 * time.Millisecond
	drb := createRecvBuffer(cfg)

	_, success, err := drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("partial"),
		IsEOF:     byte(0x00),
		SeqID:     0,
	})
	require.NoError(t, err)
	require.True(t, success)
	drb.messageStartedAt = time.Now().Add(-cfg.messageTimeout)
	drb.messageLastProgressAt = drb.messageStartedAt

	_, success, err = drb.receivePacketWithError(&Packet{
		ChannelID: common.ChannelIDTransaction,
		Bytes:     []byte("late"),
		IsEOF:     byte(0x01),
		SeqID:     1,
	})
	require.Error(t, err)
	require.False(t, success)
	require.Empty(t, drb.workspace)
	require.True(t, drb.messageStartedAt.IsZero())
}

func TestRecvAllowsProgressPastInactivityTimeout(t *testing.T) {
	cfg := getDefaultRecvBufferConfig()
	cfg.maxMessageSize = common.MaxNormalMessageSize
	cfg.messageTimeout = 10 * time.Millisecond
	drb := createRecvBuffer(cfg)

	_, success, err := drb.receivePacketWithError(&Packet{
		Bytes: []byte("part-one"), IsEOF: byte(0x00), SeqID: 0,
	})
	require.NoError(t, err)
	require.True(t, success)

	drb.messageStartedAt = time.Now().Add(-2 * cfg.messageTimeout)
	drb.messageLastProgressAt = time.Now()
	message, success, err := drb.receivePacketWithError(&Packet{
		Bytes: []byte("part-two"), IsEOF: byte(0x01), SeqID: 1,
	})
	require.NoError(t, err)
	require.True(t, success)
	require.Equal(t, []byte("part-onepart-two"), message)
}

func TestRecvRejectsTricklePastAbsoluteDeadline(t *testing.T) {
	cfg := getDefaultRecvBufferConfig()
	cfg.maxMessageSize = 64
	cfg.messageTimeout = 10 * time.Millisecond
	drb := createRecvBuffer(cfg)

	_, success, err := drb.receivePacketWithError(&Packet{
		Bytes: []byte("partial"), IsEOF: byte(0x00), SeqID: 0,
	})
	require.NoError(t, err)
	require.True(t, success)
	drb.messageStartedAt = time.Now().Add(-common.MaxP2PReassemblyDuration(
		cfg.maxMessageSize, cfg.messageTimeout))
	drb.messageLastProgressAt = time.Now()

	_, success, err = drb.receivePacketWithError(&Packet{
		Bytes: []byte("still-sending"), IsEOF: byte(0x01), SeqID: 1,
	})
	require.Error(t, err)
	require.False(t, success)
	require.Empty(t, drb.workspace)
}

func TestDefaultChannelUsesChannelMessageLimit(t *testing.T) {
	blockChannel := createDefaultChannel(common.ChannelIDBlock)
	require.Equal(t, common.MaxBlockMessageSize, blockChannel.recvBuf.config.maxMessageSize)

	txChannel := createDefaultChannel(common.ChannelIDTransaction)
	require.Equal(t, common.MaxNormalMessageSize, txChannel.recvBuf.config.maxMessageSize)
}

// --------------- Test Utilities --------------- //

func newTestDefaultRecvBuffer() RecvBuffer {
	defaultConfig := getDefaultRecvBufferConfig()
	recvBuffer := createRecvBuffer(defaultConfig)
	return recvBuffer
}
