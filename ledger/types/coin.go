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

// CalculatePercentage function calculates amount of coins for the given the percentage
func (coins Coins) CalculatePercentage(percentage uint) Coins {
	p := big.NewInt(int64(percentage))

	theta := new(big.Int)
	theta.Mul(coins.ThetaWei, p)
	theta.Div(theta, Hundred)

	gamma := new(big.Int)
	gamma.Mul(coins.GammaWei, p)
	gamma.Div(gamma, Hundred)

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
	}
}

// Currently appends an empty coin ...
func (coinsA Coins) Plus(coinsB Coins) Coins {
	theta := new(big.Int)
	theta.Add(coinsA.ThetaWei, coinsB.ThetaWei)

	gamma := new(big.Int)
	gamma.Add(coinsA.GammaWei, coinsB.GammaWei)

	return Coins{
		ThetaWei: theta,
		GammaWei: gamma,
	}
}

func (coins Coins) Negative() Coins {
	theta := new(big.Int)
	theta.Neg(coins.ThetaWei)

	gamma := new(big.Int)
	gamma.Neg(coins.GammaWei)

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
	return coins.ThetaWei.Cmp(Zero) == 0 && coins.GammaWei.Cmp(Zero) == 0
}

func (coinsA Coins) IsEqual(coinsB Coins) bool {
	return coinsA.ThetaWei == coinsB.ThetaWei && coinsA.GammaWei == coinsB.GammaWei
}

func (coins Coins) IsPositive() bool {
	return coins.ThetaWei.Cmp(Zero) > 0 && coins.GammaWei.Cmp(Zero) > 0
}

func (coins Coins) IsNonnegative() bool {
	return coins.ThetaWei.Cmp(Zero) >= 0 && coins.GammaWei.Cmp(Zero) >= 0
}
