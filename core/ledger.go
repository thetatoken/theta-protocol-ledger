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
	ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result)
	ApplyBlockTxs(blockRawTxs []common.Bytes, expectedStateRoot common.Hash) result.Result
	ResetState(height uint32, rootHash common.Hash) result.Result
	Query()
}
