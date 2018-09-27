package state

import "github.com/thetatoken/ukulele/common"

//
// ------------------------- Ledger State Keys -------------------------
//

// AccountKey construct the state key for the given address
func AccountKey(addr common.Address) common.Bytes {
	return append(common.Bytes("base/a/"), addr[:]...)
}

// SplitContractKeyPrefix returns the prefix for the split contract key
func SplitContractKeyPrefix() common.Bytes {
	return common.Bytes("base/ssc/split/") // special smart contract / split contract
}

// SplitContractKey construct the state key for the given resourceId
func SplitContractKey(resourceId common.Bytes) common.Bytes {
	return append(SplitContractKeyPrefix(), resourceId...)
}
