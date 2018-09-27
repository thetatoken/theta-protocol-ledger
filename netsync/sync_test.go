// +build unit

package netsync

import (
	"context"
	"testing"
	"time"

	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/rlp"

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

	// node1's chain initially contains only A0, A1
	initChain := blockchain.CreateTestChainByBlocks([]string{
		"A1", "A0",
	})
	simnet := simulation.NewSimnet()
	net1 := simnet.AddEndpoint("node1")
	net2 := simnet.AddEndpoint("node2")
	mockMsgHandler := &MockMsgHandler{C: make(chan interface{}, 128)}
	net2.RegisterMessageHandler(mockMsgHandler)
	simnet.Start(context.Background())

	consensus := consensus.NewConsensusEngine(initChain, net1, core.NewValidatorSet())
	mockMsgConsumer := NewMockMessageConsumer()
	dispatch := dispatcher.NewDispatcher(net1)

	sm := NewSyncManager(initChain, consensus, net1, dispatch, mockMsgConsumer)
	sm.Start(context.Background())

	// Send block A4 to node1
	payload, _ := rlp.EncodeToBytes(blockchain.CreateTestBlock("A4", "A3"))
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
	assert.Equal("A0", msg1.Start)

	// node2 replies with InventoryReponse
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.InventoryResponse{
			ChannelID: common.ChannelIDBlock,
			Entries:   []string{"A0", "A1", "A2", "A3", "A4"},
		},
	})

	// node1 should send DataRequest for A2, A3
	res = <-mockMsgHandler.C
	msg2, ok := res.(dispatcher.DataRequest)
	assert.True(ok)
	assert.Equal(common.ChannelIDBlock, msg2.ChannelID)
	// assert.Equal([]string{"A3"}, msg2.Entries)

	// node2 replies with A3 first
	payload, _ = rlp.EncodeToBytes(blockchain.CreateTestBlock("A3", "A2"))
	net2.Broadcast(types.Message{
		ChannelID: common.ChannelIDBlock,
		Content: dispatcher.DataResponse{
			ChannelID: common.ChannelIDBlock,
			Payload:   payload,
		},
	})

	res = <-mockMsgHandler.C
	msg3, ok := res.(dispatcher.DataRequest)
	assert.True(ok)
	assert.Equal(common.ChannelIDBlock, msg3.ChannelID)
	// assert.Equal([]string{"A2"}, msg3.Entries)

	time.Sleep(1 * time.Second)

	// node2 replies with A2 next
	payload, _ = rlp.EncodeToBytes(blockchain.CreateTestBlock("A2", "A1"))
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
		assert.Equal(expected[i], msg.(*core.Block).Hash.String())
	}
}
