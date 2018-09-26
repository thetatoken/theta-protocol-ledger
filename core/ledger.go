package core

import (
	"github.com/thetatoken/ukulele/common"
)

//
// Ledger defines the interface of the ledger
//
type Ledger interface {
	BeginBlock(blockHash common.Bytes, header BlockHeader) bool
	CheckTx(txBytes common.Bytes) bool
	DeliverTx(txBytes common.Bytes) bool
	EndBlock(height uint64) bool
	Commit() bool
	Query()
}
