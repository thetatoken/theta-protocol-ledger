package ledger

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	sendTxBytes := newRawSendTx(chainID, 1, true, accOut, accIns[0], false)
	_, res := ledger.ScreenTx(sendTxBytes)
	assert.True(res.IsOK(), res.Message)

	coinbaseTxBytes := newRawCoinbaseTx(chainID, ledger, 1)
	_, res = ledger.ScreenTx(coinbaseTxBytes)
	assert.Equal(result.CodeUnauthorizedTx, res.Code, res.Message)
}

func TestLedgerProposerBlockTxs(t *testing.T) {
	assert := assert.New(t)

	chainID, ledger, mempool := newTestLedger()
	numInAccs := 2 * core.MaxNumRegularTxsPerBlock
	accOut, accIns := prepareInitLedgerState(ledger, numInAccs)

	// Insert send transactions into the mempool
	numMempoolTxs := 2 * core.MaxNumRegularTxsPerBlock
	rawSendTxs := []common.Bytes{}
	for idx := 0; idx < numMempoolTxs; idx++ {
		sequence := 1
		sendTxBytes := newRawSendTx(chainID, sequence, true, accOut, accIns[idx], true)
		err := mempool.InsertTransaction(sendTxBytes)
		assert.Nil(err, fmt.Sprintf("Mempool insertion error: %v", err))
		rawSendTxs = append(rawSendTxs, sendTxBytes)
	}
	assert.Equal(numMempoolTxs, mempool.Size())

	startTime := time.Now()

	// Propose block transactions
	_, blockTxs, res := ledger.ProposeBlockTxs()

	endTime := time.Now()
	elapsed := endTime.Sub(startTime)
	log.Infof("Execution time for block proposal: %v", elapsed)

	// Transaction counts sanity checks
	expectedTotalNumTx := core.MaxNumRegularTxsPerBlock + 1
	assert.Equal(expectedTotalNumTx, len(blockTxs))
	assert.True(res.IsOK())
	assert.Equal(numMempoolTxs-expectedTotalNumTx+1, mempool.Size())

	// Transaction sanity checks
	var prevSendTx *types.SendTx
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
			currSendTx := tx.(*types.SendTx)
			if prevSendTx != nil {
				// mempool should works like a priority queue, for the same type of tx (i.e. SendTx),
				// those with higher fee should get reaped first
				feeDiff := prevSendTx.Fee.Minus(currSendTx.Fee)
				assert.True(feeDiff.IsNonnegative())
				log.Infof("tx fee: %v, feeDiff: %v", currSendTx.Fee, feeDiff)
			}
			prevSendTx = currSendTx
		}
	}
}

func TestLedgerApplyBlockTxs(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	chainID, ledger, _ := newTestLedger()
	numInAccs := 5
	accOut, accIns := prepareInitLedgerState(ledger, numInAccs)

	coinbaseTxBytes := newRawCoinbaseTx(chainID, ledger, 1)
	sendTx1Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[0], false)
	sendTx2Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[1], false)
	sendTx3Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[2], false)
	sendTx4Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[3], false)
	sendTx5Bytes := newRawSendTx(chainID, 1, true, accOut, accIns[4], false)
	inAccInitGammaWei := accIns[0].Balance.GammaWei
	txFee := getMinimumTxFee()

	blockRawTxs := []common.Bytes{
		coinbaseTxBytes,
		sendTx1Bytes, sendTx2Bytes, sendTx3Bytes, sendTx4Bytes, sendTx5Bytes,
	}
	expectedStateRoot := common.HexToHash("0d7bff2377e3638b82b09c21b7d0636ed593d2225164cb9b67f7296432194c58")

	res := ledger.ApplyBlockTxs(blockRawTxs, expectedStateRoot)
	require.True(res.IsOK(), res.Message)

	//
	// Account balance sanity checks
	//

	// Validator balance
	validators := ledger.valMgr.GetValidatorSet(common.Hash{}).Validators()
	for _, val := range validators {
		valAddr := val.Address()
		valAcc := ledger.state.Delivered().GetAccount(valAddr)
		expectedValBal := types.NewCoins(100000000000, 1000)
		assert.NotNil(valAcc)
		assert.Equal(expectedValBal, valAcc.Balance)
	}

	// Output account balance
	accOutAfter := ledger.state.Delivered().GetAccount(accOut.Address)
	expectedAccOutBal := types.NewCoins(700075, 3)
	assert.Equal(expectedAccOutBal, accOutAfter.Balance)

	// Input account balance
	expectedAccInBal := types.Coins{
		ThetaWei: new(big.Int).SetInt64(899985),
		GammaWei: inAccInitGammaWei.Sub(inAccInitGammaWei, new(big.Int).SetInt64(txFee)),
	}
	for idx, _ := range accIns {
		accInAddr := accIns[idx].Account.Address
		accInAfter := ledger.state.Delivered().GetAccount(accInAddr)
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

	ctx := context.Background()
	messenger.Start(ctx)
	mempool.Start(ctx)

	initHeight := uint64(1)
	initRootHash := common.Hash{}
	ledger.ResetState(initHeight, initRootHash)

	return chainID, ledger, mempool
}

