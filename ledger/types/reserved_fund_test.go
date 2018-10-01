package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasResourceID(t *testing.T) {
	assert := assert.New(t)

	rid1 := []byte("rid001")
	rid2 := []byte("rid002")
	rid3 := []byte("rid003")
	rid4 := []byte("rid004")

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
