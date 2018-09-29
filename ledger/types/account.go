package types

import (
	"fmt"
	"math/big"

	"github.com/pkg/errors"
	"github.com/thetatoken/ukulele/crypto"
)

type Account struct {
	PubKey                 *crypto.PublicKey `json:"pub_key"` // May be nil, if not known.
	Sequence               int               `json:"sequence"`
	Balance                Coins             `json:"coins"`
	ReservedFunds          []ReservedFund    `json:"reserved_funds"` // TODO: replace the slice with map
	LastUpdatedBlockHeight uint32            `json:"last_updated_block_height"`
}

func (acc *Account) Copy() *Account {
	if acc == nil {
		return nil
	}
	accCopy := *acc
	return &accCopy
}

func (acc *Account) String() string {
	if acc == nil {
		return "nil-Account"
	}
	return fmt.Sprintf("Account{%v %v %v %v}",
		acc.PubKey, acc.Sequence, acc.Balance, acc.ReservedFunds)
}

// CheckReserveFund verifies inputs for ReserveFund.
func (acc *Account) CheckReserveFund(collateral Coins, fund Coins, duration uint32, reserveSequence int) error {
	if duration < MinimumFundReserveDuration || duration > MaximumFundReserveDuration {
		return errors.New("Duration is out of permitted range")
	}

	if !collateral.IsValid() || !collateral.IsNonnegative() {
		return errors.New("Invalid collateral")
	}

	if !fund.IsValid() || !fund.IsNonnegative() {
		return errors.New("Invalid fund")
	}

	minimalBalance := collateral.Plus(fund)
	if !acc.Balance.IsGTE(minimalBalance) {
		return errors.New("Not enough balance")
	}

	if !collateral.Minus(fund).IsPositive() {
		return errors.New("Collateral should be strictly greater than the fund")
	}

	for _, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence >= reserveSequence {
			return errors.New("ReserveSequence should be strictly increasing")
		}
	}

	return nil
}

// ReserveFund reserves the given amount of fund for subsequence service payments
func (acc *Account) ReserveFund(collateral Coins, fund Coins, resourceIds [][]byte, endBlockHeight uint32, reserveSequence int) {
	newReservedFund := ReservedFund{
		Collateral:      collateral,
		InitialFund:     fund,
		UsedFund:        Coins{},
		ResourceIds:     resourceIds,
		EndBlockHeight:  endBlockHeight,
		ReserveSequence: reserveSequence,
	}
	acc.ReservedFunds = append(acc.ReservedFunds, newReservedFund)
	acc.Balance = acc.Balance.Minus(collateral).Minus(fund)
}

// ReleaseExpiredFunds releases all expired funds
func (acc *Account) ReleaseExpiredFunds(currentBlockHeight uint32) {
	newReservedFunds := []ReservedFund{}
	for _, reservedFund := range acc.ReservedFunds {
		minimumReleaseBlockHeight := calcMinimumReleaseBlockHeight(&reservedFund)
		if minimumReleaseBlockHeight > currentBlockHeight {
			newReservedFunds = append(newReservedFunds, reservedFund)
			continue
		}
		remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
		if !remainingFund.IsNonnegative() {
			remainingFund = Coins{} // Should NOT happen, just to be on the safe side
		}
		acc.Balance = acc.Balance.Plus(remainingFund).Plus(reservedFund.Collateral)
	}
	acc.ReservedFunds = newReservedFunds
}

// CheckReleaseFund verifies inputs for ReleaseFund
func (acc *Account) CheckReleaseFund(currentBlockHeight uint32, reserveSequence int) error {
	for _, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		minimumReleaseBlockHeight := calcMinimumReleaseBlockHeight(&reservedFund)
		if minimumReleaseBlockHeight > currentBlockHeight {
			return errors.Errorf("Fund cannot be released until blockheight %d", minimumReleaseBlockHeight) // cannot release yet
		}
		return nil // at most one matching reserveSequence
	}

	return errors.Errorf("No matching ReserveSequence")
}

func calcMinimumReleaseBlockHeight(reservedFund *ReservedFund) uint32 {
	// The "Freeze Period" is to ensure that in the event of overspending, the slashTx and the
	// releaseFundTx are NOT included in the same block. Otherwise the releaseFundTx may be
	// executed before the slashTx, and the overspender can escape from the punishment
	minimumReleaseBlockHeight := reservedFund.EndBlockHeight + ReservedFundFreezePeriodDuration
	return minimumReleaseBlockHeight
}

// ReleaseFund releases the fund reserved for service payment
func (acc *Account) ReleaseFund(currentBlockHeight uint32, reserveSequence int) {
	for idx, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
		if !remainingFund.IsNonnegative() {
			remainingFund = Coins{} // Should NOT happen, just to be on the safe side
		}
		acc.Balance = acc.Balance.Plus(remainingFund).Plus(reservedFund.Collateral)
		acc.ReservedFunds = append(acc.ReservedFunds[:idx], acc.ReservedFunds[idx+1:]...)
	}
}

