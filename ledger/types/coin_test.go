package types

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"testing"

	"github.com/thetatoken/theta/rlp"

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

func TestcCoinsRLPNil(t *testing.T) {
	assert := assert.New(t)

	a := Coins{}
	rawA, err := rlp.EncodeToBytes(a)
	assert.Nil(err)

	b := &Coins{}
	rlp.DecodeBytes(rawA, b)

	assert.NotNil(b.ThetaWei)
	assert.NotNil(b.TFuelWei)

	c := NewCoins(0, 0)
	rawC, err := rlp.EncodeToBytes(c)
	assert.Nil(err)
	assert.Equal(rawC, rawA)
}

func TestCoinsRLPCollision(t *testing.T) {
	assert := assert.New(t)

	// 0 is encoded into "80", just to make sure it doesn't collide with
	// 80 or 128 (128 == 0x80)

	zero := big.NewInt(0)
	i80 := big.NewInt(80)
	i128 := big.NewInt(128)

	zeroBytes, _ := rlp.EncodeToBytes(zero)
	i80Bytes, _ := rlp.EncodeToBytes(i80)
	i128Bytes, _ := rlp.EncodeToBytes(i128)

	fmt.Printf("0   : %v\n", hex.EncodeToString(zeroBytes))
	fmt.Printf("80  : %v\n", hex.EncodeToString(i80Bytes))
	fmt.Printf("128 : %v\n", hex.EncodeToString(i128Bytes))

	assert.True(bytes.Compare(zeroBytes, i80Bytes) != 0)
	assert.True(bytes.Compare(zeroBytes, i128Bytes) != 0)
	assert.True(bytes.Compare(i80Bytes, i128Bytes) != 0)

	coins0 := NewCoins(0, 0)
	coins0Bytes, _ := rlp.EncodeToBytes(coins0)
	fmt.Printf("coins0     : %v\n", hex.EncodeToString(coins0Bytes))

	coins80 := NewCoins(80, 80)
	coins80Bytes, _ := rlp.EncodeToBytes(coins80)
	fmt.Printf("coins80    : %v\n", hex.EncodeToString(coins80Bytes))

	coins128 := NewCoins(128, 128)
	coins128Bytes, _ := rlp.EncodeToBytes(coins128)
	fmt.Printf("coins128   : %v\n", hex.EncodeToString(coins128Bytes))

	assert.True(bytes.Compare(coins0Bytes, coins80Bytes) != 0)
	assert.True(bytes.Compare(coins0Bytes, coins128Bytes) != 0)
	assert.True(bytes.Compare(coins80Bytes, coins128Bytes) != 0)
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
	assert.Equal(int64(0), coinsD.TFuelWei.Int64())
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
	assert.True(ok)

	ret, ok = ParseCoinAmount("100000wei")
	assert.True(ok)
	assert.True(big.NewInt(100000).Cmp(ret) == 0)

	ret, ok = ParseCoinAmount("1e3wei")

	t.Logf("1e3wei => %v\n", ret)

	assert.True(ok)
	assert.True(big.NewInt(1000).Cmp(ret) == 0)

	// Case insensitive.
	ret, ok = ParseCoinAmount("1e3Wei")
	assert.True(ok)
	assert.True(big.NewInt(1000).Cmp(ret) == 0)
}

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
	assert.Nil(d.TFuelWei)
}
