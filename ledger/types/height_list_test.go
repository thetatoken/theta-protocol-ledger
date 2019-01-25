package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/rlp"
)

func TestHeightListRLPEncoding(t *testing.T) {
	assert := assert.New(t)

	heights := []uint64{11, 17, 32, 31}
	hl := &HeightList{}
	hl.Append(heights[0])
	hl.Append(heights[1])
	hl.Append(heights[2])
	hl.Append(heights[3])

	encodedHlBytes, err := rlp.EncodeToBytes(hl)
	assert.Nil(err)

	var decodedHl *HeightList
	rlp.DecodeBytes(encodedHlBytes, &decodedHl)
	for idx, h := range decodedHl.Heights {
		assert.Equal(heights[idx], h)
	}

	heights = append(heights, 41)
	heights = append(heights, 19934547)

	decodedHl.Append(heights[4])
	decodedHl.Append(heights[5])

	encodedHlBytes, err = rlp.EncodeToBytes(decodedHl)
	assert.Nil(err)

	var decodedHl2 *HeightList
	rlp.DecodeBytes(encodedHlBytes, &decodedHl2)
	for idx, h := range decodedHl2.Heights {
		assert.Equal(heights[idx], h)
	}
}
