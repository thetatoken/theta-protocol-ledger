package state

import (
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestLedgerStateBasics(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint32(127)
	initRootHash := common.Hash{}
	ls.ResetState(initHeight, initRootHash)

	// ChainID
	assert.Equal(chainID, ls.GetChainID())

	// Height
	assert.Equal(initHeight, ls.Height())
	assert.Equal(initHeight, ls.Delivered().Height())
	assert.Equal(initHeight, ls.Checked().Height())

	// SlashIntents
	si := types.SlashIntent{
		Address:         common.HexToAddress("abcd1234"),
		ReserveSequence: 187,
		Proof:           common.Bytes("hereistheproof"),
	}
	assert.Equal(0, len(ls.GetSlashIntents()))
	ls.AddSlashIntent(si)
	ls.AddSlashIntent(si)
	ls.AddSlashIntent(si)
	assert.Equal(3, len(ls.GetSlashIntents()))
	ls.ClearSlashIntents()
	assert.Equal(0, len(ls.GetSlashIntents()))

	// CoinbaseTransactionProcessed
	ls.SetCoinbaseTransactionProcessed(true)
	assert.True(ls.CoinbaseTransactinProcessed())
	ls.SetCoinbaseTransactionProcessed(false)
	assert.False(ls.CoinbaseTransactinProcessed())

	// ValidatorDiff
	_, va1PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("va1")
	assert.Nil(err)
	_, va2PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("va2")
	assert.Nil(err)
	va1 := core.NewValidator(va1PubKey.ToBytes(), uint64(100))
	va2 := core.NewValidator(va2PubKey.ToBytes(), uint64(999))
	vaDiff := []*core.Validator{&va1, &va2}
	ls.SetValidatorDiff(vaDiff)
	assert.Equal(2, len(ls.GetAndClearValidatorDiff()))
	assert.Equal(0, len(ls.GetAndClearValidatorDiff()))
}

func TestLedgerStateAccountCommit(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint32(127)
	initRootHash := common.Hash{}
	ls.ResetState(initHeight, initRootHash)

	// Account and Commit
	_, acc1PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)
	initCoin := types.Coin{"CoolToken", int64(956)}
	acc1 := &types.Account{
		PubKey:   acc1PubKey,
		Sequence: 657,
		Balance:  types.Coins{initCoin},
	}
	acc1Addr := acc1.PubKey.Address()
	ls.SetAccount(acc1Addr, acc1)
	log.Infof("Account added\n")

	rootHashChecked1 := ls.Checked().Hash()
	rootHashDelivered1 := ls.Delivered().Hash()
	assert.NotEqual(rootHashChecked1, rootHashDelivered1) // root hash of the Delivered tree should have changed due to SetAccount()

	log.Infof("Before commit, rootHashChecked  : %v\n", rootHashChecked1.Hex())
	log.Infof("Before commit, rootHashDelivered: %v\n", rootHashDelivered1.Hex())

	rootHash2 := ls.Commit()
	log.Infof("Root hash returned by Commit()  : %v\n", rootHash2.Hex())

	assert.Equal(initHeight+1, ls.Height())
	assert.Equal(initHeight+1, ls.Checked().Height())
	assert.Equal(initHeight+1, ls.Delivered().Height())

	retrivedAcc1 := ls.GetAccount(acc1Addr)
	assert.Equal(acc1.String(), retrivedAcc1.String())
	retrievedAcc1CheckedView := ls.Checked().GetAccount(acc1Addr)
	assert.Equal(acc1.String(), retrievedAcc1CheckedView.String())
	retrievedAcc1DeliveredView := ls.Delivered().GetAccount(acc1Addr)
	assert.Equal(acc1.String(), retrievedAcc1DeliveredView.String())

	rootHashChecked2 := ls.Checked().Hash()
	rootHashDelivered2 := ls.Checked().Hash()
	assert.Equal(rootHash2, rootHashChecked2)
	assert.Equal(rootHash2, rootHashDelivered2) // root hash of both the Checked and Delivered tree should be the same after the Commit()

	log.Infof("After commit, rootHashChecked   : %v\n", rootHashChecked2.Hex())
	log.Infof("After commit, rootHashDelivered : %v\n", rootHashDelivered2.Hex())

	log.Infof("Original account : %v\n", acc1)
	log.Infof("Retrieved account: %v\n", retrivedAcc1)
}

