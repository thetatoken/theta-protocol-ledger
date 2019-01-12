package ledger

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/ledger/types"
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

// Test case for stake deposit, withdrawal, and return
func TestStakeUpdate(t *testing.T) {
	assert := assert.New(t)

	chainID := "test_chain_001"
	db := backend.NewMemDatabase()

	snapshot, srcPrivAccs, valPrivAccs := genSimSnapshot(chainID, db)
	assert.Equal(6, len(srcPrivAccs))
	assert.Equal(6, len(valPrivAccs))

	es := newExecSim(chainID, db, snapshot, valPrivAccs[0])
	b0 := es.getTipBlock()

	// Add block #1 with an DepositStakeTx
	b1 := core.NewBlock()
	b1.ChainID = chainID
	b1.Height = b0.Height + 1
	b1.Parent = b0.Hash()
	es.addBlock(b1)

	txFee := getMinimumTxFee()
	depositSourcePrivAcc := srcPrivAccs[4]
	depoistHolderPrivAcc := valPrivAccs[4]
	depositStakeTx := &types.DepositStakeTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address: depositSourcePrivAcc.Address,
			Coins: types.Coins{
				ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(10), core.MinValidatorStakeDeposit),
				GammaWei: new(big.Int).SetUint64(0),
			},
			Sequence: 1,
		},
		Holder: types.TxOutput{
			Address: depoistHolderPrivAcc.Address,
		},
		Purpose: core.StakeForValidator,
	}
	signBytes := depositStakeTx.SignBytes(es.chainID)
	depositStakeTx.Source.Signature = depositSourcePrivAcc.Sign(signBytes)

	_, res := es.executor.ExecuteTx(depositStakeTx)
	assert.True(res.IsOK(), res.Message)

	b1.StateHash = es.state.Commit()

	// Add more blocks
	b2 := core.NewBlock()
	b2.ChainID = chainID
	b2.Height = b1.Height + 1
	b2.Parent = b1.Hash()
	b2.StateHash = es.state.Commit()
	es.addBlock(b2)

	b3 := core.NewBlock()
	b3.ChainID = chainID
	b3.Height = b2.Height + 1
	b3.Parent = b2.Hash()
	b3.StateHash = es.state.Commit()
	es.addBlock(b3)

	b4 := core.NewBlock()
	b4.ChainID = chainID
	b4.Height = b3.Height + 1
	b4.Parent = b3.Hash()
	b4.StateHash = es.state.Commit()
	es.addBlock(b4)

	// Directly finalize block #3
	es.finalizePreviousBlocks(b3.Hash())

	valSet0 := es.consensus.GetValidatorManager().GetValidatorSet(b0.Hash())
	log.Infof("valSet0: %v", valSet0)
	assert.Equal(4, len(valSet0.Validators()))

	valSet1 := es.consensus.GetValidatorManager().GetValidatorSet(b1.Hash())
	log.Infof("valSet1: %v", valSet1)
	assert.Equal(4, len(valSet1.Validators()))

	valSet2 := es.consensus.GetValidatorManager().GetValidatorSet(b2.Hash())
	log.Infof("valSet2: %v", valSet2)
	assert.Equal(4, len(valSet2.Validators()))

	valSet3 := es.consensus.GetValidatorManager().GetValidatorSet(b3.Hash())
	log.Infof("valSet3: %v", valSet3)
	assert.Equal(5, len(valSet3.Validators()))

	valSet4 := es.consensus.GetValidatorManager().GetValidatorSet(b4.Hash())
	log.Infof("valSet4: %v", valSet4)
	assert.Equal(5, len(valSet4.Validators()))
}
