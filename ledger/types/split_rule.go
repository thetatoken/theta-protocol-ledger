package types

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/common"
)

// ** Split Rule: Specifies the payment split agreement among participating addresses **
//

// Split contains the particiated address and percentage of the payment the address should get
type Split struct {
	Address    common.Address // Address to participate in the payment split
	Percentage uint           // An integer between 0 and 100, representing the percentage of the payment the address should get
}

func (s Split) JsonString() string {
	return fmt.Sprint("{\"Address\":%v, \"Percentage\": %v}", s.Address.JsonString(), s.Percentage)
}

// SplitRule specifies the payment split agreement among differet addresses
type SplitRule struct {
	InitiatorAddress common.Address // Address of the initiator
	ResourceID       string         // ResourceID of the payment to be split
	Splits           []Split        // Splits of the payments
	EndBlockHeight   uint64         // The block height when the split rule expires
}

type SplitRuleJSON struct {
	InitiatorAddress common.Address    `json:"initiator_address"` // Address of the initiator
	ResourceID       string            `json:"resource_id"`       // ResourceID of the payment to be split
	Splits           []Split           `json:"splits"`            // Splits of the payments
	EndBlockHeight   common.JSONUint64 `json:"end_block_height"`  // The block height when the split rule expires

}

func NewSplitRuleJSON(a SplitRule) SplitRuleJSON {
	return SplitRuleJSON{
		InitiatorAddress: a.InitiatorAddress,
		ResourceID:       a.ResourceID,
		Splits:           a.Splits,
		EndBlockHeight:   common.JSONUint64(a.EndBlockHeight),
	}
}

func (a SplitRuleJSON) SplitRule() SplitRule {
	return SplitRule{
		InitiatorAddress: a.InitiatorAddress,
		ResourceID:       a.ResourceID,
		Splits:           a.Splits,
		EndBlockHeight:   uint64(a.EndBlockHeight),
	}
}

func (a SplitRule) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewSplitRuleJSON(a))
}

func (a *SplitRule) UnmarshalJSON(data []byte) error {
	var b SplitRuleJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.SplitRule()
	return nil
}

func (sc *SplitRule) String() string {
	if sc == nil {
		return "nil-SlashIntent"
	}
	return fmt.Sprintf("SplitRule{%v %v %v %v}",
		sc.InitiatorAddress.Hex(), string(sc.ResourceID), sc.Splits, sc.EndBlockHeight)
}

func (sc *SplitRule) JsonString() string {
	if sc == nil {
		return "nil-SlashIntent"
	}
	return fmt.Sprintf("{\"InitiatorAddress\":%v, \"ResourceID\":\"%v\", \"Splits\":%v, \"EndBlockHeight\":%v}",
		sc.InitiatorAddress.JsonString(), sc.ResourceID, jsonSplits(sc.Splits), sc.EndBlockHeight)
}

func jsonSplits(splits []Split) string {
	str := "["
	for i, split := range splits {
		str += split.JsonString()
		if i < len(splits)-1 {
			str += ","
		}
	}
	str += "]"
	return str
}
