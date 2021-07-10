package state

import (
	crand "crypto/rand"
	"fmt"
	"io"
	"log"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/ledger/types"
)

const eenpRewardN = 800 // Reward receiver sampling params

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
func (eenp *EliteEdgeNodePool) RandomRewardWeight(block common.Hash, eenAddr common.Address) int {
	een := eenp.Get(eenAddr)
	if een == nil {
		logger.Debugf("elite edge node random reward weight: address = %v, block = %v, weight = 0, not staked yet", eenAddr, block.Hex())
		return 0
	}
	totalStake := eenp.sv.GetTotalEENStake()
	stake := een.TotalStake()

	seed := make([]byte, common.HashLength+common.AddressLength)
	copy(seed, block.Bytes())
	copy(seed[common.HashLength:], eenAddr[:])
	weight := sampleEENWeight(util.NewHashRand(seed), stake, totalStake)

	//logger.Debugf("elite edge node random reward weight: address = %v, block = %v, weight = %v, stake = %v, totalStake = %v", eenAddr, block.Hex(), weight, stake, totalStake)

	return weight
}

//
// The following sampling algorithm is based on Algorand's crypto sortition to randomly sample EENs.
// Denote the expected TOTAL number of selected "stake units" by n, the stake of the EEN stake by S.
// We essentially flip a biased coin (with head probability p) floor(S/S_min) times. And count the
// number of heads as the number of selected "stake units" of this EEN.
//
// The head probability p = min(1.0, a * n * S_min / S_total), where a = (S/S_min) / floor(S/S_min).
// The factor a is to compensate the cases where the stake S is not a multiple of S_min. It can be
// proved that if a user split the stakes onto multiple nodes, the expected return won't changes, the
// variance changes a bit but shouldn't be too big.
//
func sampleEENWeight(reader io.Reader, stake *big.Int, totalStake *big.Int) int {
	if stake.Cmp(big.NewInt(0)) == 0 || totalStake.Cmp(big.NewInt(0)) == 0 {
		// could happen when we sample an EEN whose stakes are all withdrawn, e.g. when
		// validating the votes from an EEN with all stakes withdrawn
		return 0
	}

	if totalStake.Cmp(big.NewInt(0)) < 0 {
		logger.Panicf("Negative total stake: %v", totalStake)
	}

	b := new(big.Int).Div(stake, core.MinEliteEdgeNodeStakeDeposit)

	base := new(big.Int).SetUint64(1e18)

	p := new(big.Int).Mul(base, big.NewInt(eenpRewardN))
	p.Mul(p, stake)
	p.Div(p, totalStake)
	p.Div(p, b)

	weight := 0
	for i := 0; i < int(b.Int64()); i++ {
		r, err := crand.Int(reader, base)
		if err != nil {
			log.Panicf("Failed to generate random number: %v", err)
		}
		if r.Cmp(p) < 0 {
			weight++
		}

		//logger.Debugf("elite edge node sampling: p = %v, r = %v, base = %v, weight = %v, stake = %v, totalStake = %v", p, r, base, weight, stake, totalStake)
	}

	return weight
}

// Contains checks if given address is in the pool.
func (eenp *EliteEdgeNodePool) Contains(eenAddr common.Address) bool {
	return (eenp.Get(eenAddr) != nil)
}

// GetPubKeys returns BLS pubkeys of given addresses.
func (eenp *EliteEdgeNodePool) GetPubKeys(eenAddrs []common.Address) []*bls.PublicKey {
	ret := []*bls.PublicKey{}
	for _, addr := range eenAddrs {
		een := eenp.Get(addr)
		if een == nil {
			return nil
		}
		ret = append(ret, een.Pubkey)
	}
	return ret
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
	cb := func(k, v common.Bytes) bool {
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

	eenp.sv.Traverse(prefix, cb)

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

	// Update total eenp stake
	totalStake := eenp.sv.GetTotalEENStake()
	totalStake.Add(totalStake, amount)
	eenp.sv.SetTotalEENStake(totalStake)

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

	// Update total eenp stake
	totalStake := eenp.sv.GetTotalEENStake()
	totalStake.Sub(totalStake, withdrawnStake.Amount)
	eenp.sv.SetTotalEENStake(totalStake)

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

			if len(een.Stakes) == 0 { // the candidate's stake becomes zero, no need to keep track of the candidate anymore
				eenp.Remove(een)
			} else {
				eenp.Upsert(een)
			}

			break // only one stake to be returned
		}
	}

	return nil
}
