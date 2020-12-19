package types

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

var EmptyCodeHash = common.BytesToHash(crypto.Keccak256(nil))

type Account struct {
	Address                common.Address
	Sequence               uint64
	Balance                Coins
	ReservedFunds          []ReservedFund // TODO: replace the slice with map
	LastUpdatedBlockHeight uint64

	// Smart contract
	Root     common.Hash `json:"root"`      // merkle root of the storage trie
	CodeHash common.Hash `json:"code_hash"` // hash of the smart contract code
}

type AccountJSON struct {
	Sequence               common.JSONUint64 `json:"sequence"`
	Balance                Coins             `json:"coins"`
	ReservedFunds          []ReservedFund    `json:"reserved_funds"`
	LastUpdatedBlockHeight common.JSONUint64 `json:"last_updated_block_height"`
	Root                   common.Hash       `json:"root"`
	CodeHash               common.Hash       `json:"code"`
}

func NewAccountJSON(acc Account) AccountJSON {
	return AccountJSON{
		Sequence:               common.JSONUint64(acc.Sequence),
		Balance:                acc.Balance,
		ReservedFunds:          acc.ReservedFunds,
		LastUpdatedBlockHeight: common.JSONUint64(acc.LastUpdatedBlockHeight),
		Root:                   acc.Root,
		CodeHash:               acc.CodeHash,
	}
}

func (acc AccountJSON) Account() Account {
	return Account{
		Sequence:               uint64(acc.Sequence),
		Balance:                acc.Balance,
		ReservedFunds:          acc.ReservedFunds,
		LastUpdatedBlockHeight: uint64(acc.LastUpdatedBlockHeight),
		Root:                   acc.Root,
		CodeHash:               acc.CodeHash,
	}
}

func (acc Account) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewAccountJSON(acc))
}

func (acc *Account) UnmarshalJSON(data []byte) error {
	var a AccountJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*acc = a.Account()
	return nil
}

func NewAccount(address common.Address) *Account {
	return &Account{
		Address:  address,
		Root:     common.Hash{},
		CodeHash: EmptyCodeHash,
		Balance:  NewCoins(0, 0),
	}
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
		acc.Address, acc.Sequence, acc.Balance, acc.ReservedFunds)
}

// IsASmartContract indicates if the account is a smart contract account
func (acc *Account) IsASmartContract() bool {
	// Note: a suicided smart contract (i.e. account.CodeHash == core.SuicidedCodeHash)
	//       is still considered as a smart contract account
	return acc.CodeHash != EmptyCodeHash
}

