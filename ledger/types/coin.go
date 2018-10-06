package types

import (
	"fmt"
)

type Coins struct {
	ThetaWei int64 `json:"thetawei"`
	GammaWei int64 `json:"gammawei"`
}

func (coins Coins) String() string {
	return fmt.Sprintf("%v%v, %v%v", coins.ThetaWei, DenomThetaWei, coins.GammaWei, DenomGammaWei)
}

func (coins Coins) IsValid() bool {
	return coins.GammaWei >= 0 && coins.ThetaWei >= 0
}

// CalculatePercentage function calculates amount of coins for the given the percentage
func (coins Coins) CalculatePercentage(percentage uint) Coins {
	return Coins{
		GammaWei: int64(coins.GammaWei * int64(percentage) / 100), // FIXME: potential overflow
		ThetaWei: int64(coins.ThetaWei * int64(percentage) / 100), // FIXME: potential overflow
	}
}

// Currently appends an empty coin ...
func (coinsA Coins) Plus(coinsB Coins) Coins {
	return Coins{
		ThetaWei: coinsA.ThetaWei + coinsB.ThetaWei,
		GammaWei: coinsA.GammaWei + coinsB.GammaWei,
	}
}

func (coins Coins) Negative() Coins {
	return Coins{
		ThetaWei: -coins.ThetaWei,
		GammaWei: -coins.GammaWei,
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
	return coins.ThetaWei == 0 && coins.GammaWei == 0
}

func (coinsA Coins) IsEqual(coinsB Coins) bool {
	return coinsA.ThetaWei == coinsB.ThetaWei && coinsA.GammaWei == coinsB.GammaWei
}

func (coins Coins) IsPositive() bool {
	return coins.ThetaWei > 0 && coins.GammaWei > 0
}

func (coins Coins) IsNonnegative() bool {
	return coins.ThetaWei >= 0 && coins.GammaWei >= 0
}
