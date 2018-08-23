package dispatcher

import (
	"github.com/thetatoken/ukulele/common"
)

// InventoryRequest defines the structure of the inventory request
type InventoryRequest struct {
	Type     common.SyncType
	Checksum common.Bytes
}

// InventoryResponse defines the structure of the inventory response
type InventoryResponse struct {
	Type     common.SyncType
	Checksum common.Bytes
	Entries  []string
}

// DataRequest defines the structure of the data request
type DataRequest struct {
	Type     common.SyncType
	Checksum common.Bytes
	Entries  []string
}

// DataResponse defines the structure of the data response
type DataResponse struct {
	Type     common.SyncType
	Checksum common.Bytes
	Payload  common.Bytes
}
