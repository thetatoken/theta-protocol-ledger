package mempool

import (
	"context"
	"math/big"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	dp "github.com/thetatoken/ukulele/dispatcher"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
	"github.com/thetatoken/ukulele/rlp"
)

func TestMempoolBasics(t *testing.T) {
	assert := assert.New(t)

	tx1 := createTestRawTx("tx1")
	tx2 := createTestRawTx("tx2")
	tx3 := createTestRawTx("tx3")
	tx4 := createTestRawTx("tx4")
	tx5 := createTestRawTx("tx5")
	tx6 := createTestRawTx("tx6")
	tx7 := createTestRawTx("tx7")
	tx8 := createTestRawTx("tx8")
	tx9 := createTestRawTx("tx9")
	tx10 := createTestRawTx("tx10")

	log.Infof("tx1 hash: %v", getTransactionHash(tx1))
	log.Infof("tx2 hash: %v", getTransactionHash(tx2))
	log.Infof("tx3 hash: %v", getTransactionHash(tx3))
	log.Infof("tx4 hash: %v", getTransactionHash(tx4))
	log.Infof("tx5 hash: %v", getTransactionHash(tx5))
	log.Infof("tx6 hash: %v", getTransactionHash(tx6))
	log.Infof("tx7 hash: %v", getTransactionHash(tx7))
	log.Infof("tx8 hash: %v", getTransactionHash(tx8))
	log.Infof("tx9 hash: %v", getTransactionHash(tx9))
	log.Infof("tx10 hash: %v", getTransactionHash(tx10))

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

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

	// should order by the designated priority
	assert.Equal("tx5", string(reapedRawTxs[0][:])) // priority: 2392992
	assert.Equal("tx2", string(reapedRawTxs[1][:])) // priority: 234234
	assert.Equal("tx4", string(reapedRawTxs[2][:])) // priority: 525
	assert.Equal(2, mempool.Size())

	// Reap operation
	log.Infof("----- Reap 2 transactions -----")
	reapedRawTxs = mempool.Reap(2)
	assert.Equal(2, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	assert.Equal("tx1", string(reapedRawTxs[0][:])) // priority: 78
	assert.Equal("tx3", string(reapedRawTxs[1][:])) // priority: 32
	assert.Equal(0, mempool.Size())

	// InsertTransaction operation
	log.Infof("----- Insert tx6, tx7, tx8, tx9, tx10 -----")
	assert.Nil(mempool.InsertTransaction(tx6))
	assert.Nil(mempool.InsertTransaction(tx7))
	assert.Nil(mempool.InsertTransaction(tx8))
	assert.Nil(mempool.InsertTransaction(tx9))
	assert.Nil(mempool.InsertTransaction(tx10))

	// Reap operation
	log.Infof("----- Reap 4 transactions -----")
	reapedRawTxs = mempool.Reap(4)
	assert.Equal(4, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	log.Infof("reapedRawTxs[3]: %v", string(reapedRawTxs[3]))

	assert.Equal("tx9", string(reapedRawTxs[0][:]))  // priority: 9273
	assert.Equal("tx10", string(reapedRawTxs[1][:])) // priority: 8281
	assert.Equal("tx7", string(reapedRawTxs[2][:]))  // priority: 5828
	assert.Equal("tx8", string(reapedRawTxs[3][:]))  // priority: 3727

	// Flush operation
	assert.Equal(1, mempool.Size())
	assert.True(mempool.txBookeepper.hasSeen(tx1))
	assert.True(mempool.txBookeepper.hasSeen(tx2))
	assert.True(mempool.txBookeepper.hasSeen(tx3))
	assert.True(mempool.txBookeepper.hasSeen(tx4))
	assert.True(mempool.txBookeepper.hasSeen(tx5))
	assert.True(mempool.txBookeepper.hasSeen(tx6))
	assert.True(mempool.txBookeepper.hasSeen(tx7))
	assert.True(mempool.txBookeepper.hasSeen(tx8))
	assert.True(mempool.txBookeepper.hasSeen(tx9))
	assert.True(mempool.txBookeepper.hasSeen(tx10))

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
	assert.False(mempool.txBookeepper.hasSeen(tx9))
	assert.False(mempool.txBookeepper.hasSeen(tx10))

	// ProcessTransaction operation
	log.Infof("----- Process tx1, tx2, tx3 -----")
	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Equal(3, mempool.Size())

	// Reap operation
	log.Infof("----- Reap all remaining transactions -----")
	reapedRawTxs = mempool.Reap(10) // try to reap 10, but should only get 3
	assert.Equal(3, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))

	assert.Equal("tx2", string(reapedRawTxs[0][:])) // priority: 234234
	assert.Equal("tx1", string(reapedRawTxs[1][:])) // priority: 78
	assert.Equal("tx3", string(reapedRawTxs[2][:])) // priority: 32
}

func TestMempoolUpdate(t *testing.T) {
	assert := assert.New(t)

	tx1 := createTestRawTx("tx1")
	tx2 := createTestRawTx("tx2")
	tx3 := createTestRawTx("tx3")
	tx4 := createTestRawTx("tx4")
	tx5 := createTestRawTx("tx5")
	tx6 := createTestRawTx("tx6")
	tx7 := createTestRawTx("tx7")
	tx8 := createTestRawTx("tx8")
	tx9 := createTestRawTx("tx9")
	tx10 := createTestRawTx("tx10")

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Nil(mempool.InsertTransaction(tx4))
	assert.Nil(mempool.InsertTransaction(tx5))
	assert.Nil(mempool.InsertTransaction(tx6))
	assert.Nil(mempool.InsertTransaction(tx7))
	assert.Nil(mempool.InsertTransaction(tx8))
	assert.Nil(mempool.InsertTransaction(tx9))
	assert.Nil(mempool.InsertTransaction(tx10))

	log.Infof("----- Update committed transactions -----")
	committedRawTxs := []common.Bytes{
		common.Bytes("tx3"),
		common.Bytes("tx9"),
		common.Bytes("tx4"),
		common.Bytes("tx7"),
		common.Bytes("tx1"),
		common.Bytes("tx1"), // intentionally repeated tx
		common.Bytes("tx4"), // intentionally repeated tx
	}

	success := mempool.Update(committedRawTxs)
	assert.True(success)

	log.Infof("----- Reap all remaining transactions -----")
	reapedRawTxs := mempool.Reap(-1)
	assert.Equal(5, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	log.Infof("reapedRawTxs[3]: %v", string(reapedRawTxs[3]))
	log.Infof("reapedRawTxs[4]: %v", string(reapedRawTxs[4]))

	assert.Equal("tx5", string(reapedRawTxs[0][:]))  // priority: 2392992
	assert.Equal("tx2", string(reapedRawTxs[1][:]))  // priority: 234234
	assert.Equal("tx10", string(reapedRawTxs[2][:])) // priority: 8281
	assert.Equal("tx8", string(reapedRawTxs[3][:]))  // priority: 3727
	assert.Equal("tx6", string(reapedRawTxs[4][:]))  // priority: 32
}

func TestMempoolTransactionGossip(t *testing.T) {
	assert := assert.New(t)

	netMsgIntercepter := newTestNetworkMessageInterceptor()
	p2psimnet := p2psim.NewSimnetWithHandler(netMsgIntercepter)

	// Add our node
	mempool, ctx := newTestMempool("peer0", p2psimnet)
	mempool.Start(ctx)

	// Add two peer nodes
	peer1 := p2psimnet.AddEndpoint("peer1")
	peer1.Start(ctx)

	peer2 := p2psimnet.AddEndpoint("peer2")
	peer2.Start(ctx)

	p2psimnet.Start(ctx)

	tx1 := createTestRawTx("tx1")
	tx2 := createTestRawTx("tx2")
	tx3 := createTestRawTx("tx3")

	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Equal(3, mempool.Size())
	log.Infof(">>> Client submitted tx1, tx2, tx3")

	numGossippedTxs := 2 * 3 // 2 peers, each should receive 3 transactions
	for i := 0; i < numGossippedTxs; i++ {
		receivedMsg := <-netMsgIntercepter.ReceivedMessages
		senderID := receivedMsg.PeerID
		dataResponse := receivedMsg.Content.(dp.DataResponse)
		rawTx := string(dataResponse.Payload[:])
		log.Infof("received transaction, sender: %v, rawTx: %v", senderID, rawTx)
		assert.True(rawTx == "tx1" || rawTx == "tx2" || rawTx == "tx3")
	}
}

// --------------- Test Utilities --------------- //

func newTestMempool(peerID string, simnet *p2psim.Simnet) (*Mempool, context.Context) {
	ctx := context.Background()

	messenger := simnet.AddEndpoint(peerID)
	dispatcher := dp.NewDispatcher(messenger)
	mempool := CreateMempool(dispatcher)
	mempool.SetLedger(newTestLedger())
	txMsgHandler := CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	messenger.Start(ctx)
	return mempool, ctx
}

type TestLedger struct {
	counter      int
	priorityList []uint64
}

func newTestLedger() core.Ledger {
	return &TestLedger{
		counter: 0,
		priorityList: []uint64{
			78,      // tx1
			234234,  // tx2
			32,      // tx3
			525,     // tx4
			2392992, // tx5
			32,      // tx6
			5828,    // tx7
			3727,    // tx8
			9273,    // tx9
			8281,    // tx10
		},
	}
}

func (tl *TestLedger) ScreenTx(rawTx common.Bytes) (*big.Int, result.Result) {
	priority := tl.priorityList[tl.counter]
	tl.counter = (tl.counter + 1) % len(tl.priorityList)
	return new(big.Int).SetUint64(priority), result.OK
}

func (tl *TestLedger) ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result) {
	return common.Hash{}, []common.Bytes{}, result.OK
}

func (tl *TestLedger) ApplyBlockTxs(blockRawTxs []common.Bytes, expectedStateRoot common.Hash) result.Result {
	return result.OK
}

func (tl *TestLedger) ResetState(height uint64, rootHash common.Hash) result.Result {
	return result.OK
}

func (tl *TestLedger) FinalizeState(height uint64, rootHash common.Hash) result.Result {
	return result.OK
}

func (tl *TestLedger) Query() {
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
