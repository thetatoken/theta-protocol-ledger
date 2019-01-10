package core

import (
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
)

const (
	StakeForValidator uint8 = 0
	StakeForGuardian  uint8 = 1

	ReturnLockingPeriod uint64 = 28800      // number of blocks, approximately 2 days with 6 second block time
	InvalidReturnHeight uint64 = ^uint64(0) // max uint64
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
	Source       common.Address
	Amount       *big.Int
	Withdrawn    bool
	ReturnHeight uint64
}

func newStake(source common.Address, amount *big.Int) *Stake {
	return &Stake{
		Source:       source,
		Amount:       amount,
		Withdrawn:    false,
		ReturnHeight: InvalidReturnHeight,
	}
}

func (s *Stake) String() string {
	return fmt.Sprintf("{Source: %v, Amount: %v, Withdrawn: %v, ReturnHeight: %v}",
		s.Source, s.Amount, s.Withdrawn, s.ReturnHeight)
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

func (sh *StakeHolder) TotalStake() *big.Int {
	totalAmount := new(big.Int).SetUint64(0)
	for _, stake := range sh.Stakes {
		if !stake.Withdrawn {
			totalAmount = new(big.Int).Add(totalAmount, stake.Amount)
		}
	}
	return totalAmount
}

func (sh *StakeHolder) depositStake(source common.Address, amount *big.Int) error {
	if amount.Cmp(Zero) < 0 {
		return fmt.Errorf("Invalid stake: %v", amount)
	}

	for _, stake := range sh.Stakes {
		if stake.Source == source {
			if stake.Withdrawn {
				return fmt.Errorf("Cannot deposit during the withdrawal locking period for: %v", source)
			}

			stake.Amount = new(big.Int).Add(stake.Amount, amount)
			return nil
		}
	}

	newStake := newStake(source, amount)
	sh.Stakes = append(sh.Stakes, newStake)

	return nil
}

func (sh *StakeHolder) withdrawStake(source common.Address, currentHeight uint64) error {
	for _, stake := range sh.Stakes {
		if stake.Source == source {
			if stake.Withdrawn {
				return fmt.Errorf("Already withdrawn, cannot withdraw again for source: %v", source)
			}
			stake.Withdrawn = true
			stake.ReturnHeight = currentHeight + ReturnLockingPeriod
			return nil
		}
	}

	return fmt.Errorf("Cannot withdraw, no matched stake source address found: %v", source)
}

func (sh *StakeHolder) returnStake(source common.Address, currentHeight uint64) (*Stake, error) {
	for idx, stake := range sh.Stakes {
		if stake.Source == source {
			if !stake.Withdrawn {
				return nil, fmt.Errorf("Cannot return, stake not withdrawn yet")
			}
			if stake.ReturnHeight > currentHeight {
				return nil, fmt.Errorf("Cannot return, current height: %v, return height: %v",
					currentHeight, stake.ReturnHeight)
			}
			sh.Stakes = append(sh.Stakes[:idx], sh.Stakes[idx+1:]...)
			return stake, nil
		}
	}

	return nil, fmt.Errorf("Cannot return, no matched stake source address found: %v", source)
}

func (sh *StakeHolder) String() string {
	return fmt.Sprintf("{holder: %v, stakes :%v}", sh.Holder, sh.Stakes)
}
