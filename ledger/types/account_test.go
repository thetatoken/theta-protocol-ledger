package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeAccount(secret string, balance Coins) Account {
	privAcc := MakeAcc(secret)
	acc := privAcc.Account
	acc.Balance = balance

	return acc
}

func makeAccountAndReserveFund(initialBalance Coins, collateral Coins, fund Coins, resourceId []byte, endBlockHeight uint32, reserveSequence int) Account {
	acc := makeAccount("srcAcc", initialBalance)
	resourceIds := [][]byte{resourceId}
	acc.ReserveFund(collateral, fund, resourceIds, endBlockHeight, reserveSequence)

	return acc
}

func prepareForTransferReservedFund() (Account, Account, Account, Account, ServicePaymentTx, int) {
	srcAccInitialBalance := Coins{
		Coin{"ThetaWei", 1000},
		Coin{"GammaWei", 20000},
	}
	srcAccCollateral := Coins{Coin{"GammaWei", 1001}}
	srcAccFund := Coins{{"GammaWei", 1000}}
	resourceId := []byte("rid001")
	endBlockHeight := uint32(199)
	reserveSequence := 1

	srcAcc := makeAccountAndReserveFund(srcAccInitialBalance,
		srcAccCollateral, srcAccFund, resourceId, endBlockHeight, reserveSequence)

	tgtAcc := makeAccount("tgtAcc", Coins{})
	splitAcc1 := makeAccount("splitAcc1", Coins{})
	splitAcc2 := makeAccount("splitAcc2", Coins{})

	servicePaymentTx := ServicePaymentTx{
		ResourceId: resourceId,
	}

	return srcAcc, tgtAcc, splitAcc1, splitAcc2, servicePaymentTx, reserveSequence
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
	initialBalance := Coins{
		Coin{"ThetaWei", 1000},
		Coin{"GammaWei", 20000},
	}
	collateral := Coins{Coin{"GammaWei", 101}}
	fund := Coins{{"GammaWei", 100}}
	resourceId := []byte("rid001")

	acc := makeAccountAndReserveFund(initialBalance, collateral, fund, resourceId, 199, 1)
	assert.Equal(t, acc.Balance.Plus(collateral).Plus(fund), initialBalance)
}

func TestReleaseExpiredFunds(t *testing.T) {
	initialBalance := Coins{
		{"ThetaWei", 1000},
		{"GammaWei", 20000},
	}
	collateral := Coins{{"GammaWei", 101}}
	fund := Coins{{"GammaWei", 100}}
	resourceIds := [][]byte{[]byte("rid001")}

	acc := makeAccount("foo", initialBalance)
	acc.ReserveFund(collateral, fund, resourceIds, 10, 1)
	acc.ReserveFund(collateral, fund, resourceIds, 20, 2)
	acc.ReserveFund(collateral, fund, resourceIds, 30, 3)

	acc.ReleaseExpiredFunds(20) // only the first ReservedFund can be released
	assert.Equal(t, 2, len(acc.ReservedFunds))

	acc.ReleaseExpiredFunds(20 + ReservedFundFreezePeriodDuration) // the first and the second ReservedFund are released
	assert.Equal(t, 1, len(acc.ReservedFunds))
}

