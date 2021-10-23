package execution

import (
	"fmt"
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/ledger/types"
)

func TestGetInputs(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	// nil submissions
	acc, res := getInputs(nil, nil)
	assert.True(res.IsOK(), "getInputs: error on nil submission")
	assert.Zero(len(acc), "getInputs: accounts returned on nil submission")

	// test getInputs for registered, non-registered account
	et.reset()
	inputs := types.Accs2TxInputs(1, et.accIn)
	acc, res = getInputs(et.state().Delivered(), inputs)
	assert.True(res.IsError(), "getInputs: expected error when using getInput with non-registered Input")

	et.acc2State(et.accIn)
	acc, res = getInputs(et.state().Delivered(), inputs)
	assert.True(res.IsOK(), "getInputs: expected to getInput from registered Input")

	// test sending duplicate accounts
	et.reset()
	et.acc2State(et.accIn, et.accIn, et.accIn)
	inputs = types.Accs2TxInputs(1, et.accIn, et.accIn, et.accIn)
	acc, res = getInputs(et.state().Delivered(), inputs)
	assert.True(res.IsError(), "getInputs: expected error when sending duplicate accounts")

	// test calculating reward
	et.reset()
	et.acc2State(et.accIn)

	et.fastforwardBy(1000) // fastforward to reach a sufficient height for TFuel generation

	inputs = types.Accs2TxInputs(1, et.accIn)
	acc, res = getInputs(et.state().Delivered(), inputs)
	assert.True(res.IsOK(), "getInputs: expected to get input from a few block heights ago")
	assert.True(acc[string(inputs[0].Address[:])].Balance.TFuelWei.Cmp(et.accIn.Balance.TFuelWei) == 0,
		"getInputs: tfuel amount should not change")
}

func TestGetOrMakeOutputs(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//nil submissions
	acc, res := getOrMakeOutputs(nil, nil, nil)
	assert.True(res.IsOK(), "getOrMakeOutputs: error on nil submission")
	assert.Zero(len(acc), "getOrMakeOutputs: accounts returned on nil submission")

	//test sending duplicate accounts
	et.reset()
	outputs := types.Accs2TxOutputs(et.accIn, et.accIn, et.accIn)
	_, res = getOrMakeOutputs(et.state().Delivered(), nil, outputs)
	assert.True(res.IsError(), "getOrMakeOutputs: expected error when sending duplicate accounts")

	//test sending to existing/new account
	et.reset()
	outputs1 := types.Accs2TxOutputs(et.accIn)
	outputs2 := types.Accs2TxOutputs(et.accOut)

	et.acc2State(et.accIn)
	_, res = getOrMakeOutputs(et.state().Delivered(), nil, outputs1)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to existing account")

	mapRes2, res := getOrMakeOutputs(et.state().Delivered(), nil, outputs2)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to new account")

	//test the map results
	_, map2ok := mapRes2[string(outputs2[0].Address[:])]
	assert.True(map2ok, "getOrMakeOutputs: account output does not contain new account map item")

	//test calculating reward
	et.reset()
	et.fastforwardBy(1000) // fastforward to reach a sufficient height for TFuel generation

	outputs1 = types.Accs2TxOutputs(et.accIn)
	outputs2 = types.Accs2TxOutputs(et.accOut)

	et.acc2State(et.accIn)
	mapRes1, res := getOrMakeOutputs(et.state().Delivered(), nil, outputs1)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to existing account")
	assert.True(mapRes1[string(outputs1[0].Address[:])].Balance.TFuelWei.Cmp(et.accIn.Balance.TFuelWei) == 0,
		"getOrMakeOutputs: tfuel amount should not change")

	mapRes2, res = getOrMakeOutputs(et.state().Delivered(), nil, outputs2)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to new account")
	assert.True(mapRes2[string(outputs2[0].Address[:])].Balance.TFuelWei.Cmp(types.Zero) == 0,
		"getOrMakeOutputs: expected to not update new output account tfuel balance")
}

func TestValidateInputsBasic(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//validate input basic
	inputs := types.Accs2TxInputs(1, et.accIn)
	res := validateInputsBasic(inputs)
	assert.True(res.IsOK(), "validateInputsBasic: expected no error on good tx input. Error: %v", res.Message)

	t.Log("inputs[0].Coins = ", inputs[0].Coins)
	inputs[0].Coins.ThetaWei = big.NewInt(-1)
	res = validateInputsBasic(inputs)
	assert.True(res.IsError(), "validateInputsBasic: expected error on bad tx input")
}

func TestValidateInputsAdvanced(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//create three temp accounts for the test
	accIn1 := types.MakeAcc("foox")
	accIn2 := types.MakeAcc("fooy")
	accIn3 := types.MakeAcc("fooz")

	//validate inputs advanced
	tx := types.MakeSendTx(1, et.accOut, accIn1, accIn2, accIn3)

	et.acc2State(accIn1, accIn2, accIn3, et.accOut)
	accMap, res := getInputs(et.state().Delivered(), tx.Inputs)
	assert.True(res.IsOK(), "validateInputsAdvanced: error retrieving accMap. Error: %v", res.Message)
	signBytes := tx.SignBytes(et.chainID)

	//test bad case, unsigned
	totalCoins, res := validateInputsAdvanced(accMap, signBytes, tx.Inputs, 1)
	assert.True(res.IsError(), "validateInputsAdvanced: expected an error on an unsigned tx input")

	//test good case sgined
	et.signSendTx(tx, accIn1, accIn2, accIn3, et.accOut)
	totalCoins, res = validateInputsAdvanced(accMap, signBytes, tx.Inputs, 1)
	assert.True(res.IsOK(), "validateInputsAdvanced: expected no error on good tx input. Error: %v", res.Message)

	txTotalCoins := tx.Inputs[0].Coins.
		Plus(tx.Inputs[1].Coins).
		Plus(tx.Inputs[2].Coins)

	assert.True(totalCoins.IsEqual(txTotalCoins),
		"ValidateInputsAdvanced: transaction total coins are not equal: got %v, expected %v", txTotalCoins, totalCoins)
}

