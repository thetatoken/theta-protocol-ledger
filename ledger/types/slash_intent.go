package types

import (
	"encoding/hex"
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

// SlashIntent contains the address, reserve sequence of the account to
// be slashed, and the proof why the account should be slashed
type SlashIntent struct {
	Address         common.Address
	ReserveSequence int
	Proof           []byte
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
	ReserveSequence int
	ServicePayments []ServicePaymentTx
}
