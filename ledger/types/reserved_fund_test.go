package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasResourceId(t *testing.T) {
	assert := assert.New(t)

	rid1 := []byte("rid001")
	rid2 := []byte("rid002")
	rid3 := []byte("rid003")
	rid4 := []byte("rid004")

	rf := ReservedFund{}
	rf.ResourceIds = append(rf.ResourceIds, rid1)
	rf.ResourceIds = append(rf.ResourceIds, rid2)
	rf.ResourceIds = append(rf.ResourceIds, rid3)
	rf.ResourceIds = append(rf.ResourceIds, rid1)

	assert.Equal(rf.HasResourceId(rid1), true)
	assert.Equal(rf.HasResourceId(rid4), false)
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