func TestCheckReleaseFund(t *testing.T) {
	initialBalance := Coins{
		Coin{"ThetaWei", 1000},
		Coin{"GammaWei", 20000},
	}
	collateral := Coins{Coin{"GammaWei", 101}}
	fund := Coins{{"GammaWei", 100}}
	resourceId := []byte("rid001")
	endBlockHeight := uint32(199)
	reserveSequence := 1

	acc := makeAccountAndReserveFund(initialBalance, collateral, fund, resourceId, endBlockHeight, reserveSequence)

	currentBlockHeight := uint32(80)
	if acc.CheckReleaseFund(currentBlockHeight, reserveSequence) == nil {
		acc.ReleaseFund(currentBlockHeight, reserveSequence)
	}
	assert.Equal(t, 1, len(acc.ReservedFunds)) // should not be able to release since currentBlockHeight < endBlockHeight

	currentBlockHeight = uint32(234)
	anotherReserveSequence := 2
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
	coinsMap[&splitAcc1] = Coins{Coin{"GammaWei", 234}}
	totalTransferAmount := Coins{}
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}

	paymentSequence := 1
	currentBlockHeight := uint32(900)
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
	coinsMap[&splitAcc1] = Coins{Coin{"GammaWei", 234}}
	totalTransferAmount := Coins{}
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}

	paymentSequence := 1
	reserveSequence2 := 2
	currentBlockHeight := uint32(100)
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
	coinsMap[&splitAcc1] = Coins{Coin{"GammaWei", 234}}
	coinsMap[&splitAcc2] = Coins{Coin{"GammaWei", 5945}}
	totalTransferAmount := Coins{}
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}
	currentBlockHeight := uint32(100)
	paymentSequence := 1

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
	coinsMap[&splitAcc1] = Coins{Coin{"GammaWei", 234}}
	coinsMap[&splitAcc2] = Coins{Coin{"GammaWei", 127}}
	coinsMap[&tgtAcc] = Coins{Coin{"GammaWei", 100}}
	totalTransferAmount := Coins{}
	for _, coins := range coinsMap {
		totalTransferAmount = totalTransferAmount.Plus(coins)
	}
	paymentSequence := 1
	currentBlockHeight := uint32(100)
	err := srcAcc.CheckTransferReservedFund(&tgtAcc, totalTransferAmount, paymentSequence, currentBlockHeight, reserveSequence)
	shouldSlash := false
	if err == nil {
		shouldSlash, _ = srcAcc.TransferReservedFund(coinsMap, currentBlockHeight, reserveSequence, &servicePaymentTx)
	}

	assert.Equal(t, nil, err)
	assert.False(t, shouldSlash)
	assert.Equal(t, Coins{Coin{"GammaWei", 234}}, splitAcc1.Balance)
	assert.Equal(t, Coins{Coin{"GammaWei", 127}}, splitAcc2.Balance)

	remainderAmount := totalTransferAmount.Minus(coinsMap[&splitAcc1]).Minus(coinsMap[&splitAcc2])
	assert.Equal(t, remainderAmount, tgtAcc.Balance)
	assert.Equal(t, totalTransferAmount, srcAcc.ReservedFunds[0].UsedFund)
}

func TestUpdateAccountGammaReward(t *testing.T) {
	assert := assert.New(t)

	var acc *Account
	currentBlockHeight := uint32(1000)

	// Should update account
	acc = &Account{
		LastUpdatedBlockHeight: 1,
		Balance: Coins{{
			Denom:  "ThetaWei",
			Amount: 1e12,
		}, {
			Denom:  "GammaWei",
			Amount: 2000,
		}}}

	acc.UpdateAccountGammaReward(currentBlockHeight)
	assert.Equal(int64(1e12), acc.Balance.GetThetaWei().Amount)
	assert.Equal(int64(189812000), acc.Balance.GetGammaWei().Amount)
	assert.Equal(uint32(1000), acc.LastUpdatedBlockHeight)

	// Underflow: Should not update account if reward is less than 1 Gamma
	acc = &Account{
		LastUpdatedBlockHeight: 1,
		Balance: Coins{{
			Denom:  "ThetaWei",
			Amount: 1000,
		}, {
			Denom:  "GammaWei",
			Amount: 2000,
		}}}

	acc.UpdateAccountGammaReward(currentBlockHeight)
	assert.Equal(int64(1000), acc.Balance.GetThetaWei().Amount)
	assert.Equal(int64(2000), acc.Balance.GetGammaWei().Amount)
	assert.Equal(uint32(1), acc.LastUpdatedBlockHeight)

	// Should not overflow for large span * balance
	currentBlockHeight = 1e7
	acc = &Account{
		LastUpdatedBlockHeight: 1,
		Balance: Coins{{
			Denom:  "ThetaWei",
			Amount: 1e12,
		}, {
			Denom:  "GammaWei",
			Amount: 2000,
		}}}

	acc.UpdateAccountGammaReward(currentBlockHeight)
	assert.Equal(int64(1e12), acc.Balance.GetThetaWei().Amount)
	assert.Equal(int64(1899999812000), acc.Balance.GetGammaWei().Amount)
	assert.Equal(uint32(1e7), acc.LastUpdatedBlockHeight)

	// Should panic if the end balance oveflow
	currentBlockHeight = 9e8
	acc = &Account{
		LastUpdatedBlockHeight: 1,
		Balance: Coins{{
			Denom:  "ThetaWei",
			Amount: 1e18,
		}, {
			Denom:  "GammaWei",
			Amount: 2000,
		}}}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	acc.UpdateAccountGammaReward(currentBlockHeight) // Should panic
}
