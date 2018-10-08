package types

import "github.com/thetatoken/ukulele/common"

type ViewDataAccessor interface {
	Height() uint64

	GetAccount(addr common.Address) *Account
	SetAccount(addr common.Address, acc *Account)

	GetSplitContract(resourceID common.Bytes) *SplitContract
	SetSplitContract(resourceID common.Bytes, splitContract *SplitContract)
	DeleteSplitContract(resourceID common.Bytes) bool
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}

type ViewDataGetter interface {
	Height() uint64
	GetAccount(addr common.Address) *Account
	GetSplitContract(resourceID common.Bytes) *SplitContract
}

type ViewDataSetter interface {
	SetAccount(addr common.Address, acc *Account)

	SetSplitContract(resourceID common.Bytes, splitContract *SplitContract)
	DeleteSplitContract(resourceID common.Bytes) bool
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}
