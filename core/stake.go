package core

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
)

const (
	StakeForValidator     uint8 = 0
	StakeForGuardian      uint8 = 1
	StakeForEliteEdgeNode uint8 = 2

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
	Holder       common.Address `rlp:"-"` // Keep reference to holder in memory to process split
	Source       common.Address
	Amount       *big.Int
	Withdrawn    bool
	ReturnHeight uint64
}

func NewStake(source common.Address, amount *big.Int) *Stake {
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

type StakeJSON struct {
	Source       common.Address  `json:"source"`
	Amount       *common.JSONBig `json:"amount"`
	Withdrawn    bool            `json:"withdrawn"`
	ReturnHeight *common.JSONBig `json:"return_height"`
}

func NewStakeJSON(stake Stake) StakeJSON {
	return StakeJSON{
		Source:       stake.Source,
		Amount:       (*common.JSONBig)(stake.Amount),
		Withdrawn:    stake.Withdrawn,
		ReturnHeight: (*common.JSONBig)(new(big.Int).SetUint64(stake.ReturnHeight)),
	}
}

func (s StakeJSON) Stake() Stake {
	return Stake{
		Source:       s.Source,
		Amount:       s.Amount.ToInt(),
		Withdrawn:    s.Withdrawn,
		ReturnHeight: s.ReturnHeight.ToInt().Uint64(),
	}
}

func (s Stake) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewStakeJSON(s))
}

func (s *Stake) UnmarshalJSON(data []byte) error {
	var a StakeJSON
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*s = a.Stake()
	return nil
}

//
// ------- StakeHolder ------- //
//

// TODO: Should rename StakeHolder to StakeDelegate
type StakeHolder struct {
	Holder common.Address
	Stakes []*Stake
}

func NewStakeHolder(holder common.Address, stakes []*Stake) *StakeHolder {
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

	newStake := NewStake(source, amount)
	sh.Stakes = append(sh.Stakes, newStake)

	return nil
}

func (sh *StakeHolder) withdrawStake(source common.Address, currentHeight uint64) (*Stake, error) {
	for _, stake := range sh.Stakes {
		if stake.Source == source {
			if stake.Withdrawn {
				return nil, fmt.Errorf("Already withdrawn, cannot withdraw again for source: %v", source)
			}
			stake.Withdrawn = true
			stake.ReturnHeight = currentHeight + ReturnLockingPeriod
			return stake, nil
		}
	}

	return nil, fmt.Errorf("Cannot withdraw, no matched stake source address found: %v", source)
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
