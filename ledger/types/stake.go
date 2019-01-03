package types

import (
	"fmt"
	"sort"

	"github.com/thetatoken/ukulele/common"
)

//
// ------- Stake ------- //
//

type Stake struct {
	Source common.Address
	Amount Coins
}

func newStake(source common.Address, amount Coins) *Stake {
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

func (sh *StakeHolder) totalStake() Coins {
	totalAmount := NewCoins(0, 0)
	for _, stake := range sh.Stakes {
		totalAmount = totalAmount.Plus(stake.Amount)
	}
	return totalAmount
}

func (sh *StakeHolder) depositStake(source common.Address, amount Coins) error {
	if !amount.IsNonnegative() {
		return fmt.Errorf("Invalid stake: %v", amount)
	}

	for _, stake := range sh.Stakes {
		if stake.Source == source {
			stake.Amount = stake.Amount.Plus(amount)
			return nil
		}
	}

	newStake := newStake(source, amount)
	sh.Stakes = append(sh.Stakes, newStake)

	return nil
}

func (sh *StakeHolder) withdrawStake(source common.Address, amount Coins) (withdrawnAmount Coins, err error) {
	withdrawnAmount = NewCoins(0, 0)
	if !amount.IsNonnegative() {
		return withdrawnAmount, fmt.Errorf("Invalid stake: %v", amount)
	}

	for idx, stake := range sh.Stakes {
		if stake.Source == source {
			if amount.IsGTE(stake.Amount) { // considered as full withdrawal
				withdrawnAmount = stake.Amount
				sh.Stakes = append(sh.Stakes[:idx], sh.Stakes[idx+1:]...)
			} else {
				withdrawnAmount = amount
				stake.Amount = stake.Amount.Minus(amount)
			}
			return withdrawnAmount, nil
		}
	}

	return withdrawnAmount, fmt.Errorf("No matched stake source address found: %v", source)
}

//
// ------- StakeHolderSet ------- //
//

type StakeHolderSet struct {
	SortedStakeHolders []*StakeHolder
}

func (shs *StakeHolderSet) DepositStake(source common.Address, holder common.Address, amount Coins) (err error) {
	if !amount.IsNonnegative() {
		return fmt.Errorf("Invalid stake: %v", amount)
	}

	matchedHolderFound := false
	for _, sh := range shs.SortedStakeHolders {
		if sh.Holder == holder {
			matchedHolderFound = true
			err = sh.depositStake(source, amount)
			if err != nil {
				return err
			}
			break
		}
	}

	if !matchedHolderFound {
		newStakeHolder := newStakeHolder(holder, []*Stake{newStake(source, amount)})
		shs.SortedStakeHolders = append(shs.SortedStakeHolders, newStakeHolder)
	}

	sort.Slice(shs.SortedStakeHolders[:], func(i, j int) bool { // descending order
		return shs.SortedStakeHolders[i].totalStake().IsGTE(shs.SortedStakeHolders[j].totalStake())
	})

	return nil
}

func (shs *StakeHolderSet) WithdrawStake(source common.Address, holder common.Address, amount Coins) (withdrawnAmount Coins, err error) {
	withdrawnAmount = NewCoins(0, 0)
	if !amount.IsNonnegative() {
		return withdrawnAmount, fmt.Errorf("Invalid stake: %v", amount)
	}

	matchedHolderFound := false
	for _, sh := range shs.SortedStakeHolders {
		if sh.Holder == holder {
			matchedHolderFound = true
			withdrawnAmount, err = sh.withdrawStake(source, amount)
			if err != nil {
				return withdrawnAmount, err
			}
			break
		}
	}

	if !matchedHolderFound {
		return withdrawnAmount, fmt.Errorf("No matched stake holder address found: %v", holder)
	}

	sort.Slice(shs.SortedStakeHolders[:], func(i, j int) bool { // descending order
		return shs.SortedStakeHolders[i].totalStake().IsGTE(shs.SortedStakeHolders[j].totalStake())
	})

	return withdrawnAmount, nil
}
