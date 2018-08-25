package connection

import (
	"github.com/thetatoken/ukulele/common"
)

func newDefaultChannel() Channel {
	chCfg := getDefaultChannelConfig()
	sbCfg := getDefaultSendBufferConfig()
	rbCfg := getDefaultRecvBufferConfig()

	channel := createChannel(common.ChannelIDTransaction, chCfg, sbCfg, rbCfg)
	return channel
}
