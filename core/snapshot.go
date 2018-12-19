package core

import "github.com/thetatoken/ukulele/common"

type SnapshotRecord struct {
	K common.Bytes // key
	V common.Bytes // value
	R common.Bytes // account root, if any
}
