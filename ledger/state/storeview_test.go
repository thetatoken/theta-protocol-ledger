package state

import (
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database/backend"
)

func TestStoreViewBasics(t *testing.T) {
	assert := assert.New(t)

	initHeight := uint64(1)
	incrementedHeight := initHeight + 1
	db := backend.NewMemDatabase()
	sv1 := NewStoreView(initHeight, common.Hash{}, db)

	// Height tests
	assert.Equal(initHeight, sv1.Height())

	sv1.IncrementHeight()
	assert.Equal(incrementedHeight, sv1.Height())

	// Set/Get tests
	k1, v1 := common.Bytes("key1"), common.Bytes("value1")
	k2, v2 := common.Bytes("key2"), common.Bytes("value2")
	k3, v3 := common.Bytes("key3"), common.Bytes("value3")
	k4, v4 := common.Bytes("key4"), common.Bytes("value4")

	sv1.Set(k1, v1)
	sv1.Set(k2, v2)
	sv1.Set(k3, v3)

	assert.Equal(v1, sv1.Get(k1))
	assert.Equal(v2, sv1.Get(k2))
	assert.Equal(v3, sv1.Get(k3))

	// Root hash tests
	sv1RootHashCalculated := sv1.Hash()
	sv1RootHashCommitted := sv1.Save()
	log.Infof("sv1 calculated root hash (before sv2 insertion): %v", sv1RootHashCalculated.Hex())
	log.Infof("sv1 committed root hash  (before sv2 insertion): %v", sv1RootHashCommitted.Hex())
	assert.Equal(sv1RootHashCalculated, sv1RootHashCommitted)

	// StoreView copy tests
	sv2, err := sv1.Copy()
	assert.Nil(err)

	assert.Equal(v1, sv2.Get(k1))
	assert.Equal(v2, sv2.Get(k2))
	assert.Equal(v3, sv2.Get(k3))

	sv2RootHashCalculated := sv2.Hash()
	log.Infof("sv2 calculated root hash (before sv2 insertion): %v", sv2RootHashCalculated.Hex())
	assert.Equal(sv1RootHashCalculated, sv2RootHashCalculated)

	sv2.Set(k4, v4)
	assert.Equal(v1, sv2.Get(k1))
	assert.Equal(v4, sv2.Get(k4))
	assert.Equal(common.Bytes(nil), sv1.Get(k4))

	sv1RootHashCalculatedAfterInsertion := sv1.Hash()
	sv2RootHashCalculatedAfterInsertion := sv2.Hash()
	log.Infof("sv1 calculated root hash (after sv2 insertion) : %v", sv1RootHashCalculatedAfterInsertion.Hex())
	log.Infof("sv2 calculated root hash (after sv2 insertion) : %v", sv2RootHashCalculatedAfterInsertion.Hex())
	assert.Equal(sv1RootHashCalculated, sv1RootHashCalculatedAfterInsertion)
	assert.NotEqual(sv2RootHashCalculated, sv2RootHashCalculatedAfterInsertion)
}

func TestStoreViewAccountAccess(t *testing.T) {
	assert := assert.New(t)

	_, pubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)

	initCoin := types.Coins{ThetaWei: big.NewInt(786), TFuelWei: big.NewInt(0)}
	acc1 := &types.Account{
		Address:  pubKey.Address(),
		Sequence: 173,
		Balance:  initCoin,
	}
	acc1Addr := acc1.Address

	db := backend.NewMemDatabase()
	sv1 := NewStoreView(uint64(1), common.Hash{}, db)

	sv1.SetAccount(acc1Addr, acc1)
	accRetrieved := sv1.GetAccount(acc1Addr)

	assert.Equal(acc1.Address, accRetrieved.Address)
	assert.Equal(acc1.Sequence, accRetrieved.Sequence)
	assert.Equal(acc1.Balance.String(), accRetrieved.Balance.String())

	log.Infof(">>>>> Original account1\n")
	log.Infof("Address: %v\n", acc1.Address)
	log.Infof("Sequence: %v\n", acc1.Sequence)
	log.Infof("Balance: %v\n\n", acc1.Balance)

	log.Infof(">>>>> Retrieved account\n")
	log.Infof("Address: %v\n", accRetrieved.Address)
	log.Infof("Sequence: %v\n", accRetrieved.Sequence)
	log.Infof("Balance: %v\n", accRetrieved.Balance)
}

