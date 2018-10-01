package state

import "github.com/thetatoken/ukulele/common"

//
// ------------------------- Ledger State Keys -------------------------
//

// ChainIDKey returns the key for chainID
func ChainIDKey() common.Bytes {
	return common.Bytes("chainid")
}

// AccountKey construct the state key for the given address
func AccountKey(addr common.Address) common.Bytes {
	return append(common.Bytes("ls/a/"), addr[:]...)
}

// SplitContractKeyPrefix returns the prefix for the split contract key
func SplitContractKeyPrefix() common.Bytes {
	return common.Bytes("ls/ssc/split/") // special smart contract / split contract
}

// SplitContractKey construct the state key for the given resourceID
func SplitContractKey(resourceID common.Bytes) common.Bytes {
	return append(SplitContractKeyPrefix(), resourceID...)
}
