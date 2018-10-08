package ledger

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	dp "github.com/thetatoken/ukulele/dispatcher"
	exec "github.com/thetatoken/ukulele/ledger/execution"
	"github.com/thetatoken/ukulele/ledger/types"
	mp "github.com/thetatoken/ukulele/mempool"
	"github.com/thetatoken/ukulele/p2p"
	p2psim "github.com/thetatoken/ukulele/p2p/simulation"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestLedgerSetup(t *testing.T) {
	assert := assert.New(t)

	_, ledger, mempool := newTestLedger()
	assert.NotNil(ledger)
	assert.NotNil(mempool)
}

func TestLedgerScreenTx(t *testing.T) {
	assert := assert.New(t)

	chainID, ledger, _ := newTestLedger()
	numInAccs := 1
	accOut, accIns := prepareInitLedgerState(ledger, numInAccs)

	sendTxBytes := newRawSendTx(chainID, 1, true, accOut, accIns[0])
	res := ledger.ScreenTx(sendTxBytes)
	assert.True(res.IsOK(), res.Message)

	coinbaseTxBytes := newRawCoinbaseTx(chainID, ledger, 1)
	res = ledger.ScreenTx(coinbaseTxBytes)
	assert.Equal(result.CodeUnauthorizedTx, res.Code, res.Message)
}

func TestLedgerProposerBlockTxs(t *testing.T) {
	assert := assert.New(t)

	chainID, ledger, mempool := newTestLedger()
	numInAccs := 200
	accOut, accIns := prepareInitLedgerState(ledger, numInAccs)

	// Insert send transactions into the mempool
	numMempoolTxs := 200
	rawSendTxs := []common.Bytes{}
	for idx := 0; idx < numMempoolTxs; idx++ {
		sendTxBytes := newRawSendTx(chainID, 1, true, accOut, accIns[idx])
		err := mempool.InsertTransaction(mp.CreateMempoolTransaction(sendTxBytes))
		assert.Nil(err, fmt.Sprintf("Mempool insertion error: %v", err))
		rawSendTxs = append(rawSendTxs, sendTxBytes)
	}
	assert.Equal(numMempoolTxs, mempool.Size())

	// Propose block transactions
	_, blockTxs, res := ledger.ProposeBlockTxs()

	// Transaction counts sanity checks
	expectedTotalNumTx := core.MaxNumRegularTxsPerBlock + 1
	assert.Equal(expectedTotalNumTx, len(blockTxs))
	assert.True(res.IsOK())
	assert.Equal(numMempoolTxs-expectedTotalNumTx+1, mempool.Size())

	// Transaction sanity checks
	for idx := 0; idx < expectedTotalNumTx; idx++ {
		rawTx := blockTxs[idx]
		tx, err := types.TxFromBytes(rawTx)
		assert.Nil(err)
		switch tx.(type) {
		case *types.CoinbaseTx:
			assert.Equal(0, idx) // The first tx needs to be a coinbase transaction
			coinbaseTx := tx.(*types.CoinbaseTx)
			signBytes := coinbaseTx.SignBytes(chainID)
			ledger.consensus.PrivateKey().PublicKey().VerifySignature(signBytes, coinbaseTx.Proposer.Signature)
		case *types.SendTx:
			assert.True(idx > 0)
			assert.Equal(rawTx, rawSendTxs[idx-1]) // mempool should works like a FIFO queue
		}
	}
}

func TestLedgerApplyBlockTxs(t *testing.T) {
	assert := assert.New(t)

	chainID, ledger, _ := newTestLedger()
	numInAccs := 5
	accOut, accIns := prepareInitLedgerState(ledger, numInAccs)

	coinbaseTxBytes := newRawCoinbaseTx(chainID, ledger, 1)
	sendTx1Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[0])
	sendTx2Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[1])
	sendTx3Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[2])
	sendTx4Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[3])
	sendTx5Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[4])

	blockRawTxs := []common.Bytes{
		coinbaseTxBytes,
		sendTx1Bytes, sendTx2Bytes, sendTx3Bytes, sendTx4Bytes, sendTx5Bytes,
	}
	expectedStateRoot := common.HexToHash("79d7136e705f0f77228fc04db28e5e583f60e1cd8166f59b65b2be8e70866594")

	res := ledger.ApplyBlockTxs(blockRawTxs, expectedStateRoot)
	assert.True(res.IsOK(), res.Message)

	//
	// Account balance sanity checks
	//

	// Validator balance
	validators := ledger.valMgr.GetValidatorSetForEpoch(0).Validators()
	for _, val := range validators {
		valPk := val.PublicKey()
		valAddr := (&valPk).Address()
		valAcc := ledger.state.GetAccount(valAddr)
		expectedValBal := types.NewCoins(100000000317, 20000)
		assert.NotNil(valAcc)
		assert.Equal(expectedValBal, valAcc.Balance)
	}

	// Output account balance
	accOutAfter := ledger.state.GetAccount(accOut.PubKey.Address())
	expectedAccOutBal := types.NewCoins(700075, 3)
	assert.Equal(expectedAccOutBal, accOutAfter.Balance)

	// Input account balance
	expectedAccInBal := types.NewCoins(899985, 49997)
	for idx, _ := range accIns {
		accInAddr := accIns[idx].Account.PubKey.Address()
		accInAfter := ledger.state.GetAccount(accInAddr)
		assert.Equal(expectedAccInBal, accInAfter.Balance)
	}
}

// ----------- Utilities ----------- //