// CheckTransferReservedFund verifies inputs for SplitReservedFund
func (acc *Account) CheckTransferReservedFund(tgtAcc *Account, transferAmount Coins, paymentSequence int, currentBlockHeight uint32, reserveSequence int) error {
	for _, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		if reservedFund.EndBlockHeight < currentBlockHeight {
			return errors.New("Already expired")
		}

		targetAddress := tgtAcc.PubKey.Address()
		err := reservedFund.VerifyPaymentSequence(targetAddress, paymentSequence)
		if err != nil {
			return err
		}

		return nil // at most one matching reserveSequence
	}
	return errors.Errorf("No matching ReservedFund with reserveSequence %d", reserveSequence)
}

// TransferReservedFund transfers the specified amount of reserved fund to the accounts participated in the payment split, and send remainder back to the source account (i.e. the acount itself)
func (acc *Account) TransferReservedFund(splittedCoinsMap map[*Account]Coins, currentBlockHeight uint32,
	reserveSequence int, servicePaymentTx *ServicePaymentTx) (shouldSlash bool, slashIntent SlashIntent) {
	for idx := range acc.ReservedFunds {
		reservedFund := &acc.ReservedFunds[idx]
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		resourceId := servicePaymentTx.ResourceId
		if !reservedFund.HasResourceId(resourceId) {
			continue
		}

		totalTransferAmount := Coins{}
		for _, coinsSplit := range splittedCoinsMap {
			totalTransferAmount = totalTransferAmount.Plus(coinsSplit)
		}

		remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
		if !remainingFund.IsGTE(totalTransferAmount) {
			slashIntent = acc.generateSlashIntent(reservedFund, servicePaymentTx)
			return true, slashIntent
		}

		reservedFund.UsedFund = reservedFund.UsedFund.Plus(totalTransferAmount)
		for account, coinsSplit := range splittedCoinsMap {
			account.Balance = account.Balance.Plus(coinsSplit)
		}

		reservedFund.RecordTransfer(servicePaymentTx)

		return false, SlashIntent{} // at most one matching reserveSequence
	}

	return false, SlashIntent{}
}

func (acc *Account) generateSlashIntent(reservedFund *ReservedFund, currentServicePaymentTx *ServicePaymentTx) SlashIntent {
	overspendingProof := constructOverspendingProof(reservedFund, currentServicePaymentTx)

	if acc.PubKey.IsEmpty() {
		panic("Account PubKey is empty!")
	}

	slashIntent := SlashIntent{
		Address:         acc.PubKey.Address(),
		ReserveSequence: reservedFund.ReserveSequence,
		Proof:           overspendingProof,
	}

	return slashIntent
}

func (acc *Account) UpdateToHeight(height uint32) {
	acc.UpdateAccountGammaReward(height)
	acc.ReleaseExpiredFunds(height)
}

func (acc *Account) UpdateAccountGammaReward(currentBlockHeight uint32) {
	if acc.LastUpdatedBlockHeight <= 0 || acc.LastUpdatedBlockHeight > currentBlockHeight {
		panic(fmt.Sprintf("Invalid LastRewardedBlockHeight: acc.LastUpdatedBlockHeight: %d, currentBlockHeight: %d", acc.LastUpdatedBlockHeight, currentBlockHeight))
	}

	totalThetaWei := acc.Balance.GetThetaWei().Amount
	span := currentBlockHeight - acc.LastUpdatedBlockHeight

	newGammaBalance := big.NewInt(int64(span))
	newGammaBalance.Mul(newGammaBalance, big.NewInt(totalThetaWei))
	newGammaBalance.Mul(newGammaBalance, big.NewInt(RegularGammaGenerationRateNumerator))
	newGammaBalance.Div(newGammaBalance, big.NewInt(RegularGammaGenerationRateDenominator))

	if newGammaBalance.Sign() <= 0 {
		// Underflow, no reward to add yet
		return
	}

	newGammaBalance.Add(newGammaBalance, big.NewInt(acc.Balance.GetGammaWei().Amount))

	if !newGammaBalance.IsInt64() {
		panic("Account Gamma balance will overflow")
	}

	newBalance := Coins{{
		Denom:  DenomThetaWei,
		Amount: acc.Balance.GetThetaWei().Amount,
	}, {
		Denom:  DenomGammaWei,
		Amount: newGammaBalance.Int64(),
	}}
	acc.Balance = newBalance
	acc.LastUpdatedBlockHeight = currentBlockHeight
}

func constructOverspendingProof(reservedFund *ReservedFund, currentServicePaymentTx *ServicePaymentTx) []byte {
	// TODO: The proof can be simplied to only contain the signed transactions that go beyond the spending limit.
	//       Combined with all the committed transactions of the same reserved pool, it can prove that
	//       the sender has overspent the deposit

	overspendingProof := OverspendingProof{}
	overspendingProof.ReserveSequence = reservedFund.ReserveSequence

	for _, transferRecord := range reservedFund.TransferRecords {
		overspendingProof.ServicePayments = append(overspendingProof.ServicePayments, transferRecord.ServicePayment)
	}
	overspendingProof.ServicePayments = append(overspendingProof.ServicePayments, *currentServicePaymentTx)
	overspendingProofBytes, _ := ToBytes(&overspendingProof)
	return overspendingProofBytes
}
