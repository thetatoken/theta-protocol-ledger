package core

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
)

const (
	StakeForValidator uint8 = 0
	StakeForGuardian  uint8 = 1
)

var (
	Zero *big.Int
)

func init() {
	Zero = big.NewInt(0)
}

//
// ------- Stake ------- //
//

type Stake struct {
	Source common.Address
	Amount *big.Int
}

func newStake(source common.Address, amount *big.Int) *Stake {
	return &Stake{
		Source: source,
		Amount: amount,
	}
}

//
// ------- StakeHolder ------- //
//

type StakeHolder struct {
	Holder common.Address
	Stakes []*Stake
}

func newStakeHolder(holder common.Address, stakes []*Stake) *StakeHolder {
	return &StakeHolder{
		Holder: holder,
		Stakes: stakes,
	}
}

func (sh *StakeHolder) totalStake() *big.Int {
	totalAmount := new(big.Int).SetUint64(0)
	for _, stake := range sh.Stakes {
		totalAmount = new(big.Int).Add(totalAmount, stake.Amount)
	}
	return totalAmount
}

func (sh *StakeHolder) depositStake(source common.Address, amount *big.Int) error {
	if amount.Cmp(Zero) < 0 {
		return fmt.Errorf("Invalid stake: %v", amount)
	}

	for _, stake := range sh.Stakes {
		if stake.Source == source {
			stake.Amount = new(big.Int).Add(stake.Amount, amount)
			return nil
		}
	}

	newStake := newStake(source, amount)
	sh.Stakes = append(sh.Stakes, newStake)

	return nil
}

func (sh *StakeHolder) withdrawStake(source common.Address) (withdrawnAmount *big.Int, err error) {
	withdrawnAmount = new(big.Int).SetUint64(0)
	for idx, stake := range sh.Stakes {
		if stake.Source == source {
			withdrawnAmount = stake.Amount // always withdraws the full amount
			sh.Stakes = append(sh.Stakes[:idx], sh.Stakes[idx+1:]...)
			return withdrawnAmount, nil
		}
	}

	return withdrawnAmount, fmt.Errorf("No matched stake source address found: %v", source)
}
