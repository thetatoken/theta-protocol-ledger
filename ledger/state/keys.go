package state

import "github.com/thetatoken/theta/common"

//
// ------------------------- Ledger State Keys -------------------------
//

// ChainIDKey returns the key for chainID
func ChainIDKey() common.Bytes {
	return common.Bytes("chainid")
}

// AccountKey constructs the state key for the given address
func AccountKey(addr common.Address) common.Bytes {
	return append(common.Bytes("ls/a/"), addr[:]...)
}

// SplitRuleKeyPrefix returns the prefix for the split rule key
func SplitRuleKeyPrefix() common.Bytes {
	return common.Bytes("ls/ssc/split/") // special smart contract / split rule
}

// SplitRuleKey constructs the state key for the given resourceID
func SplitRuleKey(resourceID string) common.Bytes {
	resourceIDBytes := common.Bytes(resourceID)
	return append(SplitRuleKeyPrefix(), resourceIDBytes[:]...)
}

// CodeKey constructs the state key for the given code hash
func CodeKey(codeHash common.Bytes) common.Bytes {
	return append(common.Bytes("ls/ch/"), codeHash...)
}

// ValidatorCandidatePoolKey returns the state key for the validator stake holder set
func ValidatorCandidatePoolKey() common.Bytes {
	return common.Bytes("ls/vcp")
}

// GuardianCandidatePoolKey returns the state key for the guadian stake holder set
func GuardianCandidatePoolKey() common.Bytes {
	return common.Bytes("ls/gcp")
}

// EliteEdgeNodePoolKey returns the state key for the elite edge node TFuel stake holder set
func EliteEdgeNodePoolKey() common.Bytes {
	return common.Bytes("ls/eenp")
}

// StakeTransactionHeightListKey returns the state key the heights of blocks
// that contain stake related transactions (i.e. StakeDeposit, StakeWithdraw, etc)
func StakeTransactionHeightListKey() common.Bytes {
	return common.Bytes("ls/sthl")
}

// StatePruningProgressKey returns the key for the state pruning progress
func StatePruningProgressKey() common.Bytes {
	return common.Bytes("ls/spp")
}

// StakeRewardDistributionRuleSetKey returns the state key for the stake reward distribution rule set
func StakeRewardDistributionRuleSetKey() common.Bytes {
	return common.Bytes("ls/srdrs")
}
