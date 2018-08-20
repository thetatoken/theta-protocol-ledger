package connection

import (
	"sync"

	log "github.com/sirupsen/logrus"
)

const (
	channelSelectionRoundRobinStrategy = 1
)

//
// ChannelGroup contains multiple channels to facilitate fair scheduling
//
type ChannelGroup struct {
	mutex sync.Mutex

	channelMap map[byte]*Channel // map: ChannelID |-> *Channel
	channels   []*Channel        // For iteration with deterministic order

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

func createChannelGroup(cgConfig ChannelGroupConfig, chConfigs []ChannelConfig) (bool, ChannelGroup) {
	var channelSelector ChannelSelector
	if cgConfig.selectionStrategy == channelSelectionRoundRobinStrategy {
		channelSelector = &RoundRobinChannelSelector{}
	} else {
		return false, ChannelGroup{}
	}

	channelGroup := ChannelGroup{
		channelSelector: channelSelector,
		config:          cgConfig,
	}

	channelID := byte(0x0)
	if len(chConfigs) > 255 {
		return false, channelGroup // too many channels
	}
	for _, chConfig := range chConfigs {
		sendBufConfig := getDefaultSendBufferConfig()
		recvBufConfig := getDefaultRecvBufferConfig()
		channel := createChannel(channelID, chConfig, sendBufConfig, recvBufConfig)
		channelGroup.addChannel(&channel)
		channelID += 1
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

	if cg.channelExists(channel.getID()) {
		return false
	}

	cg.channelMap[channel.id] = channel
	cg.channels = append(cg.channels, channel)

	return true
}

func (cg *ChannelGroup) deleteChannel(channelID byte) {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	if _, ok := cg.channelMap[channelID]; !ok {
		return
	}

	delete(cg.channelMap, channelID)
	for idx, ch := range cg.channels {
		if ch.id == channelID {
			cg.channels = append(cg.channels[:idx], cg.channels[idx+1:]...)
		}
	}
}

func (cg *ChannelGroup) getChannel(channelID byte) *Channel {
	channel, exists := cg.channelMap[channelID]
	if !exists {
		return nil
	}
	return channel
}

func (cg *ChannelGroup) channelExists(channelID byte) bool {
	_, exists := cg.channelMap[channelID]
	return exists
}

func (cg *ChannelGroup) getAllChannels() *([]*Channel) {
	return &cg.channels
}

func (cg *ChannelGroup) getTotalNumChannels() uint {
	return uint(len(cg.channels))
}

func (cg *ChannelGroup) nextChannelToSendPacket() (sucess bool, channel *Channel) {
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

func (rrcs *RoundRobinChannelSelector) nextSelectedChannelIndex(cg *ChannelGroup) (success bool, index int) {
	totalNumberOfChannels := len(*(cg.getAllChannels()))
	if totalNumberOfChannels == 0 {
		log.Errorf("[p2p] the channel group contains no channel")
		return false, -1
	}
	if rrcs.lastUsedChannelIndex < totalNumberOfChannels-1 {
		rrcs.lastUsedChannelIndex = rrcs.lastUsedChannelIndex + 1
	} else {
		rrcs.lastUsedChannelIndex = 0
	}
	return true, rrcs.lastUsedChannelIndex
}
