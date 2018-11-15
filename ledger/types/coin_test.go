package types

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoins(t *testing.T) {
	assert := assert.New(t)

	//Define the coins to be used in tests
	good := NewCoins(1, 2)
	empty := NewCoins(0, 0)
	neg := good.Negative()

	assert.True(good.IsValid(), "Coins are valid")
	assert.True(good.IsPositive(), "Expected coins to be positive: %v", good)
	assert.True(good.IsGTE(empty), "Expected %v to be >= %v", good, empty)
	assert.False(neg.IsPositive(), "Expected neg coins to not be positive: %v", neg)
	assert.False(neg.IsValid(), "Expected coins to invalid: %v", neg)

	a := NewCoins(3, 10)
	b := NewCoins(5, 15)
	assert.True(NewCoins(8, 25).IsEqual(a.Plus(b)))
}

//Test operations on invalid coins
func TestInvalidCoin(t *testing.T) {
	assert := assert.New(t)

	coinsA := NewCoins(123, 0)
	coinsEmpty := NewCoins(0, 0)

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

func TestNoNilException(t *testing.T) {
	assert := assert.New(t)

	coinsA := NewCoins(123, 456)
	coinsB := Coins{}
	coinsC := NewCoins(0, 0)

	// Should not have nil pointer exception.
	assert.True(coinsB.IsEqual(coinsC))
	assert.True(coinsB.IsNonnegative())
	assert.True(coinsB.IsValid())
	assert.True(coinsB.IsZero())

	assert.True(coinsA.Plus(coinsB).IsEqual(coinsB.Plus(coinsA)))
	assert.True(coinsB.IsEqual(coinsB.Negative()))

	coinsD := coinsB.NoNil()
	assert.Equal(int64(0), coinsD.ThetaWei.Int64())
	assert.Equal(int64(0), coinsD.GammaWei.Int64())
}

func TestParseCoinAmount(t *testing.T) {
	assert := assert.New(t)

	weiMultiply := big.NewInt(1e18)

	var ret *big.Int
	var ok bool

	tmp := new(big.Int)
	ret, ok = ParseCoinAmount("1000")
	assert.True(ok)
	assert.True(tmp.Mul(big.NewInt(1000), weiMultiply).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("0")
	assert.True(ok)
	assert.True(big.NewInt(0).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("-1000")
	assert.False(ok)

	ret, ok = ParseCoinAmount("1e3")
	assert.True(ok)
	assert.True(tmp.Mul(big.NewInt(1000), weiMultiply).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("0.001e3")
	assert.True(ok)
	assert.True(tmp.Mul(big.NewInt(1), weiMultiply).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("0.0000001e3")
	assert.False(ok)

	ret, ok = ParseCoinAmount("100000wei")
	assert.True(ok)
	assert.True(big.NewInt(100000).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("1e3wei")
	assert.True(ok)
	assert.True(big.NewInt(1000).Cmp(ret) == 0)

	// Case insensitive.
	ret, ok = ParseCoinAmount("1e3Wei")
	assert.True(ok)
	assert.True(big.NewInt(1000).Cmp(ret) == 0)

func TestJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	num, ok := new(big.Int).SetString("12313123123123123131123123313212312312312312123", 10)
	require.True(ok)

	c := Coins{
		ThetaWei: num,
	}
	s, err := json.Marshal(c)
	require.Nil(err)

	var d Coins
	err = json.Unmarshal(s, &d)
	assert.Equal(0, num.Cmp(d.ThetaWei))
	assert.Nil(d.GammaWei)
}
