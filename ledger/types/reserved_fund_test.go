package types

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasResourceID(t *testing.T) {
	assert := assert.New(t)

	rid1 := "rid001"
	rid2 := "rid002"
	rid3 := "rid003"
	rid4 := "rid004"

	rf := ReservedFund{}
	rf.ResourceIDs = append(rf.ResourceIDs, rid1)
	rf.ResourceIDs = append(rf.ResourceIDs, rid2)
	rf.ResourceIDs = append(rf.ResourceIDs, rid3)
	rf.ResourceIDs = append(rf.ResourceIDs, rid1)

	assert.Equal(rf.HasResourceID(rid1), true)
	assert.Equal(rf.HasResourceID(rid4), false)
}

func TestRecordTransfer(t *testing.T) {
	assert := assert.New(t)

	sp1 := ServicePaymentTx{}
	sp2 := ServicePaymentTx{}
	sp3 := ServicePaymentTx{}

	rf := ReservedFund{}
	rf.RecordTransfer(&sp1)
	rf.RecordTransfer(&sp2)
	rf.RecordTransfer(&sp3)

	assert.Equal(len(rf.TransferRecords), 3)
}

func TestReserveFundJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	reserveFund := ReservedFund{
		EndBlockHeight: math.MaxUint64,
	}

	s, err := json.Marshal(reserveFund)
	require.Nil(err)

	var d ReservedFund
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.EndBlockHeight)
}
