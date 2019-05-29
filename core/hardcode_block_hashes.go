package core

import "github.com/thetatoken/theta/common"

type HardcodeBlockHash struct {
	Height    uint64
	BlockHash common.Hash
}

// HardcodeBlockHashes contains hardcode block hashes for certain heights
var HardcodeBlockHashes = map[uint64]string{
	GenesisBlockHeight: "0xd8836c6cf3c3ccea0b015b4ed0f9efb0ffe6254db793a515843c9d0f68cbab65",
}

// var HardcodeBlockHashes = []HardcodeBlockHash{
// 	HardcodeBlockHash{Height: 0, BlockHash: common.HexToHash("0xd8836c6cf3c3ccea0b015b4ed0f9efb0ffe6254db793a515843c9d0f68cbab65")}
// }
