package ledger

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database/backend"
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
	inAccInitTFuelWei := accIns[0].Balance.TFuelWei
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
		valAddr := val.Address
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
		TFuelWei: inAccInitTFuelWei.Sub(inAccInitTFuelWei, new(big.Int).SetInt64(txFee)),
	}
	for idx, _ := range accIns {
		accInAddr := accIns[idx].Account.Address
		accInAfter := ledger.state.Delivered().GetAccount(accInAddr)
		assert.Equal(expectedAccInBal, accInAfter.Balance)
	}
}

// Test case for validator stake deposit, withdrawal, and return
func TestValidatorStakeUpdate(t *testing.T) {
	assert := assert.New(t)

	// ----------------- Stake Deposit ----------------- //

	chainID := "test_chain_001"
	db := backend.NewMemDatabase()

	snapshot, srcPrivAccs, valPrivAccs := genSimSnapshot(chainID, db)
	assert.Equal(6, len(srcPrivAccs))
	assert.Equal(6, len(valPrivAccs))

	es := newExecSim(chainID, db, snapshot, valPrivAccs[0])
	b0 := es.getTipBlock()

	// Add block #1 with a DepositStakeTx transaction
	b1 := core.NewBlock()
	b1.ChainID = chainID
	b1.Height = b0.Height + 1
	b1.Epoch = 1
	b1.Parent = b0.Hash()
	b1.HCC.BlockHash = b1.Parent

	txFee := getMinimumTxFee()
	depositSourcePrivAcc := srcPrivAccs[4]
	depoistHolderPrivAcc := valPrivAccs[4]
	depositStakeTx := &types.DepositStakeTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address: depositSourcePrivAcc.Address,
			Coins: types.Coins{
				ThetaWei: new(big.Int).Mul(new(big.Int).SetUint64(10), core.MinValidatorStakeDeposit),
				TFuelWei: new(big.Int).SetUint64(0),
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
	es.addBlock(b1)

	// Add more blocks
	b2 := core.NewBlock()
	b2.ChainID = chainID
	b2.Height = b1.Height + 1
	b2.Epoch = 2
	b2.Parent = b1.Hash()
	b2.StateHash = es.state.Commit()
	b2.HCC.BlockHash = b2.Parent
	es.addBlock(b2)

	b3 := core.NewBlock()
	b3.ChainID = chainID
	b3.Height = b2.Height + 1
	b3.Epoch = 3
	b3.Parent = b2.Hash()
	b3.HCC.BlockHash = b3.Parent
	b3.StateHash = es.state.Commit()
	es.addBlock(b3)

	// ----------------- Stake Withdrawal ----------------- //

	withdrawSourcePrivAcc := srcPrivAccs[0]
	withdrawHolderPrivAcc := valPrivAccs[0]

	srcAcc := es.state.Delivered().GetAccount(withdrawSourcePrivAcc.Address)
	balance0 := srcAcc.Balance
	log.Infof("Source account balance before withdrawal : %v", balance0)

	// Add block #4 with a WithdrawStakeTx transaction
	b4 := core.NewBlock()
	b4.ChainID = chainID
	b4.Height = b3.Height + 1
	b4.Epoch = 4
	b4.Parent = b3.Hash()
	b4.HCC.BlockHash = b4.Parent

	widthrawStakeTx := &types.WithdrawStakeTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address:  withdrawSourcePrivAcc.Address,
			Sequence: 1,
		},
		Holder: types.TxOutput{
			Address: withdrawHolderPrivAcc.Address,
		},
		Purpose: core.StakeForValidator,
	}
	signBytes = widthrawStakeTx.SignBytes(es.chainID)
	widthrawStakeTx.Source.Signature = withdrawSourcePrivAcc.Sign(signBytes)

	_, res = es.executor.ExecuteTx(widthrawStakeTx)
	assert.True(res.IsOK(), res.Message)

	b4.StateHash = es.state.Commit()
	es.addBlock(b4)

	b5 := core.NewBlock()
	b5.ChainID = chainID
	b5.Height = b4.Height + 1
	b5.Epoch = 5
	b5.Parent = b4.Hash()
	b5.HCC.BlockHash = b5.Parent
	b5.StateHash = es.state.Commit()

	es.addBlock(b5)

	b6 := core.NewBlock()
	b6.ChainID = chainID
	b6.Height = b5.Height + 1
	b6.Epoch = 6
	b6.Parent = b5.Hash()
	b6.HCC.BlockHash = b6.Parent
	b6.StateHash = es.state.Commit()
	es.addBlock(b6)

	// -------------- Check the ValidatorSets -------------- //

	// valSet0 := es.consensus.GetValidatorManager().GetValidatorSet(b0.Hash())
	// log.Infof("valSet for block #0: %v", valSet0)
	// assert.Equal(4, len(valSet0.Validators()))

	// valSet1 := es.consensus.GetValidatorManager().GetValidatorSet(b1.Hash())
	// log.Infof("valSet for block #1: %v", valSet1)
	// assert.Equal(4, len(valSet1.Validators()))

	valSet2 := es.consensus.GetValidatorManager().GetValidatorSet(b2.Hash())
	log.Infof("valSet for block #2: %v", valSet2)
	assert.Equal(4, len(valSet2.Validators()))

	valSet3 := es.consensus.GetValidatorManager().GetValidatorSet(b3.Hash())
	log.Infof("valSet for block #3: %v", valSet3)
	assert.Equal(5, len(valSet3.Validators()))

	valSet4 := es.consensus.GetValidatorManager().GetValidatorSet(b4.Hash())
	log.Infof("valSet for block #4: %v", valSet4)
	assert.Equal(5, len(valSet4.Validators()))

	valSet5 := es.consensus.GetValidatorManager().GetValidatorSet(b5.Hash())
	log.Infof("valSet for block #5: %v", valSet5)
	assert.Equal(5, len(valSet5.Validators()))

	valSet6 := es.consensus.GetValidatorManager().GetValidatorSet(b6.Hash())
	log.Infof("valSet for block #6: %v", valSet6)
	assert.Equal(4, len(valSet6.Validators()))

	// ----------------- Stake Return ----------------- //

	srcAcc = es.state.Delivered().GetAccount(withdrawSourcePrivAcc.Address)
	balance1 := srcAcc.Balance
	log.Infof("Source account balance after withdrawal  : %v", balance1)
	assert.Equal(balance0, balance1.Plus(types.NewCoins(0, txFee)))

	heightDelta1 := core.ReturnLockingPeriod / 10
	for h := uint64(0); h < heightDelta1; h++ {
		es.state.Commit() // increment height
	}
	expectedStateHash, _, res := es.consensus.GetLedger().ProposeBlockTxs()
	res = es.consensus.GetLedger().ApplyBlockTxs([]common.Bytes{}, expectedStateHash)
	assert.True(res.IsOK())

	srcAcc = es.state.Delivered().GetAccount(withdrawSourcePrivAcc.Address)
	balance2 := srcAcc.Balance
	log.Infof("Source account balance after %v blocks : %v", heightDelta1, balance2)

	assert.Equal(balance1, balance2) // still in the locking period, should not return stake

	heightDelta2 := core.ReturnLockingPeriod
	for h := uint64(0); h < heightDelta2; h++ {
		es.state.Commit() // increment height
	}
	expectedStateHash, _, res = es.consensus.GetLedger().ProposeBlockTxs()
	res = es.consensus.GetLedger().ApplyBlockTxs([]common.Bytes{}, expectedStateHash)
	assert.True(res.IsOK())

	srcAcc = es.state.Delivered().GetAccount(withdrawSourcePrivAcc.Address)
	balance3 := srcAcc.Balance
	log.Infof("Source account balance after %v blocks: %v", heightDelta2, balance3)

	returnedCoins := balance3.Minus(balance2)
	assert.True(returnedCoins.ThetaWei.Cmp(new(big.Int).Mul(new(big.Int).SetUint64(5), core.MinValidatorStakeDeposit)) == 0)
	assert.True(returnedCoins.TFuelWei.Cmp(core.Zero) == 0)
	log.Infof("Returned coins: %v", returnedCoins)
}
