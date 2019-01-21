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
}

type DirectlyFinalizedBlockTrio struct {
	First  BlockHeader
	Second BlockHeader
	Third  BlockHeader
}

type SnapshotMetadata struct {
	Blockheader               BlockHeader
	Votes                     []Vote
	BlocksWithValidatorChange []DirectlyFinalizedBlockTrio
}
