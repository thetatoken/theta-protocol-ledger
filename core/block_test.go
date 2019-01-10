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
	assert.Equal("0x29b9c99265a3ee1830822488517017a7018739dc2ddd1acd58059a393fdca4a7", eb.Hash().Hex())

}