func newTestLedger() (chainID string, ledger *Ledger, mempool *mp.Mempool) {
	chainID = "test_chain_id"
	peerID := "peer0"
	proposerSeed := "proposer"

	db := backend.NewMemDatabase()
	consensus := exec.NewTestConsensusEngine(proposerSeed)
	valMgr := newTesetValidatorManager(consensus)
	p2psimnet := p2psim.NewSimnetWithHandler(nil)
	messenger := p2psimnet.AddEndpoint(peerID)
	mempool = newTestMempool(peerID, messenger)
	ledger = NewLedger(chainID, db, consensus, valMgr, mempool)
	mempool.SetLedger(ledger)

	messenger.Start()
	mempool.Start()

	initHeight := uint64(1)
	initRootHash := common.Hash{}
	ledger.ResetState(initHeight, initRootHash)

	return chainID, ledger, mempool
}

func newTesetValidatorManager(consensus core.ConsensusEngine) core.ValidatorManager {
	proposerPubKeyBytes := consensus.PrivateKey().PublicKey().ToBytes()
	propser := core.NewValidator(proposerPubKeyBytes, uint64(999))

	_, val2PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("val2")
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair with seed: %v", err))
	}
	val2 := core.NewValidator(val2PubKey.ToBytes(), uint64(100))

	valSet := core.NewValidatorSet()
	valSet.AddValidator(propser)
	valSet.AddValidator(val2)
	valMgr := exec.NewTestValidatorManager(propser, valSet)

	return valMgr
}

func newTestMempool(peerID string, messenger p2p.Network) *mp.Mempool {
	dispatcher := dp.NewDispatcher(messenger)
	mempool := mp.CreateMempool(dispatcher)
	txMsgHandler := mp.CreateMempoolMessageHandler(mempool)
	messenger.RegisterMessageHandler(txMsgHandler)
	return mempool
}

func prepareInitLedgerState(ledger *Ledger, numInAccs int) (accOut types.PrivAccount, accIns []types.PrivAccount) {
	validators := ledger.valMgr.GetValidatorSetForEpoch(0).Validators()
	for _, val := range validators {
		valPubKey := val.PublicKey()
		valAccount := &types.Account{
			PubKey:                 &valPubKey,
			LastUpdatedBlockHeight: 1,
			Balance:                types.NewCoins(100000000000, 1000),
		}
		ledger.state.SetAccount(valPubKey.Address(), valAccount)
	}

	accOut = types.MakeAccWithInitBalance("accOut", types.NewCoins(700000, 3))
	ledger.state.SetAccount(accOut.Account.PubKey.Address(), &accOut.Account)

	for i := 0; i < numInAccs; i++ {
		secret := "in_secret_" + strconv.FormatInt(int64(i), 16)
		accIn := types.MakeAccWithInitBalance(secret, types.NewCoins(900000, 50000))
		accIns = append(accIns, accIn)
		ledger.state.SetAccount(accIn.Account.PubKey.Address(), &accIn.Account)
	}

	ledger.state.Commit()

	return accOut, accIns
}

func newRawCoinbaseTx(chainID string, ledger *Ledger, sequence int) common.Bytes {
	vaList := ledger.valMgr.GetValidatorSetForEpoch(0).Validators()
	if len(vaList) < 2 {
		panic("Insufficient number of validators")
	}
	outputs := []types.TxOutput{}
	for _, val := range vaList {
		valPk := val.PublicKey()
		output := types.TxOutput{(&valPk).Address(), types.NewCoins(317, 0)}
		outputs = append(outputs, output)
	}

	proposerSk := ledger.consensus.PrivateKey()
	proposerPk := proposerSk.PublicKey()
	coinbaseTx := &types.CoinbaseTx{
		Proposer:    types.TxInput{Address: proposerPk.Address(), PubKey: proposerPk, Sequence: uint64(sequence)},
		Outputs:     outputs,
		BlockHeight: 2,
	}

	signBytes := coinbaseTx.SignBytes(chainID)
	sig, err := proposerSk.Sign(signBytes)
	if err != nil {
		panic("Failed to sign the coinbase transaction")
	}
	if !coinbaseTx.SetSignature(proposerPk.Address(), sig) {
		panic("Failed to set signature for the coinbase transaction")
	}

	coinbaseTxBytes := types.TxToBytes(coinbaseTx)
	return coinbaseTxBytes
}

func newRawSendTx(chainID string, sequence int, addPubKey bool, accOut, accIn types.PrivAccount) common.Bytes {
	sendTx := &types.SendTx{
		Gas: 0,
		Fee: types.NewCoins(0, 3),
		Inputs: []types.TxInput{
			{
				Sequence: uint64(sequence),
				PubKey:   accIn.PubKey,
				Address:  accIn.PubKey.Address(),
				Coins:    types.NewCoins(15, 3),
			},
		},
		Outputs: []types.TxOutput{
			{
				Address: accOut.PubKey.Address(),
				Coins:   types.NewCoins(15, 0),
			},
		},
	}

	signBytes := sendTx.SignBytes(chainID)
	inAccs := []types.PrivAccount{accIn}
	for idx, in := range sendTx.Inputs {
		inAcc := inAccs[idx]
		sig, err := inAcc.PrivKey.Sign(signBytes)
		if err != nil {
			panic("Failed to sign the coinbase transaction")
		}
		sendTx.SetSignature(in.Address, sig)

		if !addPubKey {
			sendTx.Inputs[idx].PubKey = nil
		}
	}

	sendTxBytes := types.TxToBytes(sendTx)
	return sendTxBytes
}
