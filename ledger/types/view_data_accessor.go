package types

import "github.com/thetatoken/ukulele/common"

type ViewDataAccessor interface {
	GetAccount(addr common.Address) *Account
	SetAccount(addr common.Address, acc *Account)

	GetSplitContract(resourceId common.Bytes) *SplitContract
	SetSplitContract(resourceId common.Bytes, splitContract *SplitContract)
	DeleteSplitContract(resourceId common.Bytes) bool
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}

type ViewDataGetter interface {
	GetAccount(addr common.Address) *Account

	GetSplitContract(resourceId common.Bytes) *SplitContract
}

type ViewDataSetter interface {
	SetAccount(addr common.Address, acc *Account)

	SetSplitContract(resourceId common.Bytes, splitContract *SplitContract)
	DeleteSplitContract(resourceId common.Bytes) bool
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}
