package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/common"
)

// SlashIntent contains the address, reserve sequence of the account to
// be slashed, and the proof why the account should be slashed
type SlashIntent struct {
	Address         common.Address
	ReserveSequence uint64
	Proof           common.Bytes
}

type SlashIntentJSON struct {
	Address         common.Address
	ReserveSequence common.JSONUint64
	Proof           common.Bytes
}

func NewSlashIntentJSON(s SlashIntent) SlashIntentJSON {
	return SlashIntentJSON{
		Address:         s.Address,
		ReserveSequence: common.JSONUint64(s.ReserveSequence),
		Proof:           s.Proof,
	}
}

func (s SlashIntentJSON) SlashIntent() SlashIntent {
	return SlashIntent{
		Address:         s.Address,
		ReserveSequence: uint64(s.ReserveSequence),
		Proof:           s.Proof,
	}
}

func (s SlashIntent) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewSlashIntentJSON(s))
}

func (s *SlashIntent) UnmarshalJSON(data []byte) error {
	var a SlashIntentJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = a.SlashIntent()
	return nil
}

func (si *SlashIntent) String() string {
	if si == nil {
		return "nil-SlashIntent"
	}
	return fmt.Sprintf("SlashIntent{%v %v %v}",
		si.Address, si.ReserveSequence, hex.EncodeToString(si.Proof))
}

// OverspendingProof contains the proof that the ReservedFund has been overly spent
type OverspendingProof struct {
	ReserveSequence uint64
	ServicePayments []ServicePaymentTx
}

type OverspendingProofJSON struct {
	ReserveSequence common.JSONUint64
	ServicePayments []ServicePaymentTx
}

func NewOverspendingProofJSON(a OverspendingProof) OverspendingProofJSON {
	return OverspendingProofJSON{
		ReserveSequence: common.JSONUint64(a.ReserveSequence),
		ServicePayments: a.ServicePayments,
	}
}

func (a OverspendingProofJSON) OverspendingProof() OverspendingProof {
	return OverspendingProof{
		ReserveSequence: uint64(a.ReserveSequence),
		ServicePayments: a.ServicePayments,
	}
}

func (a OverspendingProof) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewOverspendingProofJSON(a))
}

func (a *OverspendingProof) UnmarshalJSON(data []byte) error {
	var b OverspendingProofJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.OverspendingProof()
	return nil
}
