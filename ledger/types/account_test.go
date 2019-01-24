package types

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeAccount(secret string, balance Coins) Account {
	privAcc := MakeAcc(secret)
	acc := privAcc.Account
	acc.Balance = balance

	return acc
}

func makeAccountAndReserveFund(initialBalance Coins, collateral Coins, fund Coins, resourceID string, endBlockHeight uint64, reserveSequence uint64) Account {
	acc := makeAccount("srcAcc", initialBalance)
	resourceIDs := []string{resourceID}
	acc.ReserveFund(collateral, fund, resourceIDs, endBlockHeight, reserveSequence)

	return acc
}

func prepareForTransferReservedFund() (Account, Account, Account, Account, ServicePaymentTx, uint64) {
	srcAccInitialBalance := NewCoins(1000, 20000)
	srcAccCollateral := NewCoins(0, 1001)
	srcAccFund := NewCoins(0, 1000)
	resourceID := "rid001"
	endBlockHeight := uint64(199)
	reserveSequence := uint64(1)
	srcAcc := makeAccountAndReserveFund(srcAccInitialBalance,
		srcAccCollateral, srcAccFund, resourceID, endBlockHeight, reserveSequence)

	tgtAcc := makeAccount("tgtAcc", NewCoins(0, 0))
	splitAcc1 := makeAccount("splitAcc1", NewCoins(0, 0))
	splitAcc2 := makeAccount("splitAcc2", NewCoins(0, 0))

	servicePaymentTx := ServicePaymentTx{
		ResourceID: resourceID,
	}

	return srcAcc, tgtAcc, splitAcc1, splitAcc2, servicePaymentTx, reserveSequence
}

func TestAccountJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	acc := Account{
		Sequence: math.MaxUint64,
		Balance:  NewCoins(456, 789),
	}

	s, err := json.Marshal(acc)
	require.Nil(err)
	var acc1 Account
	err = json.Unmarshal(s, &acc1)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), acc1.Sequence)
}

func TestNilAccount(t *testing.T) {

	var acc Account

	//test Copy
	accCopy := acc.Copy()
	//note that the assert.True is used instead of assert.Equal because looking at pointers
	assert.True(t, &acc != accCopy, "Account Copy Error, acc1: %v, acc2: %v", &acc, accCopy)
	assert.Equal(t, acc.Sequence, accCopy.Sequence)

	//test sending nils for panic
	var nilAcc *Account
	nilAcc.String()
	nilAcc.Copy()
}

func TestReserveFund(t *testing.T) {
	initialBalance := NewCoins(1000, 20000)
	collateral := NewCoins(0, 101)
	fund := NewCoins(0, 100)
	resourceID := "rid001"

	acc := makeAccountAndReserveFund(initialBalance, collateral, fund, resourceID, 199, 1)
	assert.Equal(t, acc.Balance.Plus(collateral).Plus(fund), initialBalance)
}

func TestReleaseExpiredFunds(t *testing.T) {
	initialBalance := NewCoins(1000, 20000)
	collateral := NewCoins(0, 101)
	fund := NewCoins(0, 100)
	resourceIDs := []string{"rid001"}

	acc := makeAccount("foo", initialBalance)
	acc.ReserveFund(collateral, fund, resourceIDs, 10, 1)
	acc.ReserveFund(collateral, fund, resourceIDs, 20, 2)
	acc.ReserveFund(collateral, fund, resourceIDs, 30, 3)

	acc.ReleaseExpiredFunds(20) // only the first ReservedFund can be released
	assert.Equal(t, 2, len(acc.ReservedFunds))

	acc.ReleaseExpiredFunds(20 + ReservedFundFreezePeriodDuration) // the first and the second ReservedFund are released
	assert.Equal(t, 1, len(acc.ReservedFunds))
}

