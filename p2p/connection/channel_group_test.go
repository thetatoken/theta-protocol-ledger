package connection

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
)

func TestDefaultChannelGroupAddChannel(t *testing.T) {
	assert := assert.New(t)

	cg := newTestEmptyChannelGroup()
	assert.Equal(uint(0), cg.getTotalNumChannels())

	ch1 := createDefaultChannel(common.ChannelIDCheckpoint)
	success := cg.addChannel(&ch1)
	assert.True(success)
	assert.Equal(uint(1), cg.getTotalNumChannels())
	assert.True(cg.channelExists(ch1.getID()))
	assert.Equal(&ch1, cg.getChannel(ch1.getID()))

	ch1a := createDefaultChannel(common.ChannelIDCheckpoint)
	success = cg.addChannel(&ch1a)
	assert.False(success) // cannot add two channels with the same ChannelID
	assert.Equal(uint(1), cg.getTotalNumChannels())

	ch2 := createDefaultChannel(common.ChannelIDHeader)
	success = cg.addChannel(&ch2)
	assert.True(success)
	assert.Equal(uint(2), cg.getTotalNumChannels())
	assert.True(cg.channelExists(ch2.getID()))
	assert.Equal(&ch2, cg.getChannel(ch2.getID()))
}

func TestDefaultChannelGroupDeleteChannel(t *testing.T) {
	assert := assert.New(t)

	cg := newTestEmptyChannelGroup()
	assert.Equal(uint(0), cg.getTotalNumChannels())

	ch1 := createDefaultChannel(common.ChannelIDCheckpoint)
	ch2 := createDefaultChannel(common.ChannelIDHeader)
	ch3 := createDefaultChannel(common.ChannelIDTransaction)

	assert.True(cg.addChannel(&ch1))
	assert.Equal(uint(1), cg.getTotalNumChannels())
	assert.True(cg.addChannel(&ch2))
	assert.Equal(uint(2), cg.getTotalNumChannels())
	assert.True(cg.addChannel(&ch3))
	assert.Equal(uint(3), cg.getTotalNumChannels())

	assert.True(cg.channelExists(ch1.getID()))
	assert.True(cg.channelExists(ch2.getID()))
	assert.True(cg.channelExists(ch3.getID()))

	cg.deleteChannel(ch1.getID())
	assert.False(cg.channelExists(ch1.getID()))
	assert.Equal(uint(2), cg.getTotalNumChannels())
	cg.deleteChannel(ch2.getID())
	assert.False(cg.channelExists(ch2.getID()))
	assert.Equal(uint(1), cg.getTotalNumChannels())
	cg.deleteChannel(ch3.getID())
	assert.False(cg.channelExists(ch3.getID()))
	assert.Equal(uint(0), cg.getTotalNumChannels())
}

func TestDefaultChannelIterationOrder(t *testing.T) {
	assert := assert.New(t)

	cg := newTestEmptyChannelGroup()

	ch1 := createDefaultChannel(common.ChannelIDCheckpoint)
	ch2 := createDefaultChannel(common.ChannelIDHeader)
	ch3 := createDefaultChannel(common.ChannelIDTransaction)
	ch4 := createDefaultChannel(common.ChannelIDBlock)
	ch5 := createDefaultChannel(common.ChannelIDVote)
	ch6 := createDefaultChannel(common.ChannelIDPeerDiscovery)

	assert.True(cg.addChannel(&ch1))
	assert.True(cg.addChannel(&ch2))
	assert.True(cg.addChannel(&ch3))
	assert.True(cg.addChannel(&ch4))
	assert.True(cg.addChannel(&ch5))
	assert.True(cg.addChannel(&ch6))

	allChannels := cg.getAllChannels()
	assert.Equal((*allChannels)[0], &ch1)
	assert.Equal((*allChannels)[1], &ch2)
	assert.Equal((*allChannels)[2], &ch3)
	assert.Equal((*allChannels)[3], &ch4)
	assert.Equal((*allChannels)[4], &ch5)
	assert.Equal((*allChannels)[5], &ch6)

	// Delete two channels

	cg.deleteChannel(ch3.getID())
	cg.deleteChannel(ch5.getID())

	allChannels = cg.getAllChannels()
	assert.Equal((*allChannels)[0], &ch1)
	assert.Equal((*allChannels)[1], &ch2)
	assert.Equal((*allChannels)[2], &ch4)
	assert.Equal((*allChannels)[3], &ch6)

	// Add the two channels back

	assert.True(cg.addChannel(&ch5))
	assert.True(cg.addChannel(&ch3))

	allChannels = cg.getAllChannels()
	assert.Equal((*allChannels)[0], &ch1)
	assert.Equal((*allChannels)[1], &ch2)
	assert.Equal((*allChannels)[2], &ch4)
	assert.Equal((*allChannels)[3], &ch6)
	assert.Equal((*allChannels)[4], &ch5)
	assert.Equal((*allChannels)[5], &ch3)
}

