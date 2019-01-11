package netsync

import (
	"context"
	"testing"
	"time"

	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/rlp"
	"github.com/thetatoken/ukulele/store/database/backend"
	"github.com/thetatoken/ukulele/store/kvstore"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/dispatcher"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/p2p/simulation"
	"github.com/thetatoken/ukulele/p2p/types"
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
	})
	// node2's chain
	_ = blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
		"A2", "A1",
		"A3", "A2",
		"A4", "A3",
	})
	simnet := simulation.NewSimnet()
	net1 := simnet.AddEndpoint("node1")
	net2 := simnet.AddEndpoint("node2")
	mockMsgHandler := &MockMsgHandler{C: make(chan interface{}, 128)}
	net2.RegisterMessageHandler(mockMsgHandler)
	simnet.Start(context.Background())

	valMgr := consensus.NewFixedValidatorManager()
	db := kvstore.NewKVStore(backend.NewMemDatabase())
	dispatch := dispatcher.NewDispatcher(net1)
	consensus := consensus.NewConsensusEngine(nil, db, initChain, dispatch, valMgr)
	mockMsgConsumer := NewMockMessageConsumer()

	sm := NewSyncManager(initChain, consensus, net1, dispatch, mockMsgConsumer)
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

	// node1 should broadcast InventoryRequest
	var res interface{}
	res = <-mockMsgHandler.C
	msg1, ok := res.(dispatcher.InventoryRequest)
	assert.True(ok)
	assert.Equal(common.ChannelIDBlock, msg1.ChannelID)
	assert.Equal(core.GetTestBlock("A1").Hash().Hex(), msg1.Start)

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

	// Sync manager should output A2, A3, A4 in order.
	assert.Equal(3, len(mockMsgConsumer.Received))
	expected := []string{"A2", "A3", "A4"}
	for i, msg := range mockMsgConsumer.Received {
		assert.Equal(core.GetTestBlock(expected[i]).Hash(), msg.(*core.Block).Hash())
	}
}