func TestStoreViewSplitRuleAccess(t *testing.T) {
	assert := assert.New(t)

	db := backend.NewMemDatabase()
	sv := NewStoreView(uint64(1), common.Hash{}, db)
	_, initiatorPubKey, err := crypto.TEST_GenerateKeyPairWithSeed("initiator")
	assert.Nil(err)

	initiatorAddr := initiatorPubKey.Address()

	rid1 := "rid1"
	sc1 := &types.SplitRule{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid1,
		EndBlockHeight:   100,
	}

	rid2 := "rid2"
	sc2 := &types.SplitRule{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid2,
		EndBlockHeight:   17,
	}

	rid3 := "rid3"
	sc3 := &types.SplitRule{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid3,
		EndBlockHeight:   28,
	}

	sv.SetSplitRule(rid1, sc1)
	sv.SetSplitRule(rid2, sc2)
	sv.SetSplitRule(rid3, sc3)

	retrievedSc1 := sv.GetSplitRule(rid1)
	retrievedSc2 := sv.GetSplitRule(rid2)
	retrievedSc3 := sv.GetSplitRule(rid3)

	log.Infof("Original SplitRule  #1: %v\n", sc1)
	log.Infof("Retrieved SplitRule #1: %v\n\n", retrievedSc1)
	assert.Equal(sc1.String(), retrievedSc1.String())

	log.Infof("Original SplitRule  #2: %v\n", sc2)
	log.Infof("Retrieved SplitRule #2: %v\n\n", retrievedSc2)
	assert.Equal(sc2.String(), retrievedSc2.String())

	log.Infof("Original SplitRule  #3: %v\n", sc3)
	log.Infof("Retrieved SplitRule #3: %v\n\n", retrievedSc3)
	assert.Equal(sc3.String(), retrievedSc3.String())

	sv.DeleteSplitRule(rid1)
	assert.Nil(sv.GetSplitRule(rid1))
	assert.NotNil(sv.GetSplitRule(rid2))
	assert.NotNil(sv.GetSplitRule(rid3))

	sv.DeleteExpiredSplitRules(29)
	assert.Nil(sv.GetSplitRule(rid1))
	assert.Nil(sv.GetSplitRule(rid2))
	assert.Nil(sv.GetSplitRule(rid3))

	sv.SetSplitRule(rid1, sc1)
	sv.SetSplitRule(rid2, sc2)
	sv.SetSplitRule(rid3, sc3)
	sv.DeleteExpiredSplitRules(19)
	assert.NotNil(sv.GetSplitRule(rid1))
	assert.Nil(sv.GetSplitRule(rid2))
	assert.NotNil(sv.GetSplitRule(rid3))
}

