package core

import (
	"fmt"
	"math/big"
	"sort"

	"github.com/thetatoken/ukulele/common"
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

func (sh *StakeHolder) withdrawStake(source common.Address, amount *big.Int) (withdrawnAmount *big.Int, err error) {
	withdrawnAmount = new(big.Int).SetUint64(0)
	if amount.Cmp(Zero) < 0 {
		return withdrawnAmount, fmt.Errorf("Invalid stake: %v", amount)
	}

	for idx, stake := range sh.Stakes {
		if stake.Source == source {
			if amount.Cmp(stake.Amount) >= 0 { // considered as full withdrawal
				withdrawnAmount = stake.Amount
				sh.Stakes = append(sh.Stakes[:idx], sh.Stakes[idx+1:]...)
			} else {
				withdrawnAmount = amount
				stake.Amount = new(big.Int).Sub(stake.Amount, amount)
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

func (shs *StakeHolderSet) DepositStake(source common.Address, holder common.Address, amount *big.Int) (err error) {
	if amount.Cmp(Zero) < 0 {
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
		return shs.SortedStakeHolders[i].totalStake().Cmp(shs.SortedStakeHolders[j].totalStake()) >= 0
	})

	return nil
}

func (shs *StakeHolderSet) WithdrawStake(source common.Address, holder common.Address, amount *big.Int) (withdrawnAmount *big.Int, err error) {
	withdrawnAmount = new(big.Int).SetUint64(0)
	if amount.Cmp(Zero) < 0 {
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
		return shs.SortedStakeHolders[i].totalStake().Cmp(shs.SortedStakeHolders[j].totalStake()) >= 0
	})

	return withdrawnAmount, nil
}
