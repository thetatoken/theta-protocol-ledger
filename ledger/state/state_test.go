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

func TestLedgerStateBasics(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint64(127)
	initRootHash := common.Hash{}
	initBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    initHeight,
			StateHash: initRootHash,
		},
	}
	//ls.ResetState(initHeight, initRootHash)
	ls.ResetState(initBlock)

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
	assert.Equal(0, len(ls.Delivered().GetSlashIntents()))
	ls.Delivered().AddSlashIntent(si)
	ls.Delivered().AddSlashIntent(si)
	ls.Delivered().AddSlashIntent(si)
	assert.Equal(3, len(ls.Delivered().GetSlashIntents()))
	ls.Delivered().ClearSlashIntents()
	assert.Equal(0, len(ls.Delivered().GetSlashIntents()))

	// CoinbaseTransactionProcessed
	ls.Delivered().SetCoinbaseTransactionProcessed(true)
	assert.True(ls.Delivered().CoinbaseTransactinProcessed())
	ls.Delivered().SetCoinbaseTransactionProcessed(false)
	assert.False(ls.Delivered().CoinbaseTransactinProcessed())
}

func TestLedgerStateAccountCommit(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint64(127)
	initRootHash := common.Hash{}
	initBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    initHeight,
			StateHash: initRootHash,
		},
	}
	//ls.ResetState(initHeight, initRootHash)
	ls.ResetState(initBlock)

	// Account and Commit
	_, acc1PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)
	initCoin := types.Coins{ThetaWei: big.NewInt(956), TFuelWei: big.NewInt(0)}
	acc1 := &types.Account{
		Address:  acc1PubKey.Address(),
		Sequence: 657,
		Balance:  initCoin,
	}
	acc1Addr := acc1.Address
	ls.Delivered().SetAccount(acc1Addr, acc1)
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

	retrivedAcc1 := ls.Delivered().GetAccount(acc1Addr)
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

func TestLedgerStateSplitRuleCommit(t *testing.T) {
	assert := assert.New(t)

	chainID := "testchain"
	db := backend.NewMemDatabase()
	ls := NewLedgerState(chainID, db)

	initHeight := uint64(127)
	initRootHash := common.Hash{}
	initBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    initHeight,
			StateHash: initRootHash,
		},
	}
	//ls.ResetState(initHeight, initRootHash)
	ls.ResetState(initBlock)

	_, acc1PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("account1")
	assert.Nil(err)
	acc1Addr := acc1PubKey.Address()

	// SplitRule and Commit
	rid1 := "rid1"
	sc1 := &types.SplitRule{
		InitiatorAddress: acc1Addr,
		ResourceID:       rid1,
		EndBlockHeight:   342,
	}
	rid2 := "rid2"
	sc2 := &types.SplitRule{
		InitiatorAddress: acc1Addr,
		ResourceID:       rid2,
		EndBlockHeight:   56,
	}
	ls.Delivered().AddSplitRule(sc1)
	ls.Delivered().AddSplitRule(sc2)
	log.Infof("Split rules added\n")

	rootHashChecked1 := ls.Checked().Hash()
	rootHashDelivered1 := ls.Delivered().Hash()
	assert.NotEqual(rootHashChecked1, rootHashDelivered1) // root hash of the Delivered tree should have changed due to AddSplitRule()

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

	assert.True(ls.Delivered().SplitRuleExists(rid1))
	assert.True(ls.Delivered().SplitRuleExists(rid2))
	assert.NotNil(ls.Delivered().GetSplitRule(rid1))
	assert.NotNil(ls.Delivered().GetSplitRule(rid2))

	ls.Delivered().DeleteExpiredSplitRules(123)

	assert.True(ls.Delivered().SplitRuleExists(rid1))
	assert.False(ls.Delivered().SplitRuleExists(rid2))
	assert.NotNil(ls.Delivered().GetSplitRule(rid1))
	assert.Nil(ls.Delivered().GetSplitRule(rid2))

	log.Infof("Before updating sc1, retrieved sc1: %v\n", ls.Delivered().GetSplitRule(rid1))
	sc1.EndBlockHeight = 567
	assert.True(ls.Delivered().UpdateSplitRule(sc1))
	sc2.EndBlockHeight = 423
	assert.False(ls.Delivered().UpdateSplitRule(sc2)) // sc2 not exists anymore
	log.Infof("Split rule sc1 updated")
	log.Infof("After updating sc1, retrieved sc1 : %v\n", ls.Delivered().GetSplitRule(rid1))

	ls.Delivered().DeleteExpiredSplitRules(500)
	assert.True(ls.Delivered().SplitRuleExists(rid1)) // sc1.EndBlockHeight should have increased
	assert.False(ls.Delivered().SplitRuleExists(rid2))

	ls.Delivered().DeleteExpiredSplitRules(900)
	assert.False(ls.Delivered().SplitRuleExists(rid1))
	assert.False(ls.Delivered().SplitRuleExists(rid2))
	log.Infof("Expired split rules deleted")

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