func TestRevertAndPruneStoreView(t *testing.T) {
	assert := assert.New(t)

	_, pubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)

	initCoin := types.Coins{ThetaWei: big.NewInt(786), TFuelWei: big.NewInt(0)}
	acc1Addr := pubKey.Address()
	acc1 := &types.Account{
		Address:  acc1Addr,
		Sequence: 173,
		Balance:  initCoin,
	}

	db := backend.NewMemDatabase()
	sv := NewStoreView(uint64(1), common.Hash{}, db)

	sv.SetAccount(acc1Addr, acc1)
	accRetrieved := sv.GetAccount(acc1Addr)

	assert.Equal(acc1.Address, accRetrieved.Address)
	assert.Equal(acc1.Sequence, accRetrieved.Sequence)
	assert.Equal(acc1.Balance.String(), accRetrieved.Balance.String())

	key1 := common.Hash(common.BytesToHash([]byte{1}))
	value1 := common.Hash(common.BytesToHash([]byte{11}))
	sv.SetState(acc1Addr, key1, value1)
	root1 := sv.Save()
	assert.Equal(value1, sv.GetState(acc1Addr, key1))

	hashMap1 := make(map[common.Hash]bool)
	for it := sv.store.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			ref, _ := db.CountReference(hash[:])
			assert.Equal(1, ref)

			hashMap1[it.Hash()] = true
		}
	}

	value2 := common.Hash(common.BytesToHash([]byte{22}))
	sv.SetState(acc1Addr, key1, value2)
	root2 := sv.Save()
	assert.Equal(value2, sv.GetState(acc1Addr, key1))

	hashMap2 := make(map[common.Hash]bool)
	for it := sv.store.NodeIterator(nil); it.Next(true); {
		if it.Hash() != (common.Hash{}) {
			hash := it.Hash()
			ref, _ := db.CountReference(hash[:])
			assert.Equal(1, ref)

			hashMap2[it.Hash()] = true
		}
	}

	sv.RevertToSnapshot(root1)
	assert.Equal(value1, sv.GetState(acc1Addr, key1))
	sv.Prune()

	for hash := range hashMap1 {
		has, _ := db.Has(hash[:])
		assert.False(has)
	}

	for hash := range hashMap2 {
		has, _ := db.Has(hash[:])
		assert.True(has)
	}

	sv.RevertToSnapshot(root2)
	assert.Equal(value2, sv.GetState(acc1Addr, key1))
}

func TestGetAndUpdateValidatorCandidatePool(t *testing.T) {
	assert := assert.New(t)

	sourceAddr1 := common.HexToAddress("0x111")
	stake1Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(1000), core.MinValidatorStakeDeposit)
	stake1Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(4000), core.MinValidatorStakeDeposit)

	sourceAddr2 := common.HexToAddress("0x222")
	stake2Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(8000), core.MinValidatorStakeDeposit)
	stake2Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(9000), core.MinValidatorStakeDeposit)

	sourceAddr3 := common.HexToAddress("0x333")
	stake3Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(500), core.MinValidatorStakeDeposit)
	stake3Amount2 := new(big.Int).Mul(new(big.Int).SetUint64(200), core.MinValidatorStakeDeposit)
	stake3Amount3 := new(big.Int).Mul(new(big.Int).SetUint64(900), core.MinValidatorStakeDeposit)

	sourceAddr4 := common.HexToAddress("0x444")
	stake4Amount1 := new(big.Int).Mul(new(big.Int).SetUint64(5000), core.MinValidatorStakeDeposit)

	holderAddr1 := common.HexToAddress("0xf01")
	holderAddr2 := common.HexToAddress("0xf02")
	holderAddr3 := common.HexToAddress("0xf03")
	holderAddr4 := common.HexToAddress("0xf04")

	vcp := &core.ValidatorCandidatePool{}

	assert.Nil(vcp.DepositStake(sourceAddr1, holderAddr1, stake1Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr2, holderAddr1, stake2Amount1))
	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr1, stake3Amount2))

	assert.Nil(vcp.DepositStake(sourceAddr1, holderAddr2, stake1Amount2))
	assert.Nil(vcp.DepositStake(sourceAddr2, holderAddr2, stake2Amount2))
	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr2, stake3Amount2))

	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr3, stake3Amount1))

	assert.Nil(vcp.DepositStake(sourceAddr3, holderAddr4, stake3Amount3))
	assert.Nil(vcp.DepositStake(sourceAddr4, holderAddr4, stake4Amount1))

	db := backend.NewMemDatabase()
	sv := NewStoreView(uint64(1), common.Hash{}, db)

	sv.UpdateValidatorCandidatePool(vcp)
	vcp1 := sv.GetValidatorCandidatePool()
	assert.True(compareValidatorCandidatePools(vcp, vcp1))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("vcp:  %v", vcp)
	log.Infof("vcp1: %v", vcp1)
	log.Infof("-------------------------------------------------")
	log.Infof("")

	height := uint64(99999)
	assert.Nil(vcp.WithdrawStake(sourceAddr1, holderAddr2, height))
	assert.Nil(vcp.WithdrawStake(sourceAddr2, holderAddr1, height))
	assert.Nil(vcp.WithdrawStake(sourceAddr3, holderAddr4, height))
	assert.Nil(vcp.WithdrawStake(sourceAddr4, holderAddr4, height))

	sv.UpdateValidatorCandidatePool(vcp)
	vcp2 := sv.GetValidatorCandidatePool()
	assert.True(compareValidatorCandidatePools(vcp, vcp2))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("vcp:  %v", vcp)
	log.Infof("vcp2: %v", vcp2)
	log.Infof("-------------------------------------------------")
	log.Infof("")

	height1 := height + core.ReturnLockingPeriod
	vcp.ReturnStakes(height1)

	sv.UpdateValidatorCandidatePool(vcp)
	vcp3 := sv.GetValidatorCandidatePool()
	assert.True(compareValidatorCandidatePools(vcp, vcp3))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("vcp:  %v", vcp)
	log.Infof("vcp2: %v", vcp3)
	log.Infof("-------------------------------------------------")
	log.Infof("")
}

