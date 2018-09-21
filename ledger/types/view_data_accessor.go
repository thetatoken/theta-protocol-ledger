package types

type ViewDataAccessor interface {
	GetAccount(addr []byte) *Account
	SetAccount(addr []byte, acc *Account)

	GetSplitContract(resourceId []byte) *SplitContract
	SetSplitContract(resourceId []byte, splitContract *SplitContract)
	DeleteSplitContract(resourceId []byte) (SplitContractBytes []byte, deleted bool)
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}

type ViewDataGetter interface {
	GetAccount(addr []byte) *Account

	GetSplitContract(resourceId []byte) *SplitContract
}

type ViewDataSetter interface {
	SetAccount(addr []byte, acc *Account)

	SetSplitContract(resourceId []byte, splitContract *SplitContract)
	DeleteSplitContract(resourceId []byte) (SplitContractBytes []byte, deleted bool)
	DeleteExpiredSplitContracts(currentBlockHeight uint64) bool
}