func TestValidateInputAdvanced(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//validate input advanced
	tx := types.MakeSendTx(1, et.accOut, et.accIn)

	et.acc2State(et.accIn, et.accOut)
	signBytes := tx.SignBytes(et.chainID)

	//unsigned case
	res := validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0], 1)
	assert.True(res.IsError(), "validateInputAdvanced: expected error on tx input without signature")

	//good signed case
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0], 1)
	assert.True(res.IsOK(), "validateInputAdvanced: expected no error on good tx input. Error: %v", res.Message)

	//bad sequence case
	et.accIn.Sequence = 1
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0], 1)
	assert.Equal(result.CodeInvalidSequence, res.Code, "validateInputAdvanced: expected error on tx input with bad sequence")
	et.accIn.Sequence = 0 //restore sequence

	//bad balance case
	et.accIn.Balance = types.NewCoins(2, 0)
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0], 1)
	assert.Equal(result.CodeInsufficientFund, res.Code,
		"validateInputAdvanced: expected error on tx input with insufficient funds %v", et.accIn.Sequence)
}

func TestValidateOutputsBasic(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//validateOutputsBasic
	tx := types.Accs2TxOutputs(et.accIn)
	res := validateOutputsBasic(tx)
	assert.True(res.IsOK(), "validateOutputsBasic: expected no error on good tx output. Error: %v", res.Message)

	tx[0].Coins.ThetaWei = big.NewInt(-1)
	res = validateOutputsBasic(tx)
	assert.True(res.IsError(), "validateInputBasic: expected error on bad tx output. Error: %v", res.Message)
}

func TestSumOutput(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//SumOutput
	tx := types.Accs2TxOutputs(et.accIn, et.accOut)
	total := sumOutputs(tx)
	assert.True(total.IsEqual(tx[0].Coins.Plus(tx[1].Coins)), "sumOutputs: total coins are not equal")
}

func TestAdjustBy(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//adjustByInputs/adjustByOutputs
	//sending transaction from accIn to accOut
	initBalIn := et.accIn.Account.Balance
	initBalOut := et.accOut.Account.Balance
	et.acc2State(et.accIn, et.accOut)

	txIn := types.Accs2TxInputs(1, et.accIn)
	txOut := types.Accs2TxOutputs(et.accOut)
	accMap, _ := getInputs(et.state().Delivered(), txIn)
	accMap, _ = getOrMakeOutputs(et.state().Delivered(), accMap, txOut)

	adjustByInputs(et.state().Delivered(), accMap, txIn)
	adjustByOutputs(et.state().Delivered(), accMap, txOut)

	inAddr := et.accIn.Account.Address
	outAddr := et.accOut.Account.Address
	endBalIn := accMap[string(inAddr[:])].Balance
	endBalOut := accMap[string(outAddr[:])].Balance
	decrBalIn := initBalIn.Minus(endBalIn)
	incrBalOut := endBalOut.Minus(initBalOut)

	assert.True(decrBalIn.IsEqual(txIn[0].Coins),
		"adjustByInputs: total coins are not equal. diff: %v, tx: %v", decrBalIn.String(), txIn[0].Coins.String())
	assert.True(incrBalOut.IsEqual(txOut[0].Coins),
		"adjustByInputs: total coins are not equal. diff: %v, tx: %v", incrBalOut.String(), txOut[0].Coins.String())
}

func TestSendTx(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	//ExecTx
	tx := types.MakeSendTx(1, et.accOut, et.accIn)
	et.acc2State(et.accIn)
	et.acc2State(et.accOut)
	et.signSendTx(tx, et.accIn)

	//Bad Balance
	et.accIn.Balance = types.NewCoins(2, 0)
	et.acc2State(et.accIn)
	res, _, _, _, _ := et.execSendTx(tx, true)
	assert.True(res.IsError(), "ExecTx/Bad CheckTx: Expected error return from ExecTx, returned: %v", res)

	res, balIn, balInExp, balOut, balOutExp := et.execSendTx(tx, false)
	assert.True(res.IsError(), "ExecTx/Bad DeliverTx: Expected error return from ExecTx, returned: %v", res)
	assert.False(balIn.IsEqual(balInExp),
		"ExecTx/Bad DeliverTx: balance shouldn't be equal for accIn: got %v, expected: %v", balIn, balInExp)
	assert.False(balOut.IsEqual(balOutExp),
		"ExecTx/Bad DeliverTx: balance shouldn't be equal for accOut: got %v, expected: %v", balOut, balOutExp)

	//Regular CheckTx
	et.reset()
	et.acc2State(et.accIn)
	et.acc2State(et.accOut)
	res, _, _, _, _ = et.execSendTx(tx, true)
	assert.True(res.IsOK(), "ExecTx/Good CheckTx: Expected OK return from ExecTx, Error: %v", res)

	//Regular DeliverTx
	et.reset()
	et.acc2State(et.accIn)
	et.acc2State(et.accOut)
	res, balIn, balInExp, balOut, balOutExp = et.execSendTx(tx, false)
	assert.True(res.IsOK(), "ExecTx/Good DeliverTx: Expected OK return from ExecTx, Error: %v", res)
	assert.True(balIn.IsEqual(balInExp),
		"ExecTx/good DeliverTx: unexpected change in input balance, got: %v, expected: %v", balIn, balInExp)
	assert.True(balOut.IsEqual(balOutExp),
		"ExecTx/good DeliverTx: unexpected change in output balance, got: %v, expected: %v", balOut, balOutExp)
}

func TestSendDuplicatedInputOutput(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	et.acc2State(et.accIn)
	et.acc2State(et.accOut)

	fee := types.NewCoins(0, getMinimumTxFee())
	c1 := types.NewCoins(20000, 0)
	c2 := types.NewCoins(50000, 3000)
	sendTx := &types.SendTx{
		Fee: fee,
		Inputs: []types.TxInput{
			types.TxInput{
				Address:  et.accIn.Address,
				Coins:    c1.Plus(fee),
				Sequence: et.accIn.Sequence + 1,
			},
			types.TxInput{
				Address:  et.accOut.Address,
				Coins:    c2,
				Sequence: et.accOut.Sequence + 1,
			},
		},
		Outputs: []types.TxOutput{
			types.TxOutput{
				Address: et.accIn.Address,
				Coins:   c1,
			},
			types.TxOutput{
				Address: et.accOut.Address,
				Coins:   c2,
			},
		},
	}

	// Sign transaction
	signBytes := sendTx.SignBytes(et.chainID)
	sendTx.Inputs[0].Signature = et.accIn.Sign(signBytes)
	sendTx.Inputs[1].Signature = et.accOut.Sign(signBytes)

	accInBal0 := et.accIn.Balance
	accOutBal0 := et.accOut.Balance
	t.Logf("----- Before executing SendTx -----\n")
	t.Logf("accIn.Balance  = %v\n", accInBal0)
	t.Logf("accOut.Balance = %v\n", accOutBal0)

	res, _, _, _, _ := et.execSendTx(sendTx, true)
	assert.False(res.IsOK(), "ExecTx/Good CheckTx: Expected OK return from ExecTx, Error: %v", res)
	et.executor.state.Commit()

	accInBal1 := et.executor.state.Delivered().GetAccount(et.accIn.Address).Balance
	accOutBal1 := et.executor.state.Delivered().GetAccount(et.accOut.Address).Balance
	t.Logf("----- After executing SendTx -----\n")
	t.Logf("accIn.Balance  = %v\n", accInBal1)
	t.Logf("accOut.Balance = %v\n", accOutBal1)

	assert.Equal(accInBal0, accInBal1)
	assert.Equal(accOutBal0, accOutBal1)
}