func TestCheckReleaseFund(t *testing.T) {
	initialBalance := NewCoins(1000, 20000)
	collateral := NewCoins(0, 101)
	fund := NewCoins(0, 100)
	resourceID := "rid001"
	endBlockHeight := uint64(199)
	reserveSequence := uint64(1)

	acc := makeAccountAndReserveFund(initialBalance, collateral, fund, resourceID, endBlockHeight, reserveSequence)

	currentBlockHeight := uint64(80)
	if acc.CheckReleaseFund(currentBlockHeight, reserveSequence) == nil {
		acc.ReleaseFund(currentBlockHeight, reserveSequence)
	}
	assert.Equal(t, 1, len(acc.ReservedFunds)) // should not be able to release since currentBlockHeight < endBlockHeight

	currentBlockHeight = uint64(234)
	anotherReserveSequence := uint64(2)
	if acc.CheckReleaseFund(currentBlockHeight, reserveSequence) == nil {
		acc.ReleaseFund(currentBlockHeight, anotherReserveSequence)
	}
	assert.Equal(t, 1, len(acc.ReservedFunds)) // should not be able to release since the reserve sequence does not match

	if acc.CheckReleaseFund(currentBlockHeight, reserveSequence) == nil {
		acc.ReleaseFund(currentBlockHeight, reserveSequence)
	}
	assert.Equal(t, 0, len(acc.ReservedFunds))
}

// Test 1: currentBlockHeight > endBlockHeight
func TestTransferReservedFund1(t *testing.T) {
	srcAcc, tgtAcc, splitAcc1, _, servicePaymentTx, reserveSequence := prepareForTransferReservedFund()

	coinsMap := make(map[*Account]Coins)
	coinsMap[&splitAcc1] = NewCoins(0, 234)
	totalTransferAmount := NewCoins(0, 0)
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}

	paymentSequence := uint64(1)
	currentBlockHeight := uint64(900)
	err := srcAcc.CheckTransferReservedFund(&tgtAcc, totalTransferAmount, paymentSequence, currentBlockHeight, reserveSequence)
	if err != nil {
		srcAcc.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, &servicePaymentTx)
	}
	assert.NotEqual(t, nil, err) // should error out since the currentBlockHeight > endBlockHeight
}

// Test 2: reserve sequence mismatch
func TestTransferReservedFund2(t *testing.T) {
	srcAcc, tgtAcc, splitAcc1, _, servicePaymentTx, reserveSequence := prepareForTransferReservedFund()

	coinsMap := make(map[*Account]Coins)
	coinsMap[&splitAcc1] = NewCoins(0, 234)
	totalTransferAmount := NewCoins(0, 0)
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}

	paymentSequence := uint64(1)
	reserveSequence2 := uint64(2)
	currentBlockHeight := uint64(100)
	err := srcAcc.CheckTransferReservedFund(&tgtAcc, totalTransferAmount, paymentSequence, currentBlockHeight, reserveSequence2)
	if err != nil {
		srcAcc.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, &servicePaymentTx)
	}
	assert.NotEqual(t, nil, err) // should error out since no matching reserve sequence is found
}

// Test 3: overspend
func TestTransferReservedFund3(t *testing.T) {
	srcAcc, tgtAcc, splitAcc1, splitAcc2, servicePaymentTx, reserveSequence := prepareForTransferReservedFund()

	coinsMap := make(map[*Account]Coins)
	coinsMap[&splitAcc1] = NewCoins(0, 234)
	coinsMap[&splitAcc2] = NewCoins(0, 5945)
	totalTransferAmount := NewCoins(0, 0)
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}
	currentBlockHeight := uint64(100)
	paymentSequence := uint64(1)

	err := srcAcc.CheckTransferReservedFund(&tgtAcc, totalTransferAmount, paymentSequence, currentBlockHeight, reserveSequence)
	shouldSlash := false
	if err == nil {
		shouldSlash, _ = srcAcc.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, &servicePaymentTx)
	}
	assert.Equal(t, nil, err)   // should be able to pass the check
	assert.True(t, shouldSlash) // overspend, should slash
}

