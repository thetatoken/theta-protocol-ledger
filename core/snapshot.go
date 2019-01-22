package core

import (
	"github.com/thetatoken/ukulele/common"
)

const (
	SVStart = iota
	SVEnd
)

type SnapshotTrieRecord struct {
	K common.Bytes // key
	V common.Bytes // value
}

type SnapshotBlock struct {
	Header BlockHeader
	Votes  []Vote
}
type SnapshotBlockTrio struct {
	First  BlockHeader
	Second BlockHeader
	Third  SnapshotBlock
}

type SnapshotMetadata struct {
	BlockTrios []SnapshotBlockTrio
}