func TestGetAndUpdateHeightList(t *testing.T) {
	assert := assert.New(t)

	db := backend.NewMemDatabase()
	sv := NewStoreView(uint64(1), common.Hash{}, db)

	hl := &types.HeightList{}
	sv.UpdateStakeTransactionHeightList(hl)
	hl1 := sv.GetStakeTransactionHeightList()
	assert.True(compareHeightList(hl, hl1))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("hl:  %v", hl)
	log.Infof("hl1: %v", hl1)
	log.Infof("-------------------------------------------------")
	log.Infof("")

	hl.Append(997)
	sv.UpdateStakeTransactionHeightList(hl)
	hl2 := sv.GetStakeTransactionHeightList()
	assert.True(compareHeightList(hl, hl2))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("hl:  %v", hl)
	log.Infof("hl2: %v", hl2)
	log.Infof("-------------------------------------------------")
	log.Infof("")

	hl.Append(1776)
	hl.Append(9823)
	hl.Append(827372)
	hl.Append(9828376)
	hl.Append(10091192)
	sv.UpdateStakeTransactionHeightList(hl)
	hl3 := sv.GetStakeTransactionHeightList()
	assert.True(compareHeightList(hl, hl3))

	log.Infof("")
	log.Infof("-------------------------------------------------")
	log.Infof("hl:  %v", hl)
	log.Infof("hl3: %v", hl3)
	log.Infof("-------------------------------------------------")
	log.Infof("")
}

// ------------------------ Utilities ------------------------ //

func compareValidatorCandidatePools(vcp1, vcp2 *core.ValidatorCandidatePool) bool {
	if len(vcp1.SortedCandidates) != len(vcp2.SortedCandidates) {
		return false
	}

	numCands := len(vcp1.SortedCandidates)
	for i := 0; i < numCands; i++ {
		cand1 := vcp1.SortedCandidates[i]
		cand2 := vcp2.SortedCandidates[i]

		if cand1.Holder != cand2.Holder {
			return false
		}

		if len(cand1.Stakes) != len(cand2.Stakes) {
			return false
		}

		numStakes := len(cand1.Stakes)
		for j := 0; j < numStakes; j++ {
			s1 := cand1.Stakes[j]
			s2 := cand2.Stakes[j]

			if s1.Amount.Cmp(s2.Amount) != 0 {
				return false
			}
			if s1.ReturnHeight != s2.ReturnHeight {
				return false
			}
			if s1.Source != s2.Source {
				return false
			}
			if s1.Withdrawn != s2.Withdrawn {
				return false
			}
		}
	}

	return true
}

func compareHeightList(hl1, hl2 *types.HeightList) bool {
	if len(hl1.Heights) != len(hl2.Heights) {
		return false
	}

	numHeights := len(hl1.Heights)
	for i := 0; i < numHeights; i++ {
		if hl1.Heights[i] != hl2.Heights[i] {
			return false
		}
	}

	return true
}
