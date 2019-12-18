package netsync

import (
	"context"
	"testing"
	"time"

	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/rlp"
	"github.com/thetatoken/theta/store/database/backend"
	"github.com/thetatoken/theta/store/kvstore"

	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/dispatcher"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/p2p/simulation"
	"github.com/thetatoken/theta/p2p/types"
)

type MockMessageConsumer struct {
	Received []interface{}
}

func NewMockMessageConsumer() *MockMessageConsumer {
	return &MockMessageConsumer{
		Received: []interface{}{},
	}
}

func (m *MockMessageConsumer) AddMessage(msg interface{}) {
	m.Received = append(m.Received, msg)
}

type MockMsgHandler struct {
	C chan interface{}
}

func (m *MockMsgHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{}
}

func (m *MockMsgHandler) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (types.Message, error) {
	return types.Message{}, nil
}

func (m *MockMsgHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	return nil, nil
}

func (m *MockMsgHandler) HandleMessage(message types.Message) error {
	m.C <- message.Content
	return nil
}

func TestSyncManager(t *testing.T) {
	assert := assert.New(t)
	core.ResetTestBlocks()

	// node1's chain initially contains only A0, A1
	initChain := blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"B2", "A1",
	})
	// node2's chain
	_ = blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"A2", "A1",
		"A3", "A2",
		"A4", "A3",
		"C3", "A2",
	})
	simnet := simulation.NewSimnet()
	net1 := simnet.AddEndpoint("node1")
	net2 := simnet.AddEndpoint("node2")
	mockMsgHandler := &MockMsgHandler{C: make(chan interface{}, 128)}
	net2.RegisterMessageHandler(mockMsgHandler)
	simnet.Start(context.Background())

	privKey, _, _ := crypto.GenerateKeyPair()
	valMgr := consensus.NewFixedValidatorManager()
	db := kvstore.NewKVStore(backend.NewMemDatabase())
	dispatch := dispatcher.NewDispatcher(net1, nil)
	consensus := consensus.NewConsensusEngine(privKey, db, initChain, dispatch, valMgr)
	mockMsgConsumer := NewMockMessageConsumer()

	sm := NewSyncManager(initChain, consensus, net1, nil, dispatch, mockMsgConsumer)
	sm.Start(context.Background())

	// Send block A4 to node1
	payload, _ := rlp.EncodeToBytes(core.CreateTestBlock("A4", "A3"))
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.DataResponse{
			ChannelID: common.ChannelIDBlock,
			Payload:   payload,
		},
	})

	// node1 should broadcast InventoryResponse
	var res interface{}
	res = <-mockMsgHandler.C
	msg1, ok := res.(dispatcher.InventoryResponse)
	assert.True(ok)
	assert.Equal(common.ChannelIDBlock, msg1.ChannelID)
	assert.Equal(core.GetTestBlock("A4").Hash().Hex(), msg1.Entries[0])

	res = <-mockMsgHandler.C
	msg11, ok := res.(dispatcher.DataResponse)
	assert.True(ok)
	assert.Equal(common.ChannelIDHeader, msg11.ChannelID)

	res = <-mockMsgHandler.C
	msg2, ok := res.(dispatcher.InventoryRequest)
	assert.True(ok)
	assert.Equal(common.ChannelIDBlock, msg2.ChannelID)
	assert.Equal(3, len(msg2.Starts))
	assert.Equal(core.GetTestBlock("B2").Hash().Hex(), msg2.Starts[0])
	assert.Equal(core.GetTestBlock("A1").Hash().Hex(), msg2.Starts[1])
	assert.Equal(core.GetTestBlock("A0").Hash().Hex(), msg2.Starts[2])

	// node2 replies with InventoryReponse
	entries := []string{}
	for _, name := range []string{"A0", "A1", "A2", "A3", "A4"} {
		entries = append(entries, core.GetTestBlock(name).Hash().Hex())
	}
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.InventoryResponse{
			ChannelID: common.ChannelIDBlock,
			Entries:   entries,
		},
	})

	// node2 replies with A3 first
	payload, _ = rlp.EncodeToBytes(core.CreateTestBlock("A3", "A2"))
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.DataResponse{
			ChannelID: common.ChannelIDBlock,
			Payload:   payload,
		},
	})

	time.Sleep(1 * time.Second)

	// node2 replies with A2 next
	payload, _ = rlp.EncodeToBytes(core.CreateTestBlock("A2", "A1"))
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.DataResponse{
			ChannelID: common.ChannelIDBlock,
			Payload:   payload,
		},
	})

	time.Sleep(1 * time.Second)

	sm.Stop()
	sm.Wait()

	// Sync manager should output A2, A3, A4 in order.
	assert.Equal(3, len(mockMsgConsumer.Received))
	expected := []string{"A2", "A3", "A4"}
	for i, msg := range mockMsgConsumer.Received {
		assert.Equal(core.GetTestBlock(expected[i]).Hash(), msg.(*core.Block).Hash())
	}
}

