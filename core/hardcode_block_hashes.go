package core

import "github.com/thetatoken/theta/common"

type HardcodeBlockHash struct {
	Height    uint64
	BlockHash common.Hash
}

// HardcodeBlockHashes maps block heights to hardcode block hashes
var HardcodeBlockHashes = map[uint64]string{}
