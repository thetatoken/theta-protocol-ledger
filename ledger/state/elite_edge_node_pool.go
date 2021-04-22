package state

import (
	"fmt"
	"log"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/ledger/types"
)

//
// ------- EliteEdgeNodePool ------- //
//

type EliteEdgeNodePool struct {
	readOnly bool
	sv       *StoreView
}

// NewEliteEdgeNodePool creates a new instance of EliteEdgeNodePool.
func NewEliteEdgeNodePool(sv *StoreView, readOnly bool) *EliteEdgeNodePool {
	return &EliteEdgeNodePool{
		readOnly: readOnly,
		sv:       sv,
	}
}

// Contains checks if given address is in the pool.
func (eenp *EliteEdgeNodePool) Contains(eenAddr common.Address) bool {
	return (eenp.Get(eenAddr) != nil)
}

// Get returns the EEN if exists, nil otherwise
func (eenp *EliteEdgeNodePool) Get(eenAddr common.Address) *core.EliteEdgeNode {
	eenKey := EliteEdgeNodeKey(eenAddr)
	data := eenp.sv.Get(eenKey)
	if data == nil || len(data) == 0 {
		return nil
	}

	een := &core.EliteEdgeNode{}
	err := types.FromBytes(data, een)
	if err != nil {
		log.Panicf("EliteEdgeNodePool.Get: Error reading elite edge node %X, error: %v",
			data, err.Error())
	}

	return een
}

// Upsert update or insert an elite edge node
func (eenp *EliteEdgeNodePool) Upsert(een *core.EliteEdgeNode) {
	if eenp.readOnly {
		log.Panicf("EliteEdgeNodePool.Upsert: the pool is read-only")
	}

	eenKey := EliteEdgeNodeKey(een.Holder)
	data, err := types.ToBytes(een)
	if err != nil {
		log.Panicf("EliteEdgeNodePool.Upsert: Error serializing elite edge node %X, error: %v",
			data, err.Error())
	}
	eenp.sv.Set(eenKey, data)
	eenp.sv.Save()
}

// Remove deletes the elite edge node from the pool
func (eenp *EliteEdgeNodePool) Remove(een *core.EliteEdgeNode) {
	if eenp.readOnly {
		log.Panicf("EliteEdgeNodePool.Upsert: the pool is read-only")
	}

	eenKey := EliteEdgeNodeKey(een.Holder)
	eenp.sv.Delete(eenKey)
}

func (eenp *EliteEdgeNodePool) GetAll(withstake bool) []*core.EliteEdgeNode {
	prefix := EliteEdgeNodeKeyPrefix()

	eenList := []*core.EliteEdgeNode{}
	cb := func() func(k, v common.Bytes) bool {
		return func(k, v common.Bytes) bool {
			een := &core.EliteEdgeNode{}
			err := types.FromBytes(v, een)
			if err != nil {
				log.Panicf("EliteEdgeNodePool.GetAll: Error reading elite edge node %X, error: %v",
					v, err.Error())
			}
			if withstake {
				hasStake := false
				for _, stake := range een.Stakes {
					if !stake.Withdrawn {
						hasStake = true
						break
					}
				}
				if !hasStake {
					return true // Skip if een dons't have non-withdrawn stake
				}
			}
			eenList = append(eenList, een)
			return true
		}
	}

	eenp.sv.Traverse(prefix, cb())

	return eenList
}

func (eenp *EliteEdgeNodePool) DepositStake(source common.Address, holder common.Address, amount *big.Int, pubkey *bls.PublicKey, blockHeight uint64) (err error) {
	if eenp.readOnly {
		log.Panicf("EliteEdgeNodePool.DepositStake: the pool is read-only")
	}

	minEliteEdgeNodeStake := core.MinEliteEdgeNodeStakeDeposit
	maxEliteEdgeNodeStake := core.MaxEliteEdgeNodeStakeDeposit
	if amount.Cmp(minEliteEdgeNodeStake) < 0 {
		return fmt.Errorf("Elite edge node staking amount below the lower limit: %v", amount)
	}
	if amount.Cmp(maxEliteEdgeNodeStake) > 0 {
		return fmt.Errorf("Elite edge node staking amount above the upper limit: %v", amount)
	}

	een := eenp.Get(holder)
	if een == nil {
		een = core.NewEliteEdgeNode(
			core.NewStakeHolder(holder, []*core.Stake{core.NewStake(source, amount)}),
			pubkey)
	} else {
		if een.Holder != holder {
			log.Panicf("EliteEdgeNodePool.DepositStake: holder mismatch, een.Holder = %v, holder = %v",
				een.Holder.Hex(), holder.Hex())
		}
		currentStake := een.TotalStake()
		expectedStake := big.NewInt(0).Add(currentStake, amount)
		if expectedStake.Cmp(maxEliteEdgeNodeStake) > 0 {
			return fmt.Errorf("Elite edge node stake would exceed the cap: %v", expectedStake)
		}
		err = een.DepositStake(source, amount)
		if err != nil {
			return err
		}
	}

	eenp.Upsert(een)

	return nil
}

func (eenp *EliteEdgeNodePool) WithdrawStake(source common.Address, holder common.Address, currentHeight uint64) (*core.Stake, error) {
	if eenp.readOnly {
		log.Panicf("EliteEdgeNodePool.WithdrawStake: the pool is read-only")
	}

	var withdrawnStake *core.Stake
	var err error

	een := eenp.Get(holder)
	if een == nil {
		return nil, fmt.Errorf("No matched stake holder address found: %v", holder)
	}

	if een.Holder != holder {
		log.Panicf("EliteEdgeNodePool.DepositStake: holder mismatch, een.Holder = %v, holder = %v",
			een.Holder.Hex(), holder.Hex())
	}

	withdrawnStake, err = een.WithdrawStake(source, currentHeight)
	if err != nil {
		return nil, err
	}

	eenp.Upsert(een)

	return withdrawnStake, nil
}

func (eenp *EliteEdgeNodePool) ReturnStake(currentHeight uint64, holder common.Address, returnedStake core.Stake) error {
	een := eenp.Get(holder)
	if een == nil {
		return fmt.Errorf("No matched stake holder address found: %v", holder)
	}

	sourceAddress := returnedStake.Source
	numStakes := len(een.Stakes)

	// need to iterate in the reverse order, since we may delete elemements from the slice while iterating through it
	for sidx := numStakes - 1; sidx >= 0; sidx-- {
		stake := een.Stakes[sidx]

		if stake.Source == sourceAddress {
			if stake.Withdrawn == false || stake.ReturnHeight != currentHeight {
				log.Panicf("Returned stake mismatch: eenAddr = %v, sourceAddr = %v, currentHeight = %v, stake.Withdrawn = %v, stake.ReturnHeight = %v",
					holder, sourceAddress, currentHeight, stake.Withdrawn, stake.ReturnHeight)
			}

			logger.Infof("Stake to be returned: source = %v, amount = %v", stake.Source, stake.Amount)
			_, err := een.ReturnStake(sourceAddress, currentHeight)
			if err != nil {
				return err
			}

			break // only one stake to be returned
		}

		if len(een.Stakes) == 0 { // the candidate's stake becomes zero, no need to keep track of the candidate anymore
			eenp.Remove(een)
		} else {
			eenp.Upsert(een)
		}
	}

	return nil
}
