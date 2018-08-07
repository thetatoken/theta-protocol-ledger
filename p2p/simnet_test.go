// +build integration

package p2p

import (
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SimMessageHandler struct {
	lock             *sync.Mutex
	ReceivedMessages []string
}

func (sm *SimMessageHandler) HandleMessage(self Network, msg interface{}) {
	sm.lock.Lock()
	defer sm.lock.Unlock()
	sm.ReceivedMessages = append(sm.ReceivedMessages, fmt.Sprintf("%s <- %v", self.ID(), msg))
}

func TestSimnetBroadcast(t *testing.T) {
	assert := assert.New(t)
	msgHandler := &SimMessageHandler{lock: &sync.Mutex{}}
	simnet := NewSimnetWithHandler(msgHandler)
	e1 := simnet.AddEndpoint("e1")
	e2 := simnet.AddEndpoint("e2")
	simnet.AddEndpoint("e3")
	simnet.Start()

	e2.Broadcast("hello!")
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e1 <- hello!", "e2 <- hello!", "e3 <- hello!"}, msgHandler.ReceivedMessages)

	msgHandler.ReceivedMessages = make([]string, 0)
	e1.Broadcast("world!")
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

	e1.Send("e3", "hello!")
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e3 <- hello!"}, msgHandler.ReceivedMessages)

	msgHandler.ReceivedMessages = make([]string, 0)
	e1.Send("e1", "world!")
	time.Sleep(1 * time.Second)
	msgHandler.lock.Lock()
	sort.Strings(msgHandler.ReceivedMessages)
	msgHandler.lock.Unlock()
	assert.EqualValues([]string{"e1 <- world!"}, msgHandler.ReceivedMessages)
}