func newTesetValidatorManager(consensus core.ConsensusEngine) core.ValidatorManager {
	proposerAddressStr := consensus.PrivateKey().PublicKey().Address().String()
	propser := core.NewValidator(proposerAddressStr, new(big.Int).SetUint64(999))

	_, val2PubKey, err := crypto.TEST_GenerateKeyPairWithSeed("val2")
	if err != nil {
		panic(fmt.Sprintf("Failed to generate key pair with seed: %v", err))
	}
	val2 := core.NewValidator(val2PubKey.Address().String(), new(big.Int).SetUint64(100))

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
	txFee := getMinimumTxFee()
	validators := ledger.valMgr.GetValidatorSet(common.Hash{}).Validators()
	for _, val := range validators {
		valAccount := &types.Account{
			Address:                val.Address(),
			LastUpdatedBlockHeight: 1,
			Balance:                types.NewCoins(100000000000, 1000),
		}
		ledger.state.Delivered().SetAccount(val.Address(), valAccount)
	}

	accOut = types.MakeAccWithInitBalance("accOut", types.NewCoins(700000, 3))
	ledger.state.Delivered().SetAccount(accOut.Account.Address, &accOut.Account)

	for i := 0; i < numInAccs; i++ {
		secret := "in_secret_" + strconv.FormatInt(int64(i), 16)
		accIn := types.MakeAccWithInitBalance(secret, types.NewCoins(900000, 50000*txFee))
		accIns = append(accIns, accIn)
		ledger.state.Delivered().SetAccount(accIn.Account.Address, &accIn.Account)
	}

	ledger.state.Commit()

	return accOut, accIns
}

func newRawCoinbaseTx(chainID string, ledger *Ledger, sequence int) common.Bytes {
	vaList := ledger.valMgr.GetValidatorSet(common.Hash{}).Validators()
	if len(vaList) < 2 {
		panic("Insufficient number of validators")
	}
	outputs := []types.TxOutput{}
	for _, val := range vaList {
		output := types.TxOutput{val.Address(), types.NewCoins(0, 0)}
		outputs = append(outputs, output)
	}

	proposerSk := ledger.consensus.PrivateKey()
	proposerPk := proposerSk.PublicKey()
	coinbaseTx := &types.CoinbaseTx{
		Proposer:    types.TxInput{Address: proposerPk.Address(), Sequence: uint64(sequence)},
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

	coinbaseTxBytes, err := types.TxToBytes(coinbaseTx)
	if err != nil {
		panic(err)
	}
	return coinbaseTxBytes
}

func newRawSendTx(chainID string, sequence int, addPubKey bool, accOut, accIn types.PrivAccount, injectFeeFluctuation bool) common.Bytes {
	delta := int64(0)
	var err error
	if injectFeeFluctuation {
		// inject so fluctuation into the txFee, so later we can test whether the
		// mempool orders the txs by txFee
		randint, err := strconv.ParseInt(string(accIn.Address.Hex()[2:9]), 16, 64)
		if randint < 0 {
			randint = -randint
		}
		delta = randint * int64(types.GasSendTxPerAccount*2)
		if err != nil {
			panic(err)
		}
	}
	txFee := getMinimumTxFee() + delta
	sendTx := &types.SendTx{
		Fee: types.NewCoins(0, txFee),
		Inputs: []types.TxInput{
			{
				Sequence: uint64(sequence),
				Address:  accIn.Address,
				Coins:    types.NewCoins(15, txFee),
			},
		},
		Outputs: []types.TxOutput{
			{
				Address: accOut.Address,
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
	}

	sendTxBytes, err := types.TxToBytes(sendTx)
	if err != nil {
		panic(err)
	}
	return sendTxBytes
}

func getMinimumTxFee() int64 {
	return int64(types.MinimumTransactionFeeGammaWei)
}
