// +build integration

package simulation

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

type SimMessageHandler struct {
	lock             *sync.Mutex
	ReceivedMessages []string
}

func (sm *SimMessageHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{
		common.ChannelIDHeader,
		common.ChannelIDBlock,
		common.ChannelIDVote,
	}
}

func (sm *SimMessageHandler) ParseMessage(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{
		ChannelID: channelID,
		Content:   rawMessageBytes,
	}
	return message, nil
}

func (sm *SimMessageHandler) HandleMessage(peerID string, msg p2ptypes.Message) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.ReceivedMessages = append(sm.ReceivedMessages, fmt.Sprintf("%s <- %v", peerID, msg.Content))
}

func createBlockMessage(content string) p2ptypes.Message {
	return p2ptypes.Message{
		ChannelID: common.ChannelIDBlock,
		Content:   content,
	}
}

func TestSimnetBroadcast(t *testing.T) {
	assert := assert.New(t)
	msgHandler := &SimMessageHandler{lock: &sync.Mutex{}}
	simnet := NewSimnetWithHandler(msgHandler)
	e1 := simnet.AddEndpoint("e1")
	e2 := simnet.AddEndpoint("e2")
	simnet.AddEndpoint("e3")
	simnet.Start()

	e2.Broadcast(createBlockMessage("hello!"))
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e1 <- hello!", "e2 <- hello!", "e3 <- hello!"}, msgHandler.ReceivedMessages)

	msgHandler.ReceivedMessages = make([]string, 0)
	e1.Broadcast(createBlockMessage("world!"))
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e1 <- world!", "e2 <- world!", "e3 <- world!"}, msgHandler.ReceivedMessages)
}

func TestSimnetSend(t *testing.T) {
	assert := assert.New(t)
	msgHandler := &SimMessageHandler{lock: &sync.Mutex{}}
	simnet := NewSimnetWithHandler(msgHandler)
	e1 := simnet.AddEndpoint("e1")
	simnet.AddEndpoint("e2")
	simnet.AddEndpoint("e3")
	simnet.Start()

	e1.Send("e3", createBlockMessage("hello!"))
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e3 <- hello!"}, msgHandler.ReceivedMessages)

	msgHandler.ReceivedMessages = make([]string, 0)
	e1.Send("e1", createBlockMessage("world!"))
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e1 <- world!"}, msgHandler.ReceivedMessages)
}
