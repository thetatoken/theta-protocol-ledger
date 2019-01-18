package core

import (
	"github.com/thetatoken/ukulele/common"
)

const (
	SVStart = iota
	SVEnd
)

type SnapshotRecord struct {
	K common.Bytes // key
	V common.Bytes // value
	B common.Bytes // boundary(start/end) record of sv
	H common.Bytes // sv block height
}

type DirectlyFinalizedBlockTrio struct {
	First  ExtendedBlock
	Second ExtendedBlock
	Third  ExtendedBlock
}

type SnapshotMetadata struct {
	Blockheader               BlockHeader
	Votes                     []Vote
	BlocksWithValidatorChange []DirectlyFinalizedBlockTrio
}
