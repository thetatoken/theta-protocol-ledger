package core

import (
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
)

type ViewSelector int

const (
	DeliveredView ViewSelector = 1
	CheckedView   ViewSelector = 2
	ScreenedView  ViewSelector = 3
)

//
// TxInfo encapsulates information used by mempool to sorting.
//
type TxInfo struct {
	EffectiveGasPrice *big.Int
	Address           common.Address
	Sequence          uint64
}

//
// Ledger defines the interface of the ledger
//
type Ledger interface {
	ScreenTx(rawTx common.Bytes) (priority *TxInfo, res result.Result)
	ProposeBlockTxs() (stateRootHash common.Hash, blockRawTxs []common.Bytes, res result.Result)
	ApplyBlockTxs(blockRawTxs []common.Bytes, expectedStateRoot common.Hash) result.Result
	ResetState(height uint64, rootHash common.Hash) result.Result
	FinalizeState(height uint64, rootHash common.Hash) result.Result
	GetFinalizedValidatorCandidatePool(blockHash common.Hash, isNext bool) (*ValidatorCandidatePool, error)
	PruneState(endHeight uint64) error
}
