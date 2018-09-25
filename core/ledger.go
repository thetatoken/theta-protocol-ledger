package core

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types/result"
)

//
// Ledger defines the interface of the ledger
//
type Ledger interface {
	BeginBlock(blockHash common.Bytes, header BlockHeader) result.Result
	CheckTx(txBytes common.Bytes) result.Result
	DeliverTx(txBytes common.Bytes) result.Result
	EndBlock(height uint64) result.Result
	Commit() result.Result
	Query()
}
