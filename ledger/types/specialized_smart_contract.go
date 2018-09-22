package types

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

//
// ----- Definition and Implementation of Various Specialized Smart Contracts ----- //
//

// ** Split Contract: Specifies the payment split agreement among participating addresses **
//

// Split contains the particiated address and percentage of the payment the address should get
type Split struct {
	Address    common.Address `json:"address"`    // Address to participate in the payment split
	Percentage uint           `json:"percentage"` // An integer between 0 and 100, representing the percentage of the payment the address should get
}

// SplitContract specifies the payment split agreement among differet addresses
type SplitContract struct {
	InitiatorAddress common.Address `json:"initiator_address"` // Address of the initiator
	ResourceId       common.Bytes   `json:"resource_id"`       // ResourceId of the payment to be split
	Splits           []Split        `json:"splits"`            // Splits of the payments
	EndBlockHeight   uint64         `json:"end_block_height"`  // The block height when the split contract expires
}

func (sc *SplitContract) String() string {
	if sc == nil {
		return "nil-SlashIntent"
	}
	return fmt.Sprintf("SplitContract{%v %v %v}",
		sc.ResourceId, sc.Splits, sc.EndBlockHeight)
}
