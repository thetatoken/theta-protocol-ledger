package types

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoins(t *testing.T) {
	assert := assert.New(t)

	//Define the coins to be used in tests
	good := NewCoins(1, 2)
	empty := Coins{}
	neg := good.Negative()

	assert.True(good.IsValid(), "Coins are valid")
	assert.True(good.IsPositive(), "Expected coins to be positive: %v", good)
	assert.True(good.IsGTE(empty), "Expected %v to be >= %v", good, empty)
	assert.False(neg.IsPositive(), "Expected neg coins to not be positive: %v", neg)
	assert.False(neg.IsValid(), "Expected coins to invalid: %v", neg)
}

//Test operations on invalid coins
func TestInvalidCoin(t *testing.T) {
	assert := assert.New(t)

	coinsA := NewCoins(123, 0)
	coinsEmpty := Coins{}

	ret1 := coinsA.Plus(coinsEmpty)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")
	ret1 = coinsEmpty.Plus(coinsA)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")

	// Result should be a copy
	ret1.ThetaWei = big.NewInt(456)
	assert.True(coinsA.ThetaWei.Cmp(big.NewInt(123)) == 0)
	assert.True(ret1.ThetaWei.Cmp(big.NewInt(456)) == 0)

	ret2 := coinsA.Minus(coinsEmpty)
	assert.True(coinsA.IsEqual(ret2), "Sum is correct")
	// Result should be a copy
	ret2.ThetaWei = big.NewInt(456)
	assert.True(coinsA.ThetaWei.Cmp(big.NewInt(123)) == 0)
	assert.True(ret2.ThetaWei.Cmp(big.NewInt(456)) == 0)
}
