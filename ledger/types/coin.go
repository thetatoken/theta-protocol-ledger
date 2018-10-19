package types

import (
	"fmt"
	"math/big"
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
	ThetaWei *big.Int `json:"thetawei"`
	GammaWei *big.Int `json:"gammawei"`
}

// NewCoins is a convenient method for creating small amount of coins.
func NewCoins(theta int, gamma int) Coins {
	return Coins{
		ThetaWei: big.NewInt(int64(theta)),
		GammaWei: big.NewInt(int64(gamma)),
	}
}

func (coins Coins) String() string {
	return fmt.Sprintf("%v%v, %v%v", coins.ThetaWei, DenomThetaWei, coins.GammaWei, DenomGammaWei)
}

func (coins Coins) IsValid() bool {
	return coins.IsNonnegative()
}

func (coins Coins) NoNil() Coins {
	theta := coins.ThetaWei
	if theta == nil {
		theta = big.NewInt(0)
	}
	gamma := coins.GammaWei
	if gamma == nil {
		gamma = big.NewInt(0)
	}

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
	}
}

// CalculatePercentage function calculates amount of coins for the given the percentage
func (coins Coins) CalculatePercentage(percentage uint) Coins {
	c := coins.NoNil()

	p := big.NewInt(int64(percentage))

	theta := new(big.Int)
	theta.Mul(c.ThetaWei, p)
	theta.Div(theta, Hundred)

	gamma := new(big.Int)
	gamma.Mul(c.GammaWei, p)
	gamma.Div(gamma, Hundred)

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
	}
}

// Currently appends an empty coin ...
func (coinsA Coins) Plus(coinsB Coins) Coins {
	cA := coinsA.NoNil()
	cB := coinsB.NoNil()

	theta := new(big.Int)
	theta.Add(cA.ThetaWei, cB.ThetaWei)

	gamma := new(big.Int)
	gamma.Add(cA.GammaWei, cB.GammaWei)

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
	}
}

func (coins Coins) Negative() Coins {
	c := coins.NoNil()

	theta := new(big.Int)
	theta.Neg(c.ThetaWei)

	gamma := new(big.Int)
	gamma.Neg(c.GammaWei)

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
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
	return c.ThetaWei.Cmp(Zero) == 0 && c.GammaWei.Cmp(Zero) == 0
}

func (coinsA Coins) IsEqual(coinsB Coins) bool {
	cA := coinsA.NoNil()
	cB := coinsB.NoNil()
	return cA.ThetaWei.Cmp(cB.ThetaWei) == 0 && cA.GammaWei.Cmp(cB.GammaWei) == 0
}

func (coins Coins) IsPositive() bool {
	c := coins.NoNil()
	return (c.ThetaWei.Cmp(Zero) > 0 && c.GammaWei.Cmp(Zero) >= 0) ||
		(c.ThetaWei.Cmp(Zero) >= 0 && c.GammaWei.Cmp(Zero) > 0)
}

func (coins Coins) IsNonnegative() bool {
	c := coins.NoNil()
	return c.ThetaWei.Cmp(Zero) >= 0 && c.GammaWei.Cmp(Zero) >= 0
}