func TestRoundRobinChannelSelector1(t *testing.T) {
	assert := assert.New(t)

	cg := newTestEmptyChannelGroup()

	ch1 := createDefaultChannel(common.ChannelIDCheckpoint)
	ch2 := createDefaultChannel(common.ChannelIDHeader)
	ch3 := createDefaultChannel(common.ChannelIDBlock)
	ch4 := createDefaultChannel(common.ChannelIDVote)
	ch5 := createDefaultChannel(common.ChannelIDTransaction)

	assert.True(cg.addChannel(&ch1))
	assert.True(cg.addChannel(&ch2))
	assert.True(cg.addChannel(&ch3))
	assert.True(cg.addChannel(&ch4))
	assert.True(cg.addChannel(&ch5))

	rrcs := createRoundRobinChannelSelector()
	success, index := rrcs.nextSelectedChannelIndex(&cg)
	assert.True(success)
	assert.Equal(0, index)
	t.Logf("index = %v", index)
	success, index = rrcs.nextSelectedChannelIndex(&cg)
	assert.True(success)
	assert.Equal(1, index)
	t.Logf("index = %v", index)
	success, index = rrcs.nextSelectedChannelIndex(&cg)
	assert.True(success)
	assert.Equal(2, index)
	t.Logf("index = %v", index)
	success, index = rrcs.nextSelectedChannelIndex(&cg)
	assert.True(success)
	assert.Equal(3, index)
	t.Logf("index = %v", index)
	success, index = rrcs.nextSelectedChannelIndex(&cg)
	assert.True(success)
	assert.Equal(4, index)
	t.Logf("index = %v", index)
}

func TestRoundRobinChannelSelector2(t *testing.T) {
	assert := assert.New(t)

	cg := newTestEmptyChannelGroup()
	ch1 := createDefaultChannel(common.ChannelIDCheckpoint)
	ch2 := createDefaultChannel(common.ChannelIDHeader)
	ch3 := createDefaultChannel(common.ChannelIDBlock)
	ch4 := createDefaultChannel(common.ChannelIDVote)
	ch5 := createDefaultChannel(common.ChannelIDTransaction)

	assert.True(cg.addChannel(&ch1))
	assert.True(cg.addChannel(&ch2))
	assert.True(cg.addChannel(&ch3))
	assert.True(cg.addChannel(&ch4))
	assert.True(cg.addChannel(&ch5))

	// Only some of the channels have messages to send
	assert.True(ch1.enqueueMessage([]byte("test1")))
	assert.True(ch2.enqueueMessage([]byte("test2")))
	assert.True(ch5.enqueueMessage([]byte("test5")))

	success, ch := cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch1, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch2, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch5, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch1, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch2, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch5, ch)

	// Clearing a channel
	port := 43253
	netconn := p2ptypes.GetTestNetconn(port)
	cfg := GetDefaultConnectionConfig()
	conn := CreateConnection(netconn, cfg)
	conn.Start(context.Background())

	nonempty, _, err := ch1.sendPacketTo(conn)
	assert.True(nonempty)
	assert.Nil(err)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch2, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch5, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch2, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch5, ch)

	// Enqueue message on some channels

	assert.True(ch4.enqueueMessage([]byte("test4")))
	assert.True(ch3.enqueueMessage([]byte("test3")))

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch2, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch3, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch4, ch)

	success, ch = cg.nextChannelToSendPacket()
	assert.True(success)
	assert.Equal(&ch5, ch)
}

// --------------- Test Utilities --------------- //

func newTestEmptyChannelGroup() ChannelGroup {
	cgCfg := getDefaultChannelGroupConfig()
	channels := []*Channel{}
	success, dcg := createChannelGroup(cgCfg, channels)
	if !success {
		panic("Failed to create channel group!")
	}

	return dcg
}
