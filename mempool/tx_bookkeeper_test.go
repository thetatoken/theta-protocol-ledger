package mempool

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"

	"github.com/stretchr/testify/assert"
)

func TestTxBookkeeper(t *testing.T) {
	assert := assert.New(t)

	tx1 := createTestRawTx("1")
	tx2 := createTestRawTx("2")
	tx3 := createTestRawTx("3")
	tx4 := createTestRawTx("4")
	tx5 := createTestRawTx("5")

	log.Infof("tx1 hash: %v", getTransactionHash(tx1))
	log.Infof("tx2 hash: %v", getTransactionHash(tx2))
	log.Infof("tx3 hash: %v", getTransactionHash(tx3))
	log.Infof("tx4 hash: %v", getTransactionHash(tx4))
	log.Infof("tx5 hash: %v", getTransactionHash(tx5))

	maxNumTxs := uint(3)
	txb := createTransactionBookkeeper(maxNumTxs)
	assert.False(txb.hasSeen(tx1))
	assert.False(txb.hasSeen(tx3))
	assert.False(txb.hasSeen(tx5))

	assert.True(txb.record(tx1))
	assert.True(txb.hasSeen(tx1))

	assert.True(txb.record(tx2))
	assert.True(txb.hasSeen(tx2))

	assert.True(txb.record(tx3))
	assert.True(txb.hasSeen(tx3))

	assert.True(txb.record(tx4))
	assert.True(txb.hasSeen(tx4))
	assert.False(txb.hasSeen(tx1)) // tx1 should have been purged

	assert.True(txb.record(tx5))
	assert.True(txb.hasSeen(tx5))
	assert.False(txb.hasSeen(tx2)) // tx2 should have been purged

	txb.remove(tx4)
	assert.False(txb.hasSeen(tx4))

	txb.remove(tx5)
	assert.False(txb.hasSeen(tx5))
}

// --------------- Test Utilities --------------- //

func createTestRawTx(rawTxStr string) common.Bytes {
	rawTx := common.Bytes(rawTxStr)
	return rawTx
}
