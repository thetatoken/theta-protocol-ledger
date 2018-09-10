package netsync

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
)

func TestMessageEncoding(t *testing.T) {
	assert := assert.New(t)

	sm := &SyncManager{}

	block := blockchain.Block{}
	block.Hash = common.Bytes("hello")

	b, err := sm.EncodeMessage(block)
	assert.Nil(err)

	parsed, err := sm.ParseMessage("", common.ChannelIDBlock, b)
	assert.Nil(err)
	assert.Equal(0, bytes.Compare(block.Hash, parsed.Content.(blockchain.Block).Hash))
}
