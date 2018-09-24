package mempool

import (
	"context"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	dp "github.com/thetatoken/ukulele/dispatcher"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/rlp"
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

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool := newTestMempool("peer0", p2psimnet)

	// ProcessTransaction operation
	log.Infof("----- Process tx1, tx2, tx3 -----")
	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Equal(3, mempool.Size())

	log.Infof("----- Process tx4, tx5 -----")
	assert.Nil(mempool.InsertTransaction(tx4))
	assert.Nil(mempool.InsertTransaction(tx5))
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

	// InsertTransaction operation
	log.Infof("----- Insert tx6, tx7, tx8 -----")
	assert.Nil(mempool.InsertTransaction(tx6))
	assert.Nil(mempool.InsertTransaction(tx7))
	assert.Nil(mempool.InsertTransaction(tx8))

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

	// Flush operation
	assert.Equal(3, mempool.Size())
	assert.True(mempool.txBookeepper.hasSeen(tx1))
	assert.True(mempool.txBookeepper.hasSeen(tx2))
	assert.True(mempool.txBookeepper.hasSeen(tx3))
	assert.True(mempool.txBookeepper.hasSeen(tx4))
	assert.True(mempool.txBookeepper.hasSeen(tx5))
	assert.True(mempool.txBookeepper.hasSeen(tx6))
	assert.True(mempool.txBookeepper.hasSeen(tx7))
	assert.True(mempool.txBookeepper.hasSeen(tx8))

	mempool.Flush()

	assert.Equal(0, mempool.Size())
	assert.False(mempool.txBookeepper.hasSeen(tx1))
	assert.False(mempool.txBookeepper.hasSeen(tx2))
	assert.False(mempool.txBookeepper.hasSeen(tx3))
	assert.False(mempool.txBookeepper.hasSeen(tx4))
	assert.False(mempool.txBookeepper.hasSeen(tx5))
	assert.False(mempool.txBookeepper.hasSeen(tx6))
	assert.False(mempool.txBookeepper.hasSeen(tx7))
	assert.False(mempool.txBookeepper.hasSeen(tx8))
}

func TestMempoolTransactionGossip(t *testing.T) {
	assert := assert.New(t)

	netMsgIntercepter := newTestNetworkMessageInterceptor()
	p2psimnet := p2psim.NewSimnetWithHandler(netMsgIntercepter)

	// Add our node
	mempool := newTestMempool("peer0", p2psimnet)
	mempool.Start()

	// Add two peer nodes
	peer1 := p2psimnet.AddEndpoint("peer1")
	peer1.Start()

	peer2 := p2psimnet.AddEndpoint("peer2")
	peer2.Start()

	p2psimnet.Start(context.Background())

	tx1 := createTestMempoolTx("tx1")
	tx2 := createTestMempoolTx("tx2")
	tx3 := createTestMempoolTx("tx3")

	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Equal(3, mempool.Size())
	log.Infof(">>> Client submitted tx1, tx2, tx3")

	numGossippedTxs := 2 * 3 // 2 peers, each should receive 3 transactions
	for i := 0; i < numGossippedTxs; i++ {
		receivedMsg := <-netMsgIntercepter.ReceivedMessages
		receiverPeerID := receivedMsg.PeerID
		dataResponse := receivedMsg.Content.(dp.DataResponse)
		rawTx := string(dataResponse.Payload[:])
		log.Infof("received transaction, receiver: %v, rawTx: %v", receiverPeerID, rawTx)

		assert.True(receiverPeerID == "peer1" || receiverPeerID == "peer2")
		assert.True(rawTx == "tx1" || rawTx == "tx2" || rawTx == "tx3")
	}
}

// --------------- Test Utilities --------------- //

func newTestMempool(peerID string, simnet *p2psim.Simnet) *Mempool {
	messenger := simnet.AddEndpoint(peerID)
	dispatcher := dp.NewDispatcher(messenger)
	mempool := CreateMempool(dispatcher)
	txMsgHandler := CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	messenger.Start()
	return mempool
}

type TestNetworkMessageInterceptor struct {
	lock             *sync.Mutex
	ReceivedMessages chan p2ptypes.Message
}

func newTestNetworkMessageInterceptor() *TestNetworkMessageInterceptor {
	return &TestNetworkMessageInterceptor{
		lock:             &sync.Mutex{},
		ReceivedMessages: make(chan p2ptypes.Message),
	}
}

func (tnmi *TestNetworkMessageInterceptor) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDTransaction,
	}
}

func (tnmi *TestNetworkMessageInterceptor) EncodeMessage(message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

func (tnmi *TestNetworkMessageInterceptor) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		PeerID:    peerID,
		ChannelID: channelID,
		Content:   rawMessageBytes,
	}
	return message, nil
}

func (tnmi *TestNetworkMessageInterceptor) HandleMessage(msg p2ptypes.Message) error {
	tnmi.lock.Lock()
	defer tnmi.lock.Unlock()
	tnmi.ReceivedMessages <- msg
	return nil
}
