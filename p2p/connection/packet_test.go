package connection

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func TestPacketEmptiness(t *testing.T) {
	assert := assert.New(t)

	packet := Packet{
		ChannelID: common.ChannelIDTransaction,
	}
	assert.True(packet.isEmpty())

	packet.Bytes = []byte("hello world")
	assert.False(packet.isEmpty())
}
