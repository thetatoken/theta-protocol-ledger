package core

import (
	"github.com/thetatoken/ukulele/common"
)

type SnapshotRecord struct {
	K common.Bytes // key
	V common.Bytes // value
	R common.Bytes // account root, if any
	S int          // sequence of storeview
}

type DirectlyFinalizedBlockPair struct {
	First  ExtendedBlock
	Second ExtendedBlock
}

type SnapshotMetadata struct {
	Blockheader               BlockHeader
	Votes                     []Vote
	BlocksWithValidatorChange []DirectlyFinalizedBlockPair
}
