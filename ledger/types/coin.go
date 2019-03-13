package types

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/thetatoken/theta/common"
)

var (
	Zero    *big.Int
	Hundred *big.Int
)

func init() {
	Zero = big.NewInt(0)
	Hundred = big.NewInt(100)
}

type Coins struct {
	ThetaWei *big.Int
	TFuelWei *big.Int
}

type CoinsJSON struct {
	ThetaWei *common.JSONBig `json:"thetawei"`
	TFuelWei *common.JSONBig `json:"tfuelwei"`
}

func NewCoinsJSON(coin Coins) CoinsJSON {
	return CoinsJSON{
		ThetaWei: (*common.JSONBig)(coin.ThetaWei),
		TFuelWei: (*common.JSONBig)(coin.TFuelWei),
	}
}

func (c CoinsJSON) Coins() Coins {
	return Coins{
		ThetaWei: (*big.Int)(c.ThetaWei),
		TFuelWei: (*big.Int)(c.TFuelWei),
	}
}

func (c Coins) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewCoinsJSON(c))
}

func (c *Coins) UnmarshalJSON(data []byte) error {
	var a CoinsJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*c = a.Coins()
	return nil
}

// NewCoins is a convenient method for creating small amount of coins.
func NewCoins(theta int64, tfuel int64) Coins {
	return Coins{
		ThetaWei: big.NewInt(theta),
		TFuelWei: big.NewInt(tfuel),
	}
}

func (coins Coins) String() string {
	return fmt.Sprintf("%v %v, %v %v", coins.ThetaWei, DenomThetaWei, coins.TFuelWei, DenomTFuelWei)
}

func (coins Coins) IsValid() bool {
	return coins.IsNonnegative()
}

func (coins Coins) NoNil() Coins {
	theta := coins.ThetaWei
	if theta == nil {
		theta = big.NewInt(0)
	}
	tfuel := coins.TFuelWei
	if tfuel == nil {
		tfuel = big.NewInt(0)
	}

	return Coins{
		ThetaWei: theta,
		TFuelWei: tfuel,
	}
}

// CalculatePercentage function calculates amount of coins for the given the percentage
func (coins Coins) CalculatePercentage(percentage uint) Coins {
	c := coins.NoNil()

	p := big.NewInt(int64(percentage))

	theta := new(big.Int)
	theta.Mul(c.ThetaWei, p)
	theta.Div(theta, Hundred)

	tfuel := new(big.Int)
	tfuel.Mul(c.TFuelWei, p)
	tfuel.Div(tfuel, Hundred)

	return Coins{
		ThetaWei: theta,
		TFuelWei: tfuel,
	}
}

// Currently appends an empty coin ...
func (coinsA Coins) Plus(coinsB Coins) Coins {
	cA := coinsA.NoNil()
	cB := coinsB.NoNil()

	theta := new(big.Int)
	theta.Add(cA.ThetaWei, cB.ThetaWei)

	tfuel := new(big.Int)
	tfuel.Add(cA.TFuelWei, cB.TFuelWei)

	return Coins{
		ThetaWei: theta,
		TFuelWei: tfuel,
	}
}

func (coins Coins) Negative() Coins {
	c := coins.NoNil()

	theta := new(big.Int)
	theta.Neg(c.ThetaWei)

	tfuel := new(big.Int)
	tfuel.Neg(c.TFuelWei)

	return Coins{
		ThetaWei: theta,
		TFuelWei: tfuel,
	}
}

func (coinsA Coins) Minus(coinsB Coins) Coins {
	return coinsA.Plus(coinsB.Negative())
}

func (coinsA Coins) IsGTE(coinsB Coins) bool {
	diff := coinsA.Minus(coinsB)
	return diff.IsNonnegative()
}

func (coins Coins) IsZero() bool {
	c := coins.NoNil()
	return c.ThetaWei.Cmp(Zero) == 0 && c.TFuelWei.Cmp(Zero) == 0
}

func (coinsA Coins) IsEqual(coinsB Coins) bool {
	cA := coinsA.NoNil()
	cB := coinsB.NoNil()
	return cA.ThetaWei.Cmp(cB.ThetaWei) == 0 && cA.TFuelWei.Cmp(cB.TFuelWei) == 0
}

func (coins Coins) IsPositive() bool {
	c := coins.NoNil()
	return (c.ThetaWei.Cmp(Zero) > 0 && c.TFuelWei.Cmp(Zero) >= 0) ||
		(c.ThetaWei.Cmp(Zero) >= 0 && c.TFuelWei.Cmp(Zero) > 0)
}

func (coins Coins) IsNonnegative() bool {
	c := coins.NoNil()
	return c.ThetaWei.Cmp(Zero) >= 0 && c.TFuelWei.Cmp(Zero) >= 0
}

// ParseCoinAmount parses a string representation of coin amount.
func ParseCoinAmount(in string) (*big.Int, bool) {
	inWei := false
	if len(in) > 3 && strings.EqualFold("wei", in[len(in)-3:]) {
		inWei = true
		in = in[:len(in)-3]
	}

	f, ok := new(big.Float).SetPrec(1024).SetString(in)
	if !ok || f.Sign() < 0 {
		return nil, false
	}

	if !inWei {
		f = f.Mul(f, new(big.Float).SetPrec(1024).SetUint64(1e18))
	}

	ret, _ := f.Int(nil)

	return ret, true
}
