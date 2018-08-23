package dispatcher

import (
	"github.com/thetatoken/ukulele/common"
)

// InventoryRequest defines the structure of the inventory request
type InventoryRequest struct {
	ChannelID common.ChannelIDEnum
	Checksum  common.Bytes
}

// InventoryResponse defines the structure of the inventory response
type InventoryResponse struct {
	ChannelID common.ChannelIDEnum
	Checksum  common.Bytes
	Entries   []string
}

// DataRequest defines the structure of the data request
type DataRequest struct {
	ChannelID common.ChannelIDEnum
	Checksum  common.Bytes
	Entries   []string
}

// DataResponse defines the structure of the data response
type DataResponse struct {
	ChannelID common.ChannelIDEnum
	Checksum  common.Bytes
	Payload   common.Bytes
}
