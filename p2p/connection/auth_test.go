package connection

import (
	"crypto/cipher"
	"errors"
	"hash"
	"io"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/golang/snappy"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto/sha3"
	"github.com/thetatoken/theta/rlp"
)

func newFrameTestMAC() hash.Hash {
	return sha3.NewKeccak256()
}

func TestEncryptedFrameRejectsOversizedHeaderBeforeReadingPayload(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	sender := newRLPXFrameRW(left, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})
	receiver := newRLPXFrameRW(right, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})

	headerWritten := make(chan error, 1)
	go func() {
		headbuf := make([]byte, 32)
		putInt24(uint32(maxPacketWireSize+1), headbuf)
		copy(headbuf[3:], zeroHeader)
		sender.enc.XORKeyStream(headbuf[:16], headbuf[:16])
		copy(headbuf[16:], updateMAC(sender.egressMAC, sender.macCipher, headbuf[:16]))
		_, err := left.Write(headbuf)
		headerWritten <- err
	}()

	packet, err := receiver.ReadPacket()
	require.Nil(t, packet)
	require.True(t, errors.Is(err, errPlainMessageTooLarge))
	select {
	case err := <-headerWritten:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("oversized frame header write remained blocked")
	}
}

func TestEncryptedFrameAllowsBoundedSnappyExpansion(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	sender := newRLPXFrameRW(left, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})
	receiver := newRLPXFrameRW(right, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})
	sender.snappy = true
	receiver.snappy = true

	payload := make([]byte, maxPayloadSize)
	_, _ = rand.New(rand.NewSource(24172)).Read(payload)
	packet := &Packet{
		ChannelID: common.ChannelIDBlock,
		Bytes:     payload,
		SeqID:     uint(common.MaxBlockMessageSize/maxPayloadSize - 1),
	}
	raw, err := rlp.EncodeToBytes(packet)
	require.NoError(t, err)
	compressed := snappy.Encode(nil, raw)
	require.Greater(t, len(compressed), maxPacketTotalSize)
	require.LessOrEqual(t, len(compressed), maxPacketWireSize)

	written := make(chan error, 1)
	go func() { written <- sender.WritePacket(packet) }()
	received, err := receiver.ReadPacket()
	require.NoError(t, err)
	require.Equal(t, packet, received)
	require.NoError(t, <-written)
}

func TestEncryptedFrameRejectsOversizedSnappyExpansion(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()
	defer right.Close()

	sender := newRLPXFrameRW(left, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})
	receiver := newRLPXFrameRW(right, secrets{
		AES:        make([]byte, 16),
		MAC:        make([]byte, 16),
		EgressMAC:  newFrameTestMAC(),
		IngressMAC: newFrameTestMAC(),
	})
	receiver.snappy = true

	compressed := snappy.Encode(nil, make([]byte, maxPacketTotalSize+1))
	require.LessOrEqual(t, len(compressed), maxPacketTotalSize)

	frameWritten := make(chan error, 1)
	go func() {
		frameWritten <- writeAuthenticatedTestFrame(sender, compressed)
	}()

	packet, err := receiver.ReadPacket()
	require.Nil(t, packet)
	require.True(t, errors.Is(err, errPlainMessageTooLarge))
	select {
	case err := <-frameWritten:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("compressed frame write remained blocked")
	}
}

func writeAuthenticatedTestFrame(rw *rlpxFrameRW, payload []byte) error {
	fsize := uint32(len(payload))
	headbuf := make([]byte, 32)
	putInt24(fsize, headbuf)
	copy(headbuf[3:], zeroHeader)
	rw.enc.XORKeyStream(headbuf[:16], headbuf[:16])
	copy(headbuf[16:], updateMAC(rw.egressMAC, rw.macCipher, headbuf[:16]))
	if _, err := rw.conn.Write(headbuf); err != nil {
		return err
	}

	rsize := fsize
	if padding := fsize % 16; padding > 0 {
		rsize += 16 - padding
	}
	framebuf := make([]byte, rsize)
	copy(framebuf, payload)
	writer := cipher.StreamWriter{S: rw.enc, W: io.MultiWriter(rw.conn, rw.egressMAC)}
	if _, err := writer.Write(framebuf); err != nil {
		return err
	}

	fmacseed := rw.egressMAC.Sum(nil)
	_, err := rw.conn.Write(updateMAC(rw.egressMAC, rw.macCipher, fmacseed))
	return err
}