// func TestCalculateThetaReward(t *testing.T) {
// 	assert := assert.New(t)

// 	res := calculateThetaReward(big.NewInt(1e17), true)
// 	assert.True(res.ThetaWei.Cmp(types.Zero) == 0) // ZERO Theta inflation
// }

// func TestCoinbaseTx(t *testing.T) {
// 	assert := assert.New(t)
// 	et := NewExecTest()

// 	va1 := et.accProposer
// 	va1.Balance = types.Coins{ThetaWei: big.NewInt(1e11), TFuelWei: big.NewInt(0)}
// 	et.acc2State(va1)

// 	va2 := et.accVal2
// 	va2.Balance = types.Coins{ThetaWei: big.NewInt(3e11), TFuelWei: big.NewInt(0)}
// 	et.acc2State(va2)

// 	user1 := types.MakeAcc("user 1")
// 	user1.Balance = types.Coins{ThetaWei: big.NewInt(1e11), TFuelWei: big.NewInt(0)}
// 	et.acc2State(user1)

// 	et.fastforwardTo(1e7)

// 	var tx *types.CoinbaseTx
// 	var res result.Result

// 	// Regular check
// 	tx = &types.CoinbaseTx{
// 		Proposer: types.TxInput{
// 			Address: va1.PrivKey.PublicKey().Address()},
// 		Outputs: []types.TxOutput{{
// 			va1.Account.Address, types.NewCoins(0, 0),
// 		}, {
// 			va2.Account.Address, types.NewCoins(0, 0),
// 		}},
// 		BlockHeight: 1e7,
// 	}
// 	tx.Proposer.Signature = va1.Sign(tx.SignBytes(et.chainID))

// 	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	assert.True(res.IsOK(), res.String())

// 	// Theta should never inflate
// 	tx = &types.CoinbaseTx{
// 		Proposer: types.TxInput{
// 			Address: va1.Address},
// 		Outputs: []types.TxOutput{{
// 			va1.Account.Address, types.NewCoins(317, 0),
// 		}, {
// 			va2.Account.Address, types.NewCoins(317, 0),
// 		}},
// 		BlockHeight: 1e7,
// 	}
// 	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	assert.True(res.IsError(), res.String())

// 	// For the initial Mainnet release, TFuel should not inflate
// 	tx = &types.CoinbaseTx{
// 		Proposer: types.TxInput{
// 			Address: va1.Address},
// 		Outputs: []types.TxOutput{{
// 			va1.Account.Address, types.NewCoins(0, 987),
// 		}, {
// 			va2.Account.Address, types.NewCoins(0, 0),
// 		}},
// 		BlockHeight: 1e7,
// 	}
// 	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	assert.True(res.IsError(), res.String())

// 	// //Error if reward Theta amount is incorrect
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}, {
// 	// 		va2.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }
// 	// res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsError(), res.String())

// 	// //Error if reward TFuel amount is incorrect
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}, {
// 	// 		va2.Account.PubKey.Address(), types.NewCoins(951, 1),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }
// 	// res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsError(), res.String())

// 	// //Error if Validator 2 is not rewarded
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }
// 	// res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsError(), res.String())

