package types

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/thetatoken/theta/common"
)

type TransferRecord struct {
	ServicePayment ServicePaymentTx `json:"service_payment"`
}

type ReservedFund struct {
	Collateral      Coins
	InitialFund     Coins
	UsedFund        Coins
	ResourceIDs     []string // List of resource ID
	EndBlockHeight  uint64
	ReserveSequence uint64           // sequence number of the corresponding ReserveFundTx transaction
	TransferRecords []TransferRecord // signed ServerPaymentTransactions
}

type ReservedFundJSON struct {
	Collateral      Coins             `json:"collateral"`
	InitialFund     Coins             `json:"initial_fund"`
	UsedFund        Coins             `json:"used_fund"`
	ResourceIDs     []string          `json:"resource_ids"` // List of resource ID
	EndBlockHeight  common.JSONUint64 `json:"end_block_height"`
	ReserveSequence common.JSONUint64 `json:"reserve_sequence"` // sequence number of the corresponding ReserveFundTx transaction
	TransferRecords []TransferRecord  `json:"transfer_records"` // signed ServerPaymentTransactions
}

func NewReservedFundJSON(resv ReservedFund) ReservedFundJSON {
	return ReservedFundJSON{
		Collateral:      resv.Collateral,
		InitialFund:     resv.InitialFund,
		UsedFund:        resv.UsedFund,
		ResourceIDs:     resv.ResourceIDs,
		EndBlockHeight:  common.JSONUint64(resv.EndBlockHeight),
		ReserveSequence: common.JSONUint64(resv.ReserveSequence),
		TransferRecords: resv.TransferRecords,
	}
}

func (resv ReservedFundJSON) ReservedFund() ReservedFund {
	return ReservedFund{
		Collateral:      resv.Collateral,
		InitialFund:     resv.InitialFund,
		UsedFund:        resv.UsedFund,
		ResourceIDs:     resv.ResourceIDs,
		EndBlockHeight:  uint64(resv.EndBlockHeight),
		ReserveSequence: uint64(resv.ReserveSequence),
		TransferRecords: resv.TransferRecords,
	}
}

func (resv ReservedFund) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewReservedFundJSON(resv))
}

func (resv *ReservedFund) UnmarshalJSON(data []byte) error {
	var a ReservedFundJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*resv = a.ReservedFund()
	return nil
}

// TODO: this implementation is not very efficient
func (reservedFund *ReservedFund) VerifyPaymentSequence(targetAddress common.Address, paymentSequence uint64) error {
	currentPaymentSequence := uint64(0)
	for _, transferRecord := range reservedFund.TransferRecords {
		transferRecordTargetAddr := transferRecord.ServicePayment.Target.Address
		if targetAddress == transferRecordTargetAddr {
			if transferRecord.ServicePayment.PaymentSequence >= currentPaymentSequence {
				currentPaymentSequence = transferRecord.ServicePayment.PaymentSequence
			}
		}
	}
	if paymentSequence <= currentPaymentSequence {
		return errors.Errorf("Invalid payment sequence for address %X: %d, expected at least %d",
			targetAddress, paymentSequence, currentPaymentSequence+1)
	}
	return nil
}

func (reservedFund *ReservedFund) RecordTransfer(serverPaymentTx *ServicePaymentTx) {
	transferRecord := TransferRecord{
		ServicePayment: *serverPaymentTx,
	}

	reservedFund.TransferRecords = append(reservedFund.TransferRecords, transferRecord)
}

func (reservedFund *ReservedFund) HasResourceID(resourceID string) bool {
	for _, rid := range reservedFund.ResourceIDs {
		if strings.Compare(rid, resourceID) == 0 {
			return true
		}
	}
	return false
}