func TestLedgerStateSplitContractCommit(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint32(127)
	initRootHash := common.Hash{}
	ls.ResetState(initHeight, initRootHash)

	_, acc1PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)
	acc1Addr := acc1PubKey.Address()

	// SplitContract and Commit
	rid1 := common.Bytes("rid1")
	sc1 := &types.SplitContract{
		InitiatorAddress: acc1Addr,
		ResourceID:       rid1,
		EndBlockHeight:   342,
	}
	rid2 := common.Bytes("rid2")
	sc2 := &types.SplitContract{
		InitiatorAddress: acc1Addr,
		ResourceID:       rid2,
		EndBlockHeight:   56,
	}
	ls.AddSplitContract(sc1)
	ls.AddSplitContract(sc2)
	log.Infof("Split contracts added\n")

	rootHashChecked1 := ls.Checked().Hash()
	rootHashDelivered1 := ls.Delivered().Hash()
	assert.NotEqual(rootHashChecked1, rootHashDelivered1) // root hash of the Delivered tree should have changed due to AddSplitContract()

	log.Infof("Before any commit, rootHashChecked  : %v\n", rootHashChecked1.Hex())
	log.Infof("Before any commit, rootHashDelivered: %v\n", rootHashDelivered1.Hex())

	rootHash2 := ls.Commit()
	log.Infof("Root hash returned by Commit() #1   : %v\n", rootHash2.Hex())

	assert.Equal(initHeight+1, ls.Height())
	assert.Equal(initHeight+1, ls.Checked().Height())
	assert.Equal(initHeight+1, ls.Delivered().Height())

	rootHashChecked2 := ls.Checked().Hash()
	rootHashDelivered2 := ls.Checked().Hash()
	assert.Equal(rootHash2, rootHashChecked2)
	assert.Equal(rootHash2, rootHashDelivered2) // root hash of both the Checked and Delivered tree should be the same after the Commit()

	log.Infof("After commit #1, rootHashChecked    : %v\n", rootHashChecked2.Hex())
	log.Infof("After commit #1, rootHashDelivered  : %v\n", rootHashDelivered2.Hex())

	assert.True(ls.SplitContractExists(rid1))
	assert.True(ls.SplitContractExists(rid2))
	assert.NotNil(ls.GetSplitContract(rid1))
	assert.NotNil(ls.GetSplitContract(rid2))

	ls.DeleteExpiredSplitContracts(123)

	assert.True(ls.SplitContractExists(rid1))
	assert.False(ls.SplitContractExists(rid2))
	assert.NotNil(ls.GetSplitContract(rid1))
	assert.Nil(ls.GetSplitContract(rid2))

	log.Infof("Before updating sc1, retrieved sc1: %v\n", ls.GetSplitContract(rid1))
	sc1.EndBlockHeight = 567
	assert.True(ls.UpdateSplitContract(sc1))
	sc2.EndBlockHeight = 423
	assert.False(ls.UpdateSplitContract(sc2)) // sc2 not exists anymore
	log.Infof("Split contract sc1 updated")
	log.Infof("After updating sc1, retrieved sc1 : %v\n", ls.GetSplitContract(rid1))

	ls.DeleteExpiredSplitContracts(500)
	assert.True(ls.SplitContractExists(rid1)) // sc1.EndBlockHeight should have increased
	assert.False(ls.SplitContractExists(rid2))

	ls.DeleteExpiredSplitContracts(900)
	assert.False(ls.SplitContractExists(rid1))
	assert.False(ls.SplitContractExists(rid2))
	log.Infof("Expired split contracts deleted")

	rootHashChecked3 := ls.Checked().Hash()
	rootHashDelivered3 := ls.Delivered().Hash()
	assert.Equal(rootHash2, rootHashChecked3)             // root hash of the Checked tree should not have changed
	assert.NotEqual(rootHashChecked3, rootHashDelivered3) // root hash of the Delivered tree should have changed
	log.Infof("Before commit #2, rootHashChecked   : %v\n", rootHashChecked3.Hex())
	log.Infof("Before commit #2, rootHashDelivered : %v\n", rootHashDelivered3.Hex())

	rootHash4 := ls.Commit()
	log.Infof("Root hash returned by Commit() #2   : %v\n", rootHash4.Hex())

	assert.Equal(initHeight+2, ls.Height())
	assert.Equal(initHeight+2, ls.Checked().Height())
	assert.Equal(initHeight+2, ls.Delivered().Height())

	rootHashChecked4 := ls.Checked().Hash()
	rootHashDelivered4 := ls.Checked().Hash()
	assert.Equal(rootHash4, rootHashChecked4)
	assert.Equal(rootHash4, rootHashDelivered4) // root hash of both the Checked and Delivered tree should be the same after the Commit()

	log.Infof("After commit #2, rootHashChecked    : %v\n", rootHashChecked4.Hex())
	log.Infof("After commit #2, rootHashDelivered  : %v\n", rootHashDelivered4.Hex())
}
