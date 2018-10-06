package state

import (
	"encoding/hex"
	"testing"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestStoreViewBasics(t *testing.T) {
	assert := assert.New(t)

	initHeight := uint32(1)
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

	initCoin := types.Coins{ThetaWei: int64(786)}
	acc1 := &types.Account{
		PubKey:   pubKey,
		Sequence: 173,
		Balance:  initCoin,
	}
	acc1Addr := acc1.PubKey.Address()

	db := backend.NewMemDatabase()
	sv1 := NewStoreView(uint32(1), common.Hash{}, db)

	sv1.SetAccount(acc1Addr, acc1)
	accRetrieved := sv1.GetAccount(acc1Addr)

	assert.Equal(acc1.PubKey.ToBytes(), accRetrieved.PubKey.ToBytes())
	assert.Equal(acc1.Sequence, accRetrieved.Sequence)
	assert.Equal(acc1.Balance.String(), accRetrieved.Balance.String())

	log.Infof(">>>>> Original account1\n")
	log.Infof("PubKey: %v\n", acc1.PubKey)
	log.Infof("PubKey Bytes: %v\n", hex.EncodeToString(acc1.PubKey.ToBytes()))
	log.Infof("Sequence: %v\n", acc1.Sequence)
	log.Infof("Balance: %v\n\n", acc1.Balance)

	log.Infof(">>>>> Retrieved account\n")
	log.Infof("PubKey: %v\n", accRetrieved.PubKey)
	log.Infof("PubKey Bytes: %v\n", hex.EncodeToString(accRetrieved.PubKey.ToBytes()))
	log.Infof("Sequence: %v\n", accRetrieved.Sequence)
	log.Infof("Balance: %v\n", accRetrieved.Balance)
}

func TestStoreViewSplitContractAccess(t *testing.T) {
	assert := assert.New(t)

	db := backend.NewMemDatabase()
	sv := NewStoreView(uint32(1), common.Hash{}, db)
	_, initiatorPubKey, err := crypto.TEST_GenerateKeyPairWithSeed("initiator")
	assert.Nil(err)

	initiatorAddr := initiatorPubKey.Address()

	rid1 := common.Bytes("rid1")
	sc1 := &types.SplitContract{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid1,
		EndBlockHeight:   100,
	}

	rid2 := common.Bytes("rid2")
	sc2 := &types.SplitContract{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid2,
		EndBlockHeight:   17,
	}

	rid3 := common.Bytes("rid3")
	sc3 := &types.SplitContract{
		InitiatorAddress: initiatorAddr,
		ResourceID:       rid3,
		EndBlockHeight:   28,
	}

	sv.SetSplitContract(rid1, sc1)
	sv.SetSplitContract(rid2, sc2)
	sv.SetSplitContract(rid3, sc3)

	retrievedSc1 := sv.GetSplitContract(rid1)
	retrievedSc2 := sv.GetSplitContract(rid2)
	retrievedSc3 := sv.GetSplitContract(rid3)

	log.Infof("Original SplitContract  #1: %v\n", sc1)
	log.Infof("Retrieved SplitContract #1: %v\n\n", retrievedSc1)
	assert.Equal(sc1.String(), retrievedSc1.String())

	log.Infof("Original SplitContract  #2: %v\n", sc2)
	log.Infof("Retrieved SplitContract #2: %v\n\n", retrievedSc2)
	assert.Equal(sc2.String(), retrievedSc2.String())

	log.Infof("Original SplitContract  #3: %v\n", sc3)
	log.Infof("Retrieved SplitContract #3: %v\n\n", retrievedSc3)
	assert.Equal(sc3.String(), retrievedSc3.String())

	sv.DeleteSplitContract(rid1)
	assert.Nil(sv.GetSplitContract(rid1))
	assert.NotNil(sv.GetSplitContract(rid2))
	assert.NotNil(sv.GetSplitContract(rid3))

	sv.DeleteExpiredSplitContracts(29)
	assert.Nil(sv.GetSplitContract(rid1))
	assert.Nil(sv.GetSplitContract(rid2))
	assert.Nil(sv.GetSplitContract(rid3))

	sv.SetSplitContract(rid1, sc1)
	sv.SetSplitContract(rid2, sc2)
	sv.SetSplitContract(rid3, sc3)
	sv.DeleteExpiredSplitContracts(19)
	assert.NotNil(sv.GetSplitContract(rid1))
	assert.Nil(sv.GetSplitContract(rid2))
	assert.NotNil(sv.GetSplitContract(rid3))
}
