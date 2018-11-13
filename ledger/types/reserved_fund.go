package types

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/thetatoken/ukulele/common"
)

type TransferRecord struct {
	ServicePayment ServicePaymentTx `json:"service_payment"`
}

type ReservedFund struct {
	Collateral      Coins            `json:"collateral"`
	InitialFund     Coins            `json:"initial_fund"`
	UsedFund        Coins            `json:"used_fund"`
	ResourceIDs     []string         `json:"resource_ids"` // List of resource ID
	EndBlockHeight  uint64           `json:"end_block_height"`
	ReserveSequence uint64           `json:"reserve_sequence"` // sequence number of the corresponding ReserveFundTx transaction
	TransferRecords []TransferRecord `json:"transfer_records"` // signed ServerPaymentTransactions
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
