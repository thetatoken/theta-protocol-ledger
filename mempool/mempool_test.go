package mempool

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	dp "github.com/thetatoken/ukulele/dispatcher"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
)

func TestMempoolBasics(t *testing.T) {
	assert := assert.New(t)

	tx1 := createTestMempoolTx("tx1")
	tx2 := createTestMempoolTx("tx2")
	tx3 := createTestMempoolTx("tx3")
	tx4 := createTestMempoolTx("tx4")
	tx5 := createTestMempoolTx("tx5")
	tx6 := createTestMempoolTx("tx6")
	tx7 := createTestMempoolTx("tx7")
	tx8 := createTestMempoolTx("tx8")

	log.Infof("tx1 hash: %v", getTransactionHash(tx1))
	log.Infof("tx2 hash: %v", getTransactionHash(tx2))
	log.Infof("tx3 hash: %v", getTransactionHash(tx3))
	log.Infof("tx4 hash: %v", getTransactionHash(tx4))
	log.Infof("tx5 hash: %v", getTransactionHash(tx5))
	log.Infof("tx6 hash: %v", getTransactionHash(tx6))
	log.Infof("tx7 hash: %v", getTransactionHash(tx7))
	log.Infof("tx8 hash: %v", getTransactionHash(tx8))

	mempool := newMempool()

	// ProcessTransaction operation
	log.Infof("----- Process tx1, tx2, tx3 -----")
	assert.Nil(mempool.ProcessTransaction(tx1))
	assert.Nil(mempool.ProcessTransaction(tx2))
	assert.Nil(mempool.ProcessTransaction(tx3))
	assert.Equal(3, mempool.Size())

	log.Infof("----- Process tx4, tx5 -----")
	assert.Nil(mempool.ProcessTransaction(tx4))
	assert.Nil(mempool.ProcessTransaction(tx5))
	assert.Equal(5, mempool.Size())

	// Reap operation
	log.Infof("----- Reap 3 transactions -----")
	reapedRawTxs := mempool.Reap(3)
	assert.Equal(3, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	assert.Equal("tx1", string(reapedRawTxs[0][:]))
	assert.Equal("tx2", string(reapedRawTxs[1][:]))
	assert.Equal("tx3", string(reapedRawTxs[2][:]))

	// Update operation
	log.Infof("----- Update tx1, tx3 -----")
	committedTxs := []common.Bytes{
		[]byte("tx1"),
		[]byte("tx3"),
	}
	assert.True(mempool.Update(committedTxs))
	assert.Equal(3, mempool.Size())

	// Reap operation
	log.Infof("----- Reap 2 transactions -----")
	reapedRawTxs = mempool.Reap(2)
	assert.Equal(2, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	assert.Equal("tx2", string(reapedRawTxs[0][:]))
	assert.Equal("tx4", string(reapedRawTxs[1][:]))

	// ProcessTransaction operation
	log.Infof("----- Process tx6, tx7, tx8 -----")
	assert.Nil(mempool.ProcessTransaction(tx6))
	assert.Nil(mempool.ProcessTransaction(tx7))
	assert.Nil(mempool.ProcessTransaction(tx8))

	// Reap operation
	log.Infof("----- Reap 5 transactions -----")
	reapedRawTxs = mempool.Reap(5)
	assert.Equal(5, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	log.Infof("reapedRawTxs[3]: %v", string(reapedRawTxs[3]))
	log.Infof("reapedRawTxs[4]: %v", string(reapedRawTxs[4]))

	assert.Equal("tx2", string(reapedRawTxs[0][:]))
	assert.Equal("tx4", string(reapedRawTxs[1][:]))
	assert.Equal("tx5", string(reapedRawTxs[2][:]))
	assert.Equal("tx6", string(reapedRawTxs[3][:]))
	assert.Equal("tx7", string(reapedRawTxs[4][:]))

	// Update operation
	log.Infof("----- Update tx2, tx6, tx7 -----")
	committedTxs = []common.Bytes{
		[]byte("tx2"),
		[]byte("tx6"),
		[]byte("tx7"),
	}
	assert.True(mempool.Update(committedTxs))
	assert.Equal(3, mempool.Size())

	// Reap operation
	log.Infof("----- Reap all remaining transactions -----")
	reapedRawTxs = mempool.Reap(10) // try to reap 10, but should only get 3
	assert.Equal(3, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))

	assert.Equal("tx4", string(reapedRawTxs[0][:]))
	assert.Equal("tx5", string(reapedRawTxs[1][:]))
	assert.Equal("tx8", string(reapedRawTxs[2][:]))
}

// --------------- Test Utilities --------------- //

func newMempool() *Mempool {
	simnet := p2psim.NewSimnetWithHandler(nil)
	messenger := simnet.AddEndpoint("messenger1")
	dispatcher := dp.NewDispatcher(messenger)
	mempool := CreateMempool(dispatcher)
	txMsgHandler := CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	return mempool
}
