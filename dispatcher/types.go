package dispatcher

import (
	"github.com/thetatoken/theta/common"
)

// MaxInventorySize defines the max number of items in InventoryRequest/InventoryResponse.
const MaxInventorySize = 50

// InventoryRequest defines the structure of the inventory request
type InventoryRequest struct {
	ChannelID common.ChannelIDEnum
	Starts    []string // Starting hashes.
	End       string   // Optional ending hash.
}

// InventoryResponse defines the structure of the inventory response
type InventoryResponse struct {
	ChannelID common.ChannelIDEnum
	Entries   []string
}

// DataRequest defines the structure of the data request
type DataRequest struct {
	ChannelID common.ChannelIDEnum
	Entries   []string
}

// DataResponse defines the structure of the data response
type DataResponse struct {
	ChannelID common.ChannelIDEnum
	Payload   common.Bytes
}