type MockConsensus struct {
	chain *blockchain.Chain
	lfb   *core.ExtendedBlock
}

func NewMockConsensus(chain *blockchain.Chain, lfb *core.ExtendedBlock) *MockConsensus {
	return &MockConsensus{
		chain: chain,
		lfb:   lfb,
	}
}

// ID() string
// PrivateKey() *crypto.PrivateKey
// GetTip(includePendingBlockingLeaf bool) *ExtendedBlock
// GetEpoch() uint64
// GetLedger() Ledger
// AddMessage(msg interface{})
// FinalizedBlocks() chan *Block
// GetLastFinalizedBlock() *ExtendedBlock

func (c *MockConsensus) ID() string {
	return ""
}

func (c *MockConsensus) PrivateKey() *crypto.PrivateKey {
	return nil
}

func (c *MockConsensus) GetTip(includePendingBlockingLeaf bool) *core.ExtendedBlock {
	return nil
}

func (c *MockConsensus) GetEpoch() uint64 {
	return 0
}

func (c *MockConsensus) GetLedger() core.Ledger {
	return (*ledger.Ledger)(nil)
}
func (c *MockConsensus) AddMessage(msg interface{}) {

}
func (c *MockConsensus) FinalizedBlocks() chan *core.Block {
	return make(chan *core.Block)
}
func (c *MockConsensus) GetLastFinalizedBlock() *core.ExtendedBlock {
	return c.lfb
}

func TestCollectBlocks(t *testing.T) {
	assert := assert.New(t)
	core.ResetTestBlocks()

	initChain := blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"A2", "A1",
		"A3", "A2",
		"A4", "A3",
		"A5", "A4",
		"C3", "A2",
		"D4", "A3",
	})

	initChain.FinalizePreviousBlocks(core.GetTestBlock("A3").Hash())

	simnet := simulation.NewSimnet()
	net1 := simnet.AddEndpoint("node1")
	net2 := simnet.AddEndpoint("node2")
	mockMsgHandler := &MockMsgHandler{C: make(chan interface{}, 128)}
	net2.RegisterMessageHandler(mockMsgHandler)
	simnet.Start(context.Background())

	dispatch := dispatcher.NewDispatcher(net1, nil)
	a3, _ := initChain.FindBlock(core.GetTestBlock("A3").Hash())
	consensus := NewMockConsensus(initChain, a3)
	mockMsgConsumer := NewMockMessageConsumer()

	sm := NewSyncManager(initChain, consensus, net1, nil, dispatch, mockMsgConsumer)

	blocks := sm.collectBlocks(core.GetTestBlock("A1").Hash(), core.GetTestBlock("A5").Hash())
	// Expected blocks: [A1, A2, A3, A4, D4, A5, A3]
	assert.Equal(7, len(blocks))
	assert.Equal(core.GetTestBlock("A1").Hash().Hex(), blocks[0])
	assert.Equal(core.GetTestBlock("A2").Hash().Hex(), blocks[1])
	assert.Equal(core.GetTestBlock("A3").Hash().Hex(), blocks[2])
	assert.Equal(core.GetTestBlock("A4").Hash().Hex(), blocks[3])
	assert.Equal(core.GetTestBlock("D4").Hash().Hex(), blocks[4])
	assert.Equal(core.GetTestBlock("A5").Hash().Hex(), blocks[5])
	assert.Equal(core.GetTestBlock("A3").Hash().Hex(), blocks[6])
}
