// +build unit

package netsync

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
)

func intToHash(i int) string {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(buf, int64(i))
	return hex.EncodeToString(buf)
}

func TestOrphanBlockPool(t *testing.T) {
	assert := assert.New(t)

	bp := NewOrphanBlockPool()

	// Should not panic when operates on empty pool.
	bp.RemoveOldest()
	bp.Remove(blockchain.CreateTestBlock("a1", "a0"))
	assert.False(bp.Contains(blockchain.CreateTestBlock("a1", "a0")))
	bp.TryGetNextBlock(blockchain.ParseHex("aabb"))

	bp.Add(blockchain.CreateTestBlock("a1", "a0"))
	// Adding duplidate block should have not effect.
	bp.Add(blockchain.CreateTestBlock("a1", "a0"))
	bp.Add(blockchain.CreateTestBlock("a2", "a1"))
	a1 := bp.TryGetNextBlock(blockchain.ParseHex("a0"))
	assert.NotNil(a1)
	assert.Equal("A1", a1.Hash.String())
	a1 = bp.TryGetNextBlock(blockchain.ParseHex("a0"))
	assert.Nil(a1, "block a1 should have been removed from pool")
	assert.Equal(1, bp.blocks.Len())
	assert.False(bp.Contains(blockchain.CreateTestBlock("a1", "a0")))
	assert.True(bp.Contains(blockchain.CreateTestBlock("a2", "a1")))

	// Verify that pool is capped.
	bp = NewOrphanBlockPool()
	for i := 0; i < maxOrphanBlockPoolSize; i++ {
		block := blockchain.CreateTestBlock(intToHash(i), intToHash(i+1))
		bp.Add(block)
	}
	firstBlock := blockchain.CreateTestBlock(intToHash(0), intToHash(1))
	assert.True(bp.Contains(firstBlock))
	assert.Equal(maxOrphanBlockPoolSize, bp.blocks.Len())
	bp.Add(blockchain.CreateTestBlock(intToHash(maxOrphanBlockPoolSize), intToHash(maxOrphanBlockPoolSize+1)))
	assert.False(bp.Contains(firstBlock), "the oldest block should have been evicted from pool")
	assert.Equal(maxOrphanBlockPoolSize, bp.blocks.Len())
}

func TestOrphanCCPool(t *testing.T) {
	assert := assert.New(t)

	cp := NewOrphanCCPool()
	cc1 := &blockchain.CommitCertificate{BlockHash: common.Bytes("a0")}
	cc2 := &blockchain.CommitCertificate{BlockHash: common.Bytes("a1")}

	// Should not panic when operates on empty pool.
	cp.RemoveOldest()
	cp.Remove(cc1)
	assert.False(cp.Contains(cc1))
	cp.TryGetCCByBlockHash(common.Bytes("a0"))

	cp.Add(cc1)
	// Adding duplidate cc should have not effect.
	cp.Add(cc1)
	cp.Add(cc2)
	a1 := cp.TryGetCCByBlockHash(common.Bytes("a0"))
	assert.NotNil(a1)
	assert.Equal(0, bytes.Compare(common.Bytes("a0"), a1.BlockHash))
	a1 = cp.TryGetCCByBlockHash(common.Bytes("a0"))
	assert.Nil(a1, "block a1 should have been removed from pool")
	assert.Equal(1, cp.ccs.Len())
	assert.False(cp.Contains(cc1))
	assert.True(cp.Contains(cc2))

	// Verify that pool is capped.
	cp = NewOrphanCCPool()
	for i := 0; i < maxOrphanCCPoolSize; i++ {
		cc := &blockchain.CommitCertificate{BlockHash: common.Bytes(fmt.Sprintf("%x", i))}
		cp.Add(cc)
	}
	firstCC := &blockchain.CommitCertificate{BlockHash: common.Bytes(fmt.Sprintf("%x", 0))}
	assert.True(cp.Contains(firstCC))
	assert.Equal(maxOrphanCCPoolSize, cp.ccs.Len())
	cp.Add(&blockchain.CommitCertificate{BlockHash: common.Bytes(fmt.Sprintf("%x", maxOrphanCCPoolSize))})
	assert.False(cp.Contains(firstCC), "the oldest CC should have been evicted from pool")
	assert.Equal(maxOrphanCCPoolSize, cp.ccs.Len())
}
