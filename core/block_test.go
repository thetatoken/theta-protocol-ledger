package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
)

func TestBlockHash(t *testing.T) {
	assert := assert.New(t)

	eb := &ExtendedBlock{}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{},
	}
	assert.Equal(eb.Hash(), common.Hash{})

	eb = &ExtendedBlock{
		Block: &Block{
			BlockHeader: &BlockHeader{
				Epoch: 1,
			},
		},
	}
	assert.Equal("0xfa9b2e03cb098783f8b39cd7faa5386118cc1d10d79ee0d20b1665f2c4a7d701", eb.Hash().Hex())

}