// 	// //Error if non-validator is rewarded
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}, {
// 	// 		va2.Account.PubKey.Address(), types.NewCoins(951, 0),
// 	// 	}, {
// 	// 		user1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }
// 	// res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsError(), res.String())

// 	// //Error if validator address is changed
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}, {
// 	// 		user1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }
// 	// res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsError(), res.String())

// 	// //Process should update validator account
// 	// tx = &types.CoinbaseTx{
// 	// 	Proposer: types.TxInput{
// 	// 		Address: va1.PubKey.Address(), PubKey: va1.PubKey},
// 	// 	Outputs: []types.TxOutput{{
// 	// 		va1.Account.PubKey.Address(), types.NewCoins(317, 0),
// 	// 	}, {
// 	// 		va2.Account.PubKey.Address(), types.NewCoins(951, 0),
// 	// 	}},
// 	// 	BlockHeight: 1e7,
// 	// }

// 	// _, res = et.executor.getTxExecutor(tx).process(et.chainID, et.state().Delivered(), tx)
// 	// assert.True(res.IsOK(), res.String())

// 	// va1balance := et.state().Delivered().GetAccount(va1.Account.PubKey.Address()).Balance
// 	// assert.Equal(int64(100000000317), va1balance.ThetaWei.Int64())
// 	// // validator's TFuel is also updated.
// 	// assert.Equal(int64(189999981000), va1balance.TFuelWei.Int64())

// 	// va2balance := et.state().Delivered().GetAccount(va2.Account.PubKey.Address()).Balance
// 	// assert.Equal(int64(300000000951), va2balance.ThetaWei.Int64())
// 	// assert.Equal(int64(569999943000), va2balance.TFuelWei.Int64())

// 	// user1balance := et.state().Delivered().GetAccount(user1.Account.PubKey.Address()).Balance
// 	// assert.Equal(int64(100000000000), user1balance.ThetaWei.Int64())
// 	// // user's TFuel is not updated.
// 	// assert.Equal(int64(0), user1balance.TFuelWei.Int64())
// }

func TestReserveFundTx(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	txFee := getMinimumTxFee()

	user1 := types.MakeAcc("user 1")
	user1.Balance = types.Coins{
		TFuelWei: big.NewInt(6200 * txFee),
		ThetaWei: big.NewInt(10000 * 1e6),
	}
	et.acc2State(user1)

	et.fastforwardTo(1e7)

	var tx *types.ReserveFundTx
	var res result.Result

	// Reserved fund not specified
	tx = &types.ReserveFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  user1.PrivKey.PublicKey().Address(),
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{"rid001"},
		Duration:    1000,
	}
	tx.Source.Signature = user1.Sign(tx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeReservedFundNotSpecified)

	// Insufficient fund
	tx = &types.ReserveFundTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address:  user1.PrivKey.PublicKey().Address(),
			Coins:    types.Coins{TFuelWei: big.NewInt(50000 * txFee), ThetaWei: big.NewInt(0)},
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(50001 * txFee), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{"rid001"},
		Duration:    1000,
	}
	tx.Source.Signature = user1.Sign(tx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeInsufficientFund)

	// Reserved fund more than collateral
	tx = &types.ReserveFundTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address:  user1.Address,
			Coins:    types.Coins{TFuelWei: big.NewInt(5000 * txFee), ThetaWei: big.NewInt(0)},
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{"rid001"},
		Duration:    1000,
	}
	tx.Source.Signature = user1.Sign(tx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeReserveFundCheckFailed, res.Message)

	// Regular check
	tx = &types.ReserveFundTx{
		Fee: types.NewCoins(0, txFee),
		Source: types.TxInput{
			Address:  user1.Address,
			Coins:    types.Coins{TFuelWei: big.NewInt(1000 * txFee), ThetaWei: big.NewInt(0)},
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{"rid001"},
		Duration:    1000,
	}
	tx.Source.Signature = user1.Sign(tx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(tx).sanityCheck(et.chainID, et.state().Delivered(), tx)
	assert.True(res.IsOK(), res.String())
	_, res = et.executor.getTxExecutor(tx).process(et.chainID, et.state().Delivered(), tx)
	assert.True(res.IsOK(), res.String())

	retrievedUserAcc := et.state().Delivered().GetAccount(user1.Address)
	assert.Equal(1, len(retrievedUserAcc.ReservedFunds))
	assert.Equal([]string{"rid001"}, retrievedUserAcc.ReservedFunds[0].ResourceIDs)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)}, retrievedUserAcc.ReservedFunds[0].Collateral)
	assert.Equal(uint64(1), retrievedUserAcc.ReservedFunds[0].ReserveSequence)
}

func TestReleaseFundTx(t *testing.T) {
	assert := assert.New(t)
	et := NewExecTest()

	user1 := types.MakeAcc("user 1")
	user1.Balance = types.Coins{
		TFuelWei: big.NewInt(50 * getMinimumTxFee()),
		ThetaWei: big.NewInt(10000 * 1e6),
	}
	et.acc2State(user1)

	et.fastforwardTo(1e7)

	var reserveFundTx *types.ReserveFundTx
	var releaseFundTx *types.ReleaseFundTx
	var res result.Result

	reserveFundTx = &types.ReserveFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  user1.Address,
			Coins:    types.Coins{TFuelWei: big.NewInt(1000 * 1e6), ThetaWei: big.NewInt(0)},
			Sequence: 1,
		},
		Collateral:  types.Coins{TFuelWei: big.NewInt(1001 * 1e6), ThetaWei: big.NewInt(0)},
		ResourceIDs: []string{"rid001"},
		Duration:    1000,
	}
	reserveFundTx.Source.Signature = user1.Sign(reserveFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(reserveFundTx).sanityCheck(et.chainID, et.state().Delivered(), reserveFundTx)
	assert.True(res.IsOK(), res.String())
	_, res = et.executor.getTxExecutor(reserveFundTx).process(et.chainID, et.state().Delivered(), reserveFundTx)
	assert.True(res.IsOK(), res.String())

	et.state().Commit()

	// Invalid Fee
	releaseFundTx = &types.ReleaseFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()-1), // insufficient transaction fee
		Source: types.TxInput{
			Address:  user1.Address,
			Sequence: 2,
		},
		ReserveSequence: 1,
	}
	releaseFundTx.Source.Signature = user1.Sign(releaseFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(releaseFundTx).sanityCheck(et.chainID, et.state().Delivered(), releaseFundTx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeInvalidFee, res.String())

	releaseFundTx = &types.ReleaseFundTx{
		Fee: types.NewCoins(100, getMinimumTxFee()), // Theta cannot be used as transaction fee
		Source: types.TxInput{
			Address:  user1.Address,
			Sequence: 2,
		},
		ReserveSequence: 1,
	}
	releaseFundTx.Source.Signature = user1.Sign(releaseFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(releaseFundTx).sanityCheck(et.chainID, et.state().Delivered(), releaseFundTx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeInvalidFee, res.String())

	// Not expire yet
	releaseFundTx = &types.ReleaseFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  user1.Address,
			Sequence: 2,
		},
		ReserveSequence: 1,
	}
	releaseFundTx.Source.Signature = user1.Sign(releaseFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(releaseFundTx).sanityCheck(et.chainID, et.state().Delivered(), releaseFundTx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeReleaseFundCheckFailed, res.String())

	et.fastforwardTo(1e7 + 9999)

	// No matching ReserveSequence
	releaseFundTx = &types.ReleaseFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  user1.Address,
			Sequence: 2,
		},
		ReserveSequence: 99,
	}
	releaseFundTx.Source.Signature = user1.Sign(releaseFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(releaseFundTx).sanityCheck(et.chainID, et.state().Delivered(), releaseFundTx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeReleaseFundCheckFailed, res.String())

	// NOTE: The following check should FAIL, since the expired ReservedFunds are now
	//       released by the Account.UpdateToHeight() function. Once the height
	//       reaches the target release height, the ReservedFunds will be released
	//       automatically. No need to explicitly execute ReleaseFundTx

	// Check auto-expiration
	releaseFundTx = &types.ReleaseFundTx{
		Fee: types.NewCoins(0, getMinimumTxFee()),
		Source: types.TxInput{
			Address:  user1.Address,
			Sequence: 2,
		},
		ReserveSequence: 1,
	}
	releaseFundTx.Source.Signature = user1.Sign(releaseFundTx.SignBytes(et.chainID))
	res = et.executor.getTxExecutor(releaseFundTx).sanityCheck(et.chainID, et.state().Delivered(), releaseFundTx)
	assert.False(res.IsOK(), res.String())
	assert.Equal(res.Code, result.CodeReleaseFundCheckFailed, res.String())
}

