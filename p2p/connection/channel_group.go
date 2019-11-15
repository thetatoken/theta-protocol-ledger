package connection

import (
	"sync"

	"github.com/thetatoken/theta/common"
)

const (
	channelSelectionRoundRobinStrategy = 1
)

//
// ChannelGroup contains multiple channels to facilitate fair scheduling
//
type ChannelGroup struct {
	mutex *sync.Mutex

	channelMap map[common.ChannelIDEnum]*Channel // map: ChannelID |-> *Channel
	channels   []*Channel                        // For iteration with deterministic order

	channelSelector ChannelSelector

	config ChannelGroupConfig
}

//
// ChannelGroupConfig specifies the configuration of the ChannelGroup
//
type ChannelGroupConfig struct {
	selectionStrategy int
}

//
// ChannelSelector defines the interface of a Channel selector
//
type ChannelSelector interface {
	nextSelectedChannelIndex(cg *ChannelGroup) (success bool, index int)
}

func createChannelGroup(cgConfig ChannelGroupConfig, channels []*Channel) (bool, ChannelGroup) {
	var channelSelector ChannelSelector
	if cgConfig.selectionStrategy == channelSelectionRoundRobinStrategy {
		channelSelector = createRoundRobinChannelSelector()
	} else {
		logger.Errorf("Invalid channel selection strategy")
		return false, ChannelGroup{}
	}

	channelGroup := ChannelGroup{
		mutex:           &sync.Mutex{},
		channelMap:      make(map[common.ChannelIDEnum]*Channel),
		channelSelector: channelSelector,
		config:          cgConfig,
	}

	for _, channel := range channels {
		channelGroup.addChannel(channel)
	}

	return true, channelGroup
}

func getDefaultChannelGroupConfig() ChannelGroupConfig {
	return ChannelGroupConfig{
		selectionStrategy: channelSelectionRoundRobinStrategy,
	}
}

func (cg *ChannelGroup) addChannel(channel *Channel) bool {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	_, exists := cg.channelMap[channel.getID()]
	if exists {
		return false
	}

	cg.channelMap[channel.id] = channel
	cg.channels = append(cg.channels, channel)

	return true
}

func (cg *ChannelGroup) deleteChannel(channelID common.ChannelIDEnum) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	if _, ok := cg.channelMap[channelID]; !ok {
		return
	}

	delete(cg.channelMap, channelID)
	for idx, ch := range cg.channels {
		if ch.id == channelID {
			cg.channels = append(cg.channels[:idx], cg.channels[idx+1:]...)
			break
		}
	}
}

func (cg *ChannelGroup) getChannel(channelID common.ChannelIDEnum) *Channel {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	channel, exists := cg.channelMap[channelID]
	if !exists {
		return nil
	}
	return channel
}

func (cg *ChannelGroup) channelExists(channelID common.ChannelIDEnum) bool {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	_, exists := cg.channelMap[channelID]
	return exists
}

func (cg *ChannelGroup) getAllChannels() *([]*Channel) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	return &cg.channels
}

func (cg *ChannelGroup) getTotalNumChannels() uint {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	return uint(len(cg.channels))
}

func (cg *ChannelGroup) nextChannelToSendPacket() (success bool, channel *Channel) {
	channels := cg.getAllChannels()
	totalNumberOfChannels := cg.getTotalNumChannels()
	for i := uint(0); i < totalNumberOfChannels; i++ {
		success, selectedChannelIndex := cg.channelSelector.nextSelectedChannelIndex(cg)
		if !success {
			return false, nil
		}
		if selectedChannelIndex > len(*channels) {
			return false, nil
		}
		selectedChannel := (*channels)[selectedChannelIndex]
		if !selectedChannel.hasPacketToSend() {
			continue
		}
		return true, selectedChannel
	}
	return true, nil
}

//
// RoundRobinChannelSelector implments the ChannelSelector interface
// with the round robin strategy
//
type RoundRobinChannelSelector struct {
	lastUsedChannelIndex int
}

func createRoundRobinChannelSelector() ChannelSelector {
	return &RoundRobinChannelSelector{
		lastUsedChannelIndex: -1,
	}
}

func (rrcs *RoundRobinChannelSelector) nextSelectedChannelIndex(cg *ChannelGroup) (success bool, index int) {
	totalNumberOfChannels := len(*(cg.getAllChannels()))
	if totalNumberOfChannels == 0 {
		logger.Errorf("The channel group contains no channel")
		return false, -1
	}
	if rrcs.lastUsedChannelIndex < totalNumberOfChannels-1 {
		rrcs.lastUsedChannelIndex = rrcs.lastUsedChannelIndex + 1
	} else {
		rrcs.lastUsedChannelIndex = 0
	}
	return true, rrcs.lastUsedChannelIndex
}
