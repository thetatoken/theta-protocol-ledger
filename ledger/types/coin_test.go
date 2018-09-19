package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCoins(t *testing.T) {
	assert := assert.New(t)

	//Define the coins to be used in tests
	good := Coins{
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
		Coin{"TREE", 1},
	}
	neg := good.Negative()
	sum := good.Plus(neg)
	empty := Coins{
		Coin{"GOLD", 0},
	}
	badSort1 := Coins{
		Coin{"TREE", 1},
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
	}
	badSort2 := Coins{ // both are after the first one, but the second and third are in the wrong order
		Coin{"GAS", 1},
		Coin{"TREE", 1},
		Coin{"MINERAL", 1},
	}
	badAmt := Coins{
		Coin{"GAS", 1},
		Coin{"TREE", 0},
		Coin{"MINERAL", 1},
	}
	dup := Coins{
		Coin{"GAS", 1},
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
	}

	assert.True(good.IsValid(), "Coins are valid")
	assert.True(good.IsPositive(), "Expected coins to be positive: %v", good)
	assert.True(good.IsGTE(empty), "Expected %v to be >= %v", good, empty)
	assert.False(neg.IsPositive(), "Expected neg coins to not be positive: %v", neg)
	assert.Zero(len(sum), "Expected 0 coins")
	assert.False(badSort1.IsValid(), "Coins are not sorted")
	assert.False(badSort2.IsValid(), "Coins are not sorted")
	assert.False(badAmt.IsValid(), "Coins cannot include 0 amounts")
	assert.False(dup.IsValid(), "Duplicate coin")

}

//Test operations on invalid coins
func TestInvalidCoin(t *testing.T) {
	assert := assert.New(t)

	coinsA := Coins{{Denom: "ThetaWei", Amount: 123}}
	coinsEmpty := Coins{}
	coinsEmpty2 := Coins{{}}

	ret1 := coinsA.Plus(coinsEmpty)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")
	ret1 = coinsEmpty.Plus(coinsA)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")

	ret1 = coinsA.Plus(coinsEmpty2)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")
	ret1 = coinsEmpty2.Plus(coinsA)
	assert.True(coinsA.IsEqual(ret1), "Sum is correct")

	// Result should be a copy
	ret1[0].Amount = 456
	assert.True(coinsA[0].Amount == 123)
	assert.True(ret1[0].Amount == 456)

	ret2 := coinsA.Minus(coinsEmpty)
	assert.True(coinsA.IsEqual(ret2), "Sum is correct")
	// Result should be a copy
	ret2[0].Amount = 456
	assert.True(coinsA[0].Amount == 123)
	assert.True(ret2[0].Amount == 456)
}

//Test the parse coin and parse coins functionality
func TestParse(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		input    string
		valid    bool  // if false, we expect an error on parse
		expected Coins // if valid is true, make sure this is returned
	}{
		{"", true, nil},
		{"1foo", true, Coins{{"foo", 1}}},
		{"10bar", true, Coins{{"bar", 10}}},
		{"99bar,1foo", true, Coins{{"bar", 99}, {"foo", 1}}},
		{"98 bar , 1 foo  ", true, Coins{{"bar", 98}, {"foo", 1}}},
		{"  55\t \t bling\n", true, Coins{{"bling", 55}}},
		{"2foo, 97 bar", true, Coins{{"bar", 97}, {"foo", 2}}},
		{"5 mycoin,", false, nil},             // no empty coins in a list
		{"2 3foo, 97 bar", false, nil},        // 3foo is invalid coin name
		{"11me coin, 12you coin", false, nil}, // no spaces in coin names
		{"1.2btc", false, nil},                // amount must be integer
		{"5foo-bar", false, nil},              // once more, only letters in coin name
	}

	for _, tc := range cases {
		res, err := ParseCoins(tc.input)
		if !tc.valid {
			assert.NotNil(err, "%s: %#v", tc.input, res)
		} else if assert.Nil(err, "%s: %+v", tc.input, err) {
			assert.Equal(tc.expected, res)
		}
	}

}

func TestSortCoins(t *testing.T) {
	assert := assert.New(t)

	good := Coins{
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
		Coin{"TREE", 1},
	}
	empty := Coins{
		Coin{"GOLD", 0},
	}
	badSort1 := Coins{
		Coin{"TREE", 1},
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
	}
	badSort2 := Coins{ // both are after the first one, but the second and third are in the wrong order
		Coin{"GAS", 1},
		Coin{"TREE", 1},
		Coin{"MINERAL", 1},
	}
	badAmt := Coins{
		Coin{"GAS", 1},
		Coin{"TREE", 0},
		Coin{"MINERAL", 1},
	}
	dup := Coins{
		Coin{"GAS", 1},
		Coin{"GAS", 1},
		Coin{"MINERAL", 1},
	}

	cases := []struct {
		coins         Coins
		before, after bool // valid before/after sort
	}{
		{good, true, true},
		{empty, false, false},
		{badSort1, false, true},
		{badSort2, false, true},
		{badAmt, false, false},
		{dup, false, false},
	}

	for _, tc := range cases {
		assert.Equal(tc.before, tc.coins.IsValid())
		tc.coins.Sort()
		assert.Equal(tc.after, tc.coins.IsValid())
	}
}