func TestServicePaymentTxNormalExecutionAndSlash(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	et.state().Commit()

	txFee := getMinimumTxFee()

	retrievedAliceAcc0 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(1, len(retrievedAliceAcc0.ReservedFunds))
	assert.Equal([]string{resourceID}, retrievedAliceAcc0.ReservedFunds[0].ResourceIDs)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)}, retrievedAliceAcc0.ReservedFunds[0].Collateral)
	assert.Equal(uint64(1), retrievedAliceAcc0.ReservedFunds[0].ReserveSequence)

	// Simulate micropayment #1 between Alice and Bob
	payAmount1 := int64(80 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 10*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 50*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx1 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount1, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res := et.executor.getTxExecutor(servicePaymentTx1).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx1)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx1).process(et.chainID, et.state().Delivered(), servicePaymentTx1)
	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc1 := et.state().Delivered().GetAccount(alice.Address)

	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1), ThetaWei: big.NewInt(0)}, retrievedAliceAcc1.ReservedFunds[0].UsedFund)
	retrievedBobAcc1 := et.state().Delivered().GetAccount(bob.Address)
	assert.Equal(bobInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount1 - txFee), ThetaWei: big.NewInt(0)}), retrievedBobAcc1.Balance) // payAmount1 - txFee: need to account for tx fee

	// Simulate micropayment #2 between Alice and Bob
	payAmount2 := int64(50 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 2, 2, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 30*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx2 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount2, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx2).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx2)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx2).process(et.chainID, et.state().Delivered(), servicePaymentTx2)
	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc2 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2), ThetaWei: big.NewInt(0)}, retrievedAliceAcc2.ReservedFunds[0].UsedFund)
	retrievedBobAcc2 := et.state().Delivered().GetAccount(bob.Address)
	assert.Equal(bobInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2 - 2*txFee)}), retrievedBobAcc2.Balance) // payAmount1 + payAmount2 - 2*txFee: need to account for tx fee

	// Simulate micropayment #3 between Alice and Carol
	payAmount3 := int64(120 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 1, 3, 1
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 30*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx3 := createServicePaymentTx(et.chainID, &alice, &carol, payAmount3, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx3).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx3)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx3).process(et.chainID, et.state().Delivered(), servicePaymentTx3)
	assert.True(res.IsOK(), res.Message)
	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))

	et.state().Commit()

	retrievedAliceAcc3 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1 + payAmount2 + payAmount3), ThetaWei: big.NewInt(0)}, retrievedAliceAcc3.ReservedFunds[0].UsedFund)
	retrievedCarolAcc3 := et.state().Delivered().GetAccount(carol.Address)
	assert.Equal(carolInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount3 - txFee)}), retrievedCarolAcc3.Balance) // payAmount3 - txFee: need to account for tx fee

	// Simulate micropayment #4 between Alice and Carol. This is an overspend, alice should get slashed.
	payAmount4 := int64(2000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 2, 4, 1
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 70000*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx4 := createServicePaymentTx(et.chainID, &alice, &carol, payAmount4, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx4).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx4)
	assert.True(res.IsOK(), res.Message) // the following process() call will create an SlashIntent

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx4).process(et.chainID, et.state().Delivered(), servicePaymentTx4)
	assert.True(res.IsOK(), res.Message)
	//assert.Equal(1, len(et.state().Delivered().GetSlashIntents()))
}

func TestServicePaymentTxExpiration(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, _, _, bobInitBalance, _ := setupForServicePayment(assert)
	et.state().Commit()

	txFee := getMinimumTxFee()

	retrievedAliceAcc1 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(1, len(retrievedAliceAcc1.ReservedFunds))
	assert.Equal([]string{resourceID}, retrievedAliceAcc1.ReservedFunds[0].ResourceIDs)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(1001 * txFee), ThetaWei: big.NewInt(0)}, retrievedAliceAcc1.ReservedFunds[0].Collateral)
	assert.Equal(uint64(1), retrievedAliceAcc1.ReservedFunds[0].ReserveSequence)

	// Simulate micropayment #1 between Alice and Bobs
	payAmount1 := int64(80 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 10*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 50*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx1 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount1, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res := et.executor.getTxExecutor(servicePaymentTx1).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx1)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(servicePaymentTx1).process(et.chainID, et.state().Delivered(), servicePaymentTx1)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	retrievedAliceAcc2 := et.state().Delivered().GetAccount(alice.Address)
	assert.Equal(types.Coins{TFuelWei: big.NewInt(payAmount1), ThetaWei: big.NewInt(0)}, retrievedAliceAcc2.ReservedFunds[0].UsedFund)
	retrievedBobAcc2 := et.state().Delivered().GetAccount(bob.Address)
	assert.Equal(bobInitBalance.Plus(types.Coins{TFuelWei: big.NewInt(payAmount1 - txFee)}), retrievedBobAcc2.Balance) // payAmount1 - txFee: need to account for Gas

	et.fastforwardBy(1e4) // The reservedFund should expire after the fastforward

	// Simulate micropayment #2 between Alice and Bobs
	payAmount2 := int64(50 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq = 1, 2, 2, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 30*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx2 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount2, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx2).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx2)
	assert.False(res.IsOK(), res.Message)
	assert.Equal(result.CodeCheckTransferReservedFundFailed, res.Code)
	log.Infof("Service payment check message: %v", res.Message)
}

// func TestSlashTx(t *testing.T) {
// 	assert := assert.New(t)
// 	et, resourceID, alice, bob, _, _, _, _ := setupForServicePayment(assert)

// 	proposer := et.accProposer
// 	proposerInitBalance := proposer.Account.Balance
// 	et.acc2State(proposer)
// 	log.Infof("Proposer's Address: %v", proposer.Address.Hex())

// 	et.state().Commit()

// 	txFee := getMinimumTxFee()

// 	retrievedAliceAccount := et.state().Delivered().GetAccount(alice.Address)
// 	assert.Equal(1, len(retrievedAliceAccount.ReservedFunds))
// 	aliceCollateral := retrievedAliceAccount.ReservedFunds[0].Collateral
// 	aliceReservedFund := retrievedAliceAccount.ReservedFunds[0].InitialFund
// 	expectedAliceSlashedAmount := aliceCollateral.Plus(aliceReservedFund)

