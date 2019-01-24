package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
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
	assert.Equal("0x87a331c1e807476de260f2dc2e4d531dc42500764587605c7574179bc4cbd5bc", eb.Hash().Hex())
}

func TestCreateTestBlock(t *testing.T) {
	assert := assert.New(t)

	b11 := CreateTestBlock("B1", "")
	b12 := CreateTestBlock("b1", "")

	assert.Equal(b11.Hash(), b12.Hash())
}