// CheckReserveFund verifies inputs for ReserveFund.
func (acc *Account) CheckReserveFund(collateral Coins, fund Coins, duration uint64, reserveSequence uint64) error {
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
func (acc *Account) ReserveFund(collateral Coins, fund Coins, resourceIDs []string, endBlockHeight uint64, reserveSequence uint64) {
	newReservedFund := ReservedFund{
		Collateral:      collateral.NoNil(),
		InitialFund:     fund.NoNil(),
		UsedFund:        NewCoins(0, 0),
		ResourceIDs:     resourceIDs,
		EndBlockHeight:  endBlockHeight,
		ReserveSequence: reserveSequence,
	}
	acc.ReservedFunds = append(acc.ReservedFunds, newReservedFund)
	acc.Balance = acc.Balance.Minus(collateral).Minus(fund)
}

// ReleaseExpiredFunds releases all expired funds
func (acc *Account) ReleaseExpiredFunds(currentBlockHeight uint64) {
	newReservedFunds := []ReservedFund{}
	for _, reservedFund := range acc.ReservedFunds {
		minimumReleaseBlockHeight := calcMinimumReleaseBlockHeight(&reservedFund)
		if minimumReleaseBlockHeight > currentBlockHeight {
			newReservedFunds = append(newReservedFunds, reservedFund)
			continue
		}
		remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
		if !remainingFund.IsNonnegative() {
			remainingFund = NewCoins(0, 0) // Should NOT happen, just to be on the safe side
		}
		acc.Balance = acc.Balance.Plus(remainingFund).Plus(reservedFund.Collateral)
	}
	acc.ReservedFunds = newReservedFunds
}

// CheckReleaseFund verifies inputs for ReleaseFund
func (acc *Account) CheckReleaseFund(currentBlockHeight uint64, reserveSequence uint64) error {
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

func calcMinimumReleaseBlockHeight(reservedFund *ReservedFund) uint64 {
	// The "Freeze Period" is to ensure that in the event of overspending, the slashTx and the
	// releaseFundTx are NOT included in the same block. Otherwise the releaseFundTx may be
	// executed before the slashTx, and the overspender can escape from the punishment
	minimumReleaseBlockHeight := reservedFund.EndBlockHeight + ReservedFundFreezePeriodDuration
	return minimumReleaseBlockHeight
}

// ReleaseFund releases the fund reserved for service payment
func (acc *Account) ReleaseFund(currentBlockHeight uint64, reserveSequence uint64) {
	for idx, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		remainingFund := reservedFund.InitialFund.Minus(reservedFund.UsedFund)
		if !remainingFund.IsNonnegative() {
			remainingFund = NewCoins(0, 0) // Should NOT happen, just to be on the safe side
		}
		acc.Balance = acc.Balance.Plus(remainingFund).Plus(reservedFund.Collateral)
		acc.ReservedFunds = append(acc.ReservedFunds[:idx], acc.ReservedFunds[idx+1:]...)
	}
}

// CheckTransferReservedFund verifies inputs for SplitReservedFund
func (acc *Account) CheckTransferReservedFund(tgtAcc *Account, transferAmount Coins, paymentSequence uint64, currentBlockHeight uint64, reserveSequence uint64) error {
	for _, reservedFund := range acc.ReservedFunds {
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		if reservedFund.EndBlockHeight < currentBlockHeight {
			return errors.New("Already expired")
		}

		targetAddress := tgtAcc.Address
		err := reservedFund.VerifyPaymentSequence(targetAddress, paymentSequence)
		if err != nil {
			return err
		}

		return nil // at most one matching reserveSequence
	}
	return errors.Errorf("No matching ReservedFund with reserveSequence %d", reserveSequence)
}

// TransferReservedFund transfers the specified amount of reserved fund to the accounts participated in the payment split, and send remainder back to the source account (i.e. the acount itself)
func (acc *Account) TransferReservedFund(splittedCoinsMap map[*Account]Coins, currentBlockHeight uint64,
	reserveSequence uint64, servicePaymentTx *ServicePaymentTx) (shouldSlash bool, slashIntent SlashIntent) {
	for idx := range acc.ReservedFunds {
		reservedFund := &acc.ReservedFunds[idx]
		if reservedFund.ReserveSequence != reserveSequence {
			continue
		}

		resourceID := servicePaymentTx.ResourceID
		if !reservedFund.HasResourceID(resourceID) {
			continue
		}

		totalTransferAmount := NewCoins(0, 0)
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

	slashIntent := SlashIntent{
		Address:         acc.Address,
		ReserveSequence: reservedFund.ReserveSequence,
		Proof:           overspendingProof,
	}

	return slashIntent
}

func (acc *Account) UpdateToHeight(height uint64) {
	//	acc.UpdateAccountTFuelReward(height) // Initial TFuel inflation should be zero for all accounts
	acc.ReleaseExpiredFunds(height)
}

// func (acc *Account) UpdateAccountTFuelReward(currentBlockHeight uint64) {
// 	if acc.LastUpdatedBlockHeight < 0 || acc.LastUpdatedBlockHeight > currentBlockHeight {
// 		panic(fmt.Sprintf("Invalid LastRewardedBlockHeight: acc.LastUpdatedBlockHeight: %d, currentBlockHeight: %d", acc.LastUpdatedBlockHeight, currentBlockHeight))
// 	}

// 	totalThetaWei := acc.Balance.ThetaWei
// 	if totalThetaWei == nil {
// 		totalThetaWei = big.NewInt(0)
// 	}
// 	span := currentBlockHeight - acc.LastUpdatedBlockHeight

// 	newTFuelBalance := big.NewInt(int64(span))
// 	newTFuelBalance.Mul(newTFuelBalance, totalThetaWei)
// 	newTFuelBalance.Mul(newTFuelBalance, big.NewInt(RegularTFuelGenerationRateNumerator))
// 	newTFuelBalance.Div(newTFuelBalance, big.NewInt(RegularTFuelGenerationRateDenominator))

// 	if newTFuelBalance.Sign() <= 0 {
// 		// Underflow, no reward to add yet
// 		return
// 	}

// 	newTFuelBalance.Add(newTFuelBalance, acc.Balance.TFuelWei)

// 	if !newTFuelBalance.IsInt64() {
// 		panic("Account TFuel balance will overflow")
// 	}

// 	newBalance := Coins{
// 		ThetaWei: acc.Balance.ThetaWei,
// 		TFuelWei: newTFuelBalance,
// 	}
// 	acc.Balance = newBalance
// 	acc.LastUpdatedBlockHeight = currentBlockHeight
// }

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