// 	// Simulate micropayment #1 between Alice and Bob, which is an overspend
// 	payAmount1 := int64(8000 * txFee)
// 	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
// 	_ = createServicePaymentTx(et.chainID, &alice, &bob, 10*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
// 	_ = createServicePaymentTx(et.chainID, &alice, &bob, 50*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
// 	servicePaymentTx1 := createServicePaymentTx(et.chainID, &alice, &bob, payAmount1, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
// 	res := et.executor.getTxExecutor(servicePaymentTx1).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx1)
// 	assert.True(res.IsOK(), res.Message)

// 	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
// 	_, res = et.executor.getTxExecutor(servicePaymentTx1).process(et.chainID, et.state().Delivered(), servicePaymentTx1)
// 	assert.True(res.IsOK(), res.Message)
// 	assert.Equal(1, len(et.state().Delivered().GetSlashIntents()))

// 	slashIntent := et.state().Delivered().GetSlashIntents()[0]

// 	et.state().Commit()

// 	// Test the slashTx
// 	slashTx := &types.SlashTx{
// 		Proposer: types.TxInput{
// 			Address:  proposer.Address,
// 			Sequence: 1,
// 		},
// 		SlashedAddress:  slashIntent.Address,
// 		ReserveSequence: slashIntent.ReserveSequence,
// 		SlashProof:      slashIntent.Proof,
// 	}
// 	signBytes := slashTx.SignBytes(et.chainID)
// 	slashTx.Proposer.Signature = proposer.Sign(signBytes)

// 	res = et.executor.getTxExecutor(slashTx).sanityCheck(et.chainID, et.state().Delivered(), slashTx)
// 	assert.True(res.IsOK(), res.Message)
// 	_, res = et.executor.getTxExecutor(slashTx).process(et.chainID, et.state().Delivered(), slashTx)
// 	assert.True(res.IsOK(), res.Message)

// 	retrievedProposerAccount := et.state().Delivered().GetAccount(proposer.Address)
// 	assert.Equal(proposerInitBalance.Plus(expectedAliceSlashedAmount), retrievedProposerAccount.Balance) // slashed tokens transferred to the proposer

// 	retrievedAliceAccountAfterSlash := et.state().Delivered().GetAccount(alice.Address)
// 	assert.Equal(0, len(retrievedAliceAccountAfterSlash.ReservedFunds)) // Alice is slashed

// 	log.Infof("Proposer initial balance: %v", proposerInitBalance)
// 	log.Infof("Alice collateral: %v", aliceCollateral)
// 	log.Infof("Alice reserved fund: %v", aliceReservedFund)
// 	log.Infof("Proposer final balance: %v", retrievedProposerAccount.Balance)
// }

func TestSplitRuleTxNormalExecution(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 30,
	}
	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitCarol},
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &bob, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 70 / 100), ThetaWei: big.NewInt(0)}
	servicePaymentTxFee := types.NewCoins(0, txFee)
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 30 / 100), ThetaWei: big.NewInt(0)}
	assert.Equal(bobInitBalance.Plus(bobSplitCoins).Minus(servicePaymentTxFee), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins), carolFinalBalance)
}

func TestSplitRuleTxExpiration(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 30,
	}
	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitCarol},
		Duration: uint64(100),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	et.fastforwardBy(105) // The split rule should expire after the fastforward

	// Simulate micropayment #1 between Alice and Bob, Carol should NOT get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 100, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &bob, 500, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &bob, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount), ThetaWei: big.NewInt(0)}
	servicePaymentTxFee := types.NewCoins(0, txFee)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins).Minus(servicePaymentTxFee), bobFinalBalance)
	assert.Equal(carolInitBalance, carolFinalBalance) // Carol gets no cut since the split rule has expired
}

func TestSplitRuleTxUpdate(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, _, _, carol, _, _, _ := setupForServicePayment(assert)
	et.fastforwardBy(1000)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	fakeInitiator := types.MakeAcc("User Eric")
	fakeInitiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(fakeInitiator)

	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 30,
	}
	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitCarol},
		Duration: uint64(100),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	splitRule := et.executor.state.Delivered().GetSplitRule(resourceID)
	assert.NotNil(splitRule)
	originalEndHeight := splitRule.EndBlockHeight
	log.Infof("originalEndHeight = %v", originalEndHeight)

	// Another user tries to update the split rule, should fail
	fakeSplitRuleUpdateTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  fakeInitiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitCarol},
		Duration: uint64(1000),
	}
	signBytes = fakeSplitRuleUpdateTx.SignBytes(et.chainID)
	fakeSplitRuleUpdateTx.Initiator.Signature = fakeInitiator.Sign(signBytes)

	res = et.executor.getTxExecutor(fakeSplitRuleUpdateTx).sanityCheck(et.chainID, et.state().Delivered(), fakeSplitRuleUpdateTx)
	assert.False(res.IsOK(), res.Message)
	assert.Equal(result.CodeUnauthorizedToUpdateSplitRule, res.Code)
	_, res = et.executor.getTxExecutor(fakeSplitRuleUpdateTx).process(et.chainID, et.state().Delivered(), fakeSplitRuleUpdateTx)
	assert.False(res.IsOK(), res.Message)
	assert.Equal(result.CodeUnauthorizedToUpdateSplitRule, res.Code)

	splitRule1 := et.executor.state.Delivered().GetSplitRule(resourceID)
	assert.NotNil(splitRule1)
	endHeight1 := splitRule1.EndBlockHeight
	assert.Equal(originalEndHeight, endHeight1)
	log.Infof("endHeight1 = %v", endHeight1)

	// The original initiator tries to update the split rule, should succeed
	extendedDuration := uint64(1000)
	splitRuleUpdateTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 2,
		},
		Splits:   []types.Split{splitCarol},
		Duration: extendedDuration,
	}
	signBytes = splitRuleUpdateTx.SignBytes(et.chainID)
	splitRuleUpdateTx.Initiator.Signature = initiator.Sign(signBytes)

	res = et.executor.getTxExecutor(splitRuleUpdateTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleUpdateTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleUpdateTx).process(et.chainID, et.state().Delivered(), splitRuleUpdateTx)
	assert.True(res.IsOK(), res.Message)

	splitRule2 := et.executor.state.Delivered().GetSplitRule(resourceID)
	assert.NotNil(splitRule2)
	currHeight := et.executor.state.Height()
	endHeight2 := splitRule2.EndBlockHeight
	assert.Equal(currHeight+extendedDuration, endHeight2)
	log.Infof("currHeight = %v", currHeight)
	log.Infof("endHeight2 = %v", endHeight2)
}

