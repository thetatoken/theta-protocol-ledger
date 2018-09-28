package core

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
)

//
// Ledger defines the interface of the ledger
//
type Ledger interface {
	CheckTx(rawTx common.Bytes) result.Result
	DeliverBlockTxs() (blockRawTxs []common.Bytes, res result.Result)
	Query()
}