// Test 4: normal spend, should pass
func TestTransferReservedFund4(t *testing.T) {
	srcAcc, tgtAcc, splitAcc1, splitAcc2, servicePaymentTx, reserveSequence := prepareForTransferReservedFund()

	coinsMap := make(map[*Account]Coins)
	coinsMap[&splitAcc1] = NewCoins(0, 234)
	coinsMap[&splitAcc2] = NewCoins(0, 127)
	coinsMap[&tgtAcc] = NewCoins(0, 100)
	totalTransferAmount := NewCoins(0, 0)
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}
	paymentSequence := uint64(1)
	currentBlockHeight := uint64(100)
	err := srcAcc.CheckTransferReservedFund(&tgtAcc, totalTransferAmount, paymentSequence, currentBlockHeight, reserveSequence)
	shouldSlash := false
	if err == nil {
		shouldSlash, _ = srcAcc.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, &servicePaymentTx)
	}

	assert.Equal(t, nil, err)
	assert.False(t, shouldSlash)
	assert.Equal(t, NewCoins(0, 234), splitAcc1.Balance)
	assert.Equal(t, NewCoins(0, 127), splitAcc2.Balance)

	remainderAmount := totalTransferAmount.Minus(coinsMap[&splitAcc1]).Minus(coinsMap[&splitAcc2])
	assert.Equal(t, remainderAmount, tgtAcc.Balance)
	assert.Equal(t, totalTransferAmount, srcAcc.ReservedFunds[0].UsedFund)
}

// For the initial Mainnet release, TFuel should not inflate
// func TestUpdateAccountTFuelReward(t *testing.T) {
// 	assert := assert.New(t)

// 	var acc *Account
// 	currentBlockHeight := uint64(1000)

// 	// Should update account
// 	acc = &Account{
// 		LastUpdatedBlockHeight: 1,
// 		Balance:                NewCoins(1e12, 2000),
// 	}

// 	acc.UpdateAccountTFuelReward(currentBlockHeight)
// 	assert.Equal(int64(1e12), acc.Balance.ThetaWei.Int64())
// 	assert.Equal(int64(189812000), acc.Balance.TFuelWei.Int64())
// 	assert.Equal(uint64(1000), acc.LastUpdatedBlockHeight)

// 	// Underflow: Should not update account if reward is less than 1 TFuel
// 	acc = &Account{
// 		LastUpdatedBlockHeight: 1,
// 		Balance:                NewCoins(1000, 2000),
// 	}

// 	acc.UpdateAccountTFuelReward(currentBlockHeight)
// 	assert.Equal(int64(1000), acc.Balance.ThetaWei.Int64())
// 	assert.Equal(int64(2000), acc.Balance.TFuelWei.Int64())
// 	assert.Equal(uint64(1), acc.LastUpdatedBlockHeight)

// 	// Should not overflow for large span * balance
// 	currentBlockHeight = 1e7
// 	acc = &Account{
// 		LastUpdatedBlockHeight: 1,
// 		Balance:                NewCoins(1e12, 2000),
// 	}

// 	acc.UpdateAccountTFuelReward(currentBlockHeight)
// 	assert.Equal(int64(1e12), acc.Balance.ThetaWei.Int64())
// 	assert.Equal(int64(1899999812000), acc.Balance.TFuelWei.Int64())
// 	assert.Equal(uint64(1e7), acc.LastUpdatedBlockHeight)

// 	// Should panic if the end balance oveflow
// 	currentBlockHeight = 9e8
// 	acc = &Account{
// 		LastUpdatedBlockHeight: 1,
// 		Balance:                NewCoins(1e18, 2000),
// 	}
// 	defer func() {
// 		if r := recover(); r == nil {
// 			t.Errorf("The code did not panic")
// 		}
// 	}()
// 	acc.UpdateAccountTFuelReward(currentBlockHeight) // Should panic
// }
