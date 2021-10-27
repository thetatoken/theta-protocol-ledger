package mempool

import (
	"context"
	"math/big"
	"strconv"
	"sync"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	dp "github.com/thetatoken/theta/dispatcher"
	p2psim "github.com/thetatoken/theta/p2p/simulation"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
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

func TestMempoolReapOrder(t *testing.T) {
	assert := assert.New(t)

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

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

	mempool.InsertTransaction(tx1)
	mempool.InsertTransaction(tx2)
	mempool.InsertTransaction(tx3)
	mempool.InsertTransaction(tx4)
	mempool.InsertTransaction(tx5)
	mempool.InsertTransaction(tx6)
	mempool.InsertTransaction(tx7)
	mempool.InsertTransaction(tx8)
	mempool.InsertTransaction(tx9)
	mempool.InsertTransaction(tx10)

	reapedRawTxs := mempool.Reap(-1)
	assert.Equal(10, len(reapedRawTxs))

	// Transactions from the same address must be ordered by sequence number regardless of gas price,
	// i.e. tx8 > tx5, tx4 > tx1.
	assert.Equal("tx2", string(reapedRawTxs[0][:]))  // gasPrice: 234234, address: A2, seq: 1011
	assert.Equal("tx9", string(reapedRawTxs[1][:]))  // gasPrice: 9273, address: C2, seq: 3021
	assert.Equal("tx10", string(reapedRawTxs[2][:])) // gasPrice: 8281, address: A4, seq: 3022
	assert.Equal("tx7", string(reapedRawTxs[3][:]))  // gasPrice: 5828, address: C1, seq: 3025
	assert.Equal("tx8", string(reapedRawTxs[4][:]))  // gasPrice: 3727, address: B1, seq: 1032
	assert.Equal("tx5", string(reapedRawTxs[5][:]))  // gasPrice: 2392992, address: B1, seq: 1033
	assert.Equal("tx4", string(reapedRawTxs[6][:]))  // gasPrice: 525, address: A1, seq: 1000
	assert.Equal("tx1", string(reapedRawTxs[7][:]))  // gasPrice: 78, address: A1, seq: 1023
	assert.Equal("tx6", string(reapedRawTxs[8][:]))  // gasPrice: 32, address: B2, seq: 3023
	assert.Equal("tx3", string(reapedRawTxs[9][:]))  // gasPrice: 32, address: A3, seq: 2012
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

	assert.Equal(10, mempool.Size())

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

	mempool.Update(committedRawTxs)
	assert.Equal(5, mempool.Size())

	log.Infof("----- Reap all remaining transactions -----")
	reapedRawTxs := mempool.Reap(-1)
	assert.Equal(5, len(reapedRawTxs))
	log.Infof("reapedRawTxs[0]: %v", string(reapedRawTxs[0]))
	log.Infof("reapedRawTxs[1]: %v", string(reapedRawTxs[1]))
	log.Infof("reapedRawTxs[2]: %v", string(reapedRawTxs[2]))
	log.Infof("reapedRawTxs[3]: %v", string(reapedRawTxs[3]))
	log.Infof("reapedRawTxs[4]: %v", string(reapedRawTxs[4]))

	// Tx5 will be taken after Tx8 due to seq number even as it has higher gas price.
	assert.Equal("tx2", string(reapedRawTxs[0][:]))  // gasPrice: 234234, address: A2, seq: 1011
	assert.Equal("tx10", string(reapedRawTxs[1][:])) // gasPrice: 8281, address: A4, seq: 3022
	assert.Equal("tx8", string(reapedRawTxs[2][:]))  // gasPrice: 3727, address: B1, seq: 1032
	assert.Equal("tx5", string(reapedRawTxs[3][:]))  // gasPrice: 2392992, address: B1, seq: 1033
	assert.Equal("tx6", string(reapedRawTxs[4][:]))  // gasPrice: 32, address: B2, seq: 3023
}

func TestMempoolUpdateAndInsert(t *testing.T) {
	assert := assert.New(t)

	tx1 := createTestRawTx("tx1")
	tx2 := createTestRawTx("tx2")
	tx3 := createTestRawTx("tx3")
	tx4 := createTestRawTx("tx4")

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

	assert.Nil(mempool.InsertTransaction(tx1))
	assert.Nil(mempool.InsertTransaction(tx2))
	assert.Nil(mempool.InsertTransaction(tx3))
	assert.Equal(3, mempool.Size())

	committedRawTxs := []common.Bytes{
		common.Bytes("tx1"),
	}

	mempool.Update(committedRawTxs)
	assert.Equal(2, mempool.Size())

	// tx4 and tx1 are from the same address.
	assert.Nil(mempool.InsertTransaction(tx4))
	assert.Equal(3, mempool.Size())
}

func TestMempoolBigBatchUpdateAndReaping(t *testing.T) {
	assert := assert.New(t)

	// Initialize the mempool

	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	mempool, _ := newTestMempool("peer0", p2psimnet)

	committedRawTxs := []common.Bytes{}
	multiplier := 30
	targetRemainder := 3
	for i := 0; i < multiplier*core.MaxNumRegularTxsPerBlock; i++ {
		tx := createTestRawTx("tx_" + strconv.FormatInt(int64(i), 10))
		if i%multiplier == targetRemainder {
			committedRawTxs = append(committedRawTxs, tx)
		}
		err := mempool.InsertTransaction(tx)
		assert.Nil(err)
	}

	numInitCandidateTxs := mempool.Size()
	log.Infof("Number of initial candidate txs: %v", numInitCandidateTxs)
	log.Infof("Number of committed raw txs: %v", len(committedRawTxs))

	// Update the mempool

	t1 := time.Now()

	mempool.Update(committedRawTxs)

	t2 := time.Now()
	elapsedA := t2.Sub(t1)
	log.Infof("Execution time for mempool update: %v", elapsedA)

	elems := mempool.candidateTxs.ElementList()
	for _, elem := range *elems {
		txGroup := elem.(*mempoolTransactionGroup)
		txs := txGroup.txs.ElementList()
		for _, txElem := range *txs {
			mptx := txElem.(*mempoolTransaction)
			txidx, err := strconv.ParseInt(string(mptx.rawTransaction[3:]), 10, 64)
			assert.Nil(err)
			assert.True(txidx%int64(multiplier) != int64(targetRemainder)) // should have been removed by mempool.Update()
		}
	}

	// Reap the mempool

	t3 := time.Now()

	reapedTxs := mempool.Reap(core.MaxNumRegularTxsPerBlock)
	numReapedTxs := len(reapedTxs)
	assert.Equal(core.MaxNumRegularTxsPerBlock, numReapedTxs)

	t4 := time.Now()
	elapsedB := t4.Sub(t3)
	log.Infof("Execution time for mempool reaping: %v", elapsedB)
	log.Infof("Number of txs reaped: %v", numReapedTxs)

	numFinalCandidateTxs := mempool.Size()
	log.Infof("Number of final candidate txs: %v", numFinalCandidateTxs)

	assert.Equal(numInitCandidateTxs-2*core.MaxNumRegularTxsPerBlock, numFinalCandidateTxs)
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
	dispatcher := dp.NewDispatcher(messenger, nil)
	mempool := CreateMempool(dispatcher)
	mempool.SetLedger(newTestLedger())
	txMsgHandler := CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	messenger.Start(ctx)
	return mempool, ctx
}

type TestLedger struct {
	counter               int
	effectiveGasPriceList []uint64
	addressList           []string
	sequenceList          []uint64
}

func newTestLedger() core.Ledger {
	return &TestLedger{
		counter: 0,
		effectiveGasPriceList: []uint64{
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
		addressList: []string{
			"A1",
			"A2",
			"A3",
			"A1",
			"B1",
			"B2",
			"C1",
			"B1",
			"C2",
			"A4",
		},
		sequenceList: []uint64{
			1023,
			1011,
			2012,
			1000,
			1033,
			3023,
			3025,
			1032,
			3021,
			3022,
		},
	}
}

func (tl *TestLedger) ScreenTxUnsafe(rawTx common.Bytes) result.Result {
	_, res := tl.ScreenTx(rawTx)
	return res
}

func (tl *TestLedger) ScreenTx(rawTx common.Bytes) (*core.TxInfo, result.Result) {
	txInfo := &core.TxInfo{
		EffectiveGasPrice: new(big.Int).SetUint64(tl.effectiveGasPriceList[tl.counter]),
		Address:           common.HexToAddress(tl.addressList[tl.counter]),
		Sequence:          tl.sequenceList[tl.counter],
	}
	tl.counter = (tl.counter + 1) % len(tl.effectiveGasPriceList)
	return txInfo, result.OK
}

func (tl *TestLedger) GetCurrentBlock() *core.Block {
	return nil
}

func (tl *TestLedger) ProposeBlockTxs(block *core.Block, shouldIncludeValidatorUpdateTxs bool) (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result) {
	return common.Hash{}, []common.Bytes{}, result.OK
}

func (tl *TestLedger) ApplyBlockTxs(block *core.Block) result.Result {
	return result.OK
}

func (tl *TestLedger) ResetState(height uint64, rootHash common.Hash) result.Result {
	return result.OK
}

func (tl *TestLedger) FinalizeState(height uint64, rootHash common.Hash) result.Result {
	return result.OK
}

func (tl *TestLedger) GetFinalizedValidatorCandidatePool(blockHash common.Hash, isNext bool) (*core.ValidatorCandidatePool, error) {
	return nil, nil
}

func (tl *TestLedger) GetGuardianCandidatePool(blockHash common.Hash) (*core.GuardianCandidatePool, error) {
	return nil, nil
}

func (tl *TestLedger) PruneState(endHeight uint64) error {
	return nil
}

func (tl *TestLedger) ApplyBlockTxsForChainCorrection(block *core.Block) (common.Hash, result.Result) {
	return common.Hash{}, result.Result{}
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