func TestSplitPaymentInternalMethod(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, _, _, _ := setupForServicePayment(assert)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 5,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 10,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 20,
	}

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitRule := &types.SplitRule{
		InitiatorAddress: initiator.Address,
		ResourceID:       resourceID,
		Splits:           []types.Split{splitCarol, splitAlice, splitBob, splitCarol, splitBob, splitBob}, // intentionally repeat splitBob and splitCarol here
		EndBlockHeight:   uint64(99999),
	}

	exec := NewServicePaymentTxExecutor(et.state())
	fullAmount := types.NewCoins(0, 10000)

	// carol is the target account
	success, addressCoinsMap := exec.splitPayment(et.state().Delivered(), splitRule, resourceID, carol.Address, fullAmount)

	assert.True(success)
	assert.Equal(3, len(addressCoinsMap))

	assert.Equal(addressCoinsMap[alice.Address], types.NewCoins(0, 10000*0.05))
	assert.Equal(addressCoinsMap[bob.Address], types.NewCoins(0, 10000*0.3))
	assert.Equal(addressCoinsMap[carol.Address], types.NewCoins(0, 10000*0.65))
}

func TestSplitRuleTxTargetAddressAlsoSplits(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 5,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 10,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 20,
	}
	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitCarol, splitBob, splitCarol}, // intentionally repeat splitCarol here
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 5 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 10 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 85 / 100), ThetaWei: big.NewInt(0)}
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleTxManyDups(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 5,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 5,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 0,
	}
	splits := []types.Split{splitAlice, splitCarol, splitBob, splitBob}
	for i := 0; i < 10; i++ {
		splits = append(splits, types.Split{
			Address:    bob.Address,
			Percentage: 0,
		})
	}
	for len(splits)+1 < types.MaxAccountsAffectedPerTx {
		splits = append(splits, splitCarol)
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   splits,
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Alice's final balance:   %v", aliceFinalBalance)
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 5 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 10 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.NewCoins(0, payAmount).Minus(aliceSplitCoins).Minus(bobSplitCoins)
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleTxSmallAmount(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Alice's initial balance:   %v", aliceInitBalance)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 5,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 10,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 0,
	}
	// Many empty splits.
	splits := []types.Split{splitAlice, splitCarol, splitBob}
	for len(splits)+1 < types.MaxAccountsAffectedPerTx {
		splits = append(splits, splitCarol)
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   splits,
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1 + txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	log.Infof("Payment amount: %v", payAmount)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 5 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 10 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.NewCoins(0, payAmount).Minus(aliceSplitCoins).Minus(bobSplitCoins)
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	log.Infof("Alice's final balance:   %v", aliceFinalBalance)
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleTxSmallPercentage(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Alice's initial balance:   %v", aliceInitBalance)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 5,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 10,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 0,
	}
	// Many accounts to receive small split.
	smallAddrs := []common.Address{}
	splits := []types.Split{splitAlice, splitCarol, splitBob}
	for len(splits)+1 < types.MaxAccountsAffectedPerTx {
		addr := common.BytesToAddress([]byte(fmt.Sprintf("small addr %d", len(smallAddrs))))
		smallAddrs = append(smallAddrs, addr)
		if len(smallAddrs) <= 85 {
			splits = append(splits, types.Split{
				Address:    addr,
				Percentage: 1,
			})
		} else {
			splits = append(splits, types.Split{
				Address:    addr,
				Percentage: 0,
			})
		}
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   splits, // intentionally repeat splitCarol here
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1 + txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	log.Infof("Payment amount: %v", payAmount)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	smallFinalBalance := types.NewCoins(0, 0)
	for _, addr := range smallAddrs {
		smallFinalBalance = smallFinalBalance.Plus(et.state().Delivered().GetAccount(addr).Balance)
	}

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 5 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 10 / 100), ThetaWei: big.NewInt(0)}
	smallSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 85 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.NewCoins(0, payAmount).Minus(aliceSplitCoins).Minus(bobSplitCoins).Minus(smallSplitCoins)
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	log.Infof("Alice's final balance:   %v", aliceFinalBalance)
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(smallSplitCoins, smallFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleHundredPercSplits(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	// 100% of the payment are split, 0% left for the target account
	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 70,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 30,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob}, // Alice and Bob split the payment, leaving nothing to Carol
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 70 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 30 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)} // Carol should get nothing
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleOverHundredPercSplits(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, _, _, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 91,
	}

	splitBob := types.Split{
		Address:    alice.Address,
		Percentage: 10,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.False(res.IsOK(), res.Message) // should be rejected
}

func TestSplitRuleSingleHundredPercSplits(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	// 100% of the payment are split, 0% left for the target account
	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 100,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice}, // Alice splits all the payment, leaving nothing to Carol
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 100 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)} // Carol should get nothing
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleSplitZeroPercSplits(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 0,
	}

	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 0,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount), ThetaWei: big.NewInt(0)} // Carol should get 100%
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleSplitEmptyRule(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{},
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount), ThetaWei: big.NewInt(0)} // Carol should get 100%
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleSplitZeroPayment(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 10,
	}

	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 20,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(0 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 0, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 0, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)} // zero payment
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)}   // zero payment
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(0), ThetaWei: big.NewInt(0)} // zero payment
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsZero())
}
func TestSplitRuleTxSplitPaymentRounding(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 27,
	}
	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 19,
	}
	splitCarol := types.Split{
		Address:    carol.Address,
		Percentage: 17,
	}
	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitCarol, splitBob, splitCarol}, // intentionally repeat splitCarol here
		Duration: uint64(99999),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(7238923) // down-rounding should occur for Alice and Bob's split calculation
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 27 / 100), ThetaWei: big.NewInt(0)}
	bobSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount * 19 / 100), ThetaWei: big.NewInt(0)}
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount - payAmount*27/100 - payAmount*19/100), ThetaWei: big.NewInt(0)}
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)
	log.Infof("Payment amount: %v TFuelWei", payAmount)
	log.Infof("Alice's split : %v TFuelWei", aliceSplitCoins.TFuelWei)
	log.Infof("Bob's   split : %v TFuelWei", bobSplitCoins.TFuelWei)
	log.Infof("Carol's split : %v TFuelWei", carolSplitCoins.TFuelWei)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleExpiration(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, carol, aliceInitBalance, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 10,
	}

	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 20,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(50),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	et.state().Commit()

	retrievedSplitRule := et.state().Delivered().GetSplitRule(resourceID)
	assert.NotNil(retrievedSplitRule)

	//
	// ------------------ Service payments after the split rule expires ------------------ //
	//

	et.fastforwardBy(100) // The split rule contract should expire after the fast-forward

	retrievedSplitRule2 := et.state().Delivered().GetSplitRule(resourceID)
	assert.NotNil(retrievedSplitRule2) // should still exists

	// Simulate micropayment #1 between Alice and Bob, Carol should get a cut
	payAmount := int64(1000 * txFee)
	srcSeq, tgtSeq, paymentSeq, reserveSeq := 1, 1, 1, 1

	// Alice send the service payment to Carol, whose address is included in the split address list
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 100*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	_ = createServicePaymentTx(et.chainID, &alice, &carol, 500*txFee, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	servicePaymentTx := createServicePaymentTx(et.chainID, &alice, &carol, payAmount, srcSeq, tgtSeq, paymentSeq, reserveSeq, resourceID)
	res = et.executor.getTxExecutor(servicePaymentTx).sanityCheck(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	assert.Equal(0, len(et.state().Delivered().GetSlashIntents()))
	_, res = et.executor.getTxExecutor(servicePaymentTx).process(et.chainID, et.state().Delivered(), servicePaymentTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	retrievedSplitRule3 := et.state().Delivered().GetSplitRule(resourceID)
	assert.Nil(retrievedSplitRule3) // should NOT exist, deleted by the ServicePaymentTx

	aliceFinalBalance := et.state().Delivered().GetAccount(alice.Address).Balance
	bobFinalBalance := et.state().Delivered().GetAccount(bob.Address).Balance
	carolFinalBalance := et.state().Delivered().GetAccount(carol.Address).Balance
	log.Infof("Bob's final balance:   %v", bobFinalBalance)
	log.Infof("Carol's final balance: %v", carolFinalBalance)

	// Check the balances of the relevant accounts
	aliceSplitCoins := types.Coins{}.NoNil()                                                 // Alice should get ZERO split since the split rule has expired
	bobSplitCoins := types.Coins{}.NoNil()                                                   // Bob should get ZERO split since the split rule has expired
	carolSplitCoins := types.Coins{TFuelWei: big.NewInt(payAmount), ThetaWei: big.NewInt(0)} // Carol should get the full payment since the split rule has expired
	aliceReservedFund := types.Coins{TFuelWei: big.NewInt(2001 * txFee), ThetaWei: big.NewInt(0)}
	reserveFundTxFee := types.NewCoins(0, getMinimumTxFee())
	servicePaymentTxFee := types.NewCoins(0, txFee)

	assert.Equal(aliceInitBalance.Minus(aliceReservedFund).Minus(reserveFundTxFee).Plus(aliceSplitCoins), aliceFinalBalance)
	assert.Equal(bobInitBalance.Plus(bobSplitCoins), bobFinalBalance)
	assert.Equal(carolInitBalance.Plus(carolSplitCoins).Minus(servicePaymentTxFee), carolFinalBalance)
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(alice.Address).Sequence) // seq=1 due to reserveFundTx
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(bob.Address).Sequence)
	assert.Equal(uint64(0), et.state().Delivered().GetAccount(carol.Address).Sequence)     // target's seq should not increase after servicePaymentTx
	assert.Equal(uint64(1), et.state().Delivered().GetAccount(initiator.Address).Sequence) // seq=1 due to splitRuleTx
	assert.Equal(1, len(et.state().Delivered().GetAccount(alice.Address).ReservedFunds))
	assert.True(et.state().Delivered().GetAccount(alice.Address).ReservedFunds[0].UsedFund.IsPositive())
}

func TestSplitRuleZeroDuration(t *testing.T) {
	assert := assert.New(t)
	et, resourceID, alice, bob, _, _, bobInitBalance, carolInitBalance := setupForServicePayment(assert)
	log.Infof("Bob's initial balance:   %v", bobInitBalance)
	log.Infof("Carol's initial balance: %v", carolInitBalance)

	txFee := getMinimumTxFee()

	initiator := types.MakeAcc("User David")
	initiator.Balance = types.Coins{TFuelWei: big.NewInt(10000 * txFee), ThetaWei: big.NewInt(0)}
	et.acc2State(initiator)

	splitAlice := types.Split{
		Address:    alice.Address,
		Percentage: 10,
	}

	splitBob := types.Split{
		Address:    bob.Address,
		Percentage: 20,
	}

	splitRuleTx := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 1,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(0),
	}
	signBytes := splitRuleTx.SignBytes(et.chainID)
	splitRuleTx.Initiator.Signature = initiator.Sign(signBytes)

	res := et.executor.getTxExecutor(splitRuleTx).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx).process(et.chainID, et.state().Delivered(), splitRuleTx)
	assert.True(res.IsOK(), res.Message)
	et.state().Commit()

	retrievedSplitRule := et.state().Delivered().GetSplitRule(resourceID)
	assert.NotNil(retrievedSplitRule)

	et.fastforwardBy(10) // The split rule contract should expire after the fast-forward

	resourceID2 := "resID2"
	splitRuleTx2 := &types.SplitRuleTx{
		Fee:        types.NewCoins(0, txFee),
		ResourceID: resourceID2,
		Initiator: types.TxInput{
			Address:  initiator.Address,
			Sequence: 2,
		},
		Splits:   []types.Split{splitAlice, splitBob},
		Duration: uint64(1000),
	}
	signBytes2 := splitRuleTx2.SignBytes(et.chainID)
	splitRuleTx2.Initiator.Signature = initiator.Sign(signBytes2)

	res = et.executor.getTxExecutor(splitRuleTx2).sanityCheck(et.chainID, et.state().Delivered(), splitRuleTx2)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(splitRuleTx2).process(et.chainID, et.state().Delivered(), splitRuleTx2)
	assert.True(res.IsOK(), res.Message)
	et.state().Commit()

	retrievedSplitRule2ndTime := et.state().Delivered().GetSplitRule(resourceID)
	assert.Nil(retrievedSplitRule2ndTime) // Should be expired and got deleted
}
