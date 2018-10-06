package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/types"
)

func TestGetInputs(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	// nil submissions
	acc, res := getInputs(nil, nil)
	assert.True(res.IsOK(), "getInputs: error on nil submission")
	assert.Zero(len(acc), "getInputs: accounts returned on nil submission")

	// test getInputs for registered, non-registered account
	et.reset()
	inputs := types.Accs2TxInputs(1, et.accIn)
	acc, res = getInputs(et.state(), inputs)
	assert.True(res.IsError(), "getInputs: expected error when using getInput with non-registered Input")

	et.acc2State(et.accIn)
	acc, res = getInputs(et.state(), inputs)
	assert.True(res.IsOK(), "getInputs: expected to getInput from registered Input")

	// test sending duplicate accounts
	et.reset()
	et.acc2State(et.accIn, et.accIn, et.accIn)
	inputs = types.Accs2TxInputs(1, et.accIn, et.accIn, et.accIn)
	acc, res = getInputs(et.state(), inputs)
	assert.True(res.IsError(), "getInputs: expected error when sending duplicate accounts")

	// test calculating reward
	et.reset()
	et.acc2State(et.accIn)

	et.fastforwardBy(1000) // fastforward to reach a sufficient height for Gamma generation

	inputs = types.Accs2TxInputs(1, et.accIn)
	acc, res = getInputs(et.state(), inputs)
	assert.True(res.IsOK(), "getInputs: expected to get input from a few block heights ago")
	assert.True(acc[string(inputs[0].Address[:])].Balance.GammaWei > et.accIn.Balance.GammaWei,
		"getInputs: expected to update input account gamma balance")
}

func TestGetOrMakeOutputs(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//nil submissions
	acc, res := getOrMakeOutputs(nil, nil, nil)
	assert.True(res.IsOK(), "getOrMakeOutputs: error on nil submission")
	assert.Zero(len(acc), "getOrMakeOutputs: accounts returned on nil submission")

	//test sending duplicate accounts
	et.reset()
	outputs := types.Accs2TxOutputs(et.accIn, et.accIn, et.accIn)
	_, res = getOrMakeOutputs(et.state(), nil, outputs)
	assert.True(res.IsError(), "getOrMakeOutputs: expected error when sending duplicate accounts")

	//test sending to existing/new account
	et.reset()
	outputs1 := types.Accs2TxOutputs(et.accIn)
	outputs2 := types.Accs2TxOutputs(et.accOut)

	et.acc2State(et.accIn)
	_, res = getOrMakeOutputs(et.state(), nil, outputs1)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to existing account")

	mapRes2, res := getOrMakeOutputs(et.state(), nil, outputs2)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to new account")

	//test the map results
	_, map2ok := mapRes2[string(outputs2[0].Address[:])]
	assert.True(map2ok, "getOrMakeOutputs: account output does not contain new account map item")

	//test calculating reward
	et.reset()
	et.fastforwardBy(1000) // fastforward to reach a sufficient height for Gamma generation

	outputs1 = types.Accs2TxOutputs(et.accIn)
	outputs2 = types.Accs2TxOutputs(et.accOut)

	et.acc2State(et.accIn)
	mapRes1, res := getOrMakeOutputs(et.state(), nil, outputs1)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to existing account")
	assert.True(mapRes1[string(outputs1[0].Address[:])].Balance.GammaWei > et.accIn.Balance.GammaWei,
		"getOrMakeOutputs: expected to update existing output account gamma balance")

	mapRes2, res = getOrMakeOutputs(et.state(), nil, outputs2)
	assert.True(res.IsOK(), "getOrMakeOutputs: error when sending to new account")
	assert.True(mapRes2[string(outputs2[0].Address[:])].Balance.GammaWei == 0,
		"getOrMakeOutputs: expected to not update new output account gamma balance")
}

func TestValidateInputsBasic(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//validate input basic
	inputs := types.Accs2TxInputs(1, et.accIn)
	res := validateInputsBasic(inputs)
	assert.True(res.IsOK(), "validateInputsBasic: expected no error on good tx input. Error: %v", res.Message)

	t.Log("inputs[0].Coins = ", inputs[0].Coins)
	inputs[0].Coins.ThetaWei = -1
	res = validateInputsBasic(inputs)
	//assert.True(res.IsError(), "validateInputsBasic: expected error on bad tx input")
	assert.True(res.IsOK(), "validateInputsBasic: expected error on bad tx input")
}

func TestValidateInputsAdvanced(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//create three temp accounts for the test
	accIn1 := types.MakeAcc("foox")
	accIn2 := types.MakeAcc("fooy")
	accIn3 := types.MakeAcc("fooz")

	//validate inputs advanced
	tx := types.MakeSendTx(1, et.accOut, accIn1, accIn2, accIn3)

	et.acc2State(accIn1, accIn2, accIn3, et.accOut)
	accMap, res := getInputs(et.state(), tx.Inputs)
	assert.True(res.IsOK(), "validateInputsAdvanced: error retrieving accMap. Error: %v", res.Message)
	signBytes := tx.SignBytes(et.chainID)

	//test bad case, unsigned
	totalCoins, res := validateInputsAdvanced(accMap, signBytes, tx.Inputs)
	assert.True(res.IsError(), "validateInputsAdvanced: expected an error on an unsigned tx input")

	//test good case sgined
	et.signSendTx(tx, accIn1, accIn2, accIn3, et.accOut)
	totalCoins, res = validateInputsAdvanced(accMap, signBytes, tx.Inputs)
	assert.True(res.IsOK(), "validateInputsAdvanced: expected no error on good tx input. Error: %v", res.Message)

	txTotalCoins := tx.Inputs[0].Coins.
		Plus(tx.Inputs[1].Coins).
		Plus(tx.Inputs[2].Coins)

	assert.True(totalCoins.IsEqual(txTotalCoins),
		"ValidateInputsAdvanced: transaction total coins are not equal: got %v, expected %v", txTotalCoins, totalCoins)
}

func TestValidateInputAdvanced(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//validate input advanced
	tx := types.MakeSendTx(1, et.accOut, et.accIn)

	et.acc2State(et.accIn, et.accOut)
	signBytes := tx.SignBytes(et.chainID)

	//unsigned case
	res := validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0])
	assert.True(res.IsError(), "validateInputAdvanced: expected error on tx input without signature")

	//good signed case
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0])
	assert.True(res.IsOK(), "validateInputAdvanced: expected no error on good tx input. Error: %v", res.Message)

	//bad sequence case
	et.accIn.Sequence = 1
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0])
	assert.Equal(result.CodeInvalidSequence, res.Code, "validateInputAdvanced: expected error on tx input with bad sequence")
	et.accIn.Sequence = 0 //restore sequence

	//bad balance case
	et.accIn.Balance = types.Coins{ThetaWei: 2}
	et.signSendTx(tx, et.accIn, et.accOut)
	res = validateInputAdvanced(&et.accIn.Account, signBytes, tx.Inputs[0])
	assert.Equal(result.CodeInsufficientFund, res.Code,
		"validateInputAdvanced: expected error on tx input with insufficient funds %v", et.accIn.Sequence)
}

func TestValidateOutputsBasic(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//validateOutputsBasic
	tx := types.Accs2TxOutputs(et.accIn)
	res := validateOutputsBasic(tx)
	assert.True(res.IsOK(), "validateOutputsBasic: expected no error on good tx output. Error: %v", res.Message)

	tx[0].Coins.ThetaWei = -1
	res = validateOutputsBasic(tx)
	assert.True(res.IsError(), "validateInputBasic: expected error on bad tx output. Error: %v", res.Message)
}

func TestSumOutput(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//SumOutput
	tx := types.Accs2TxOutputs(et.accIn, et.accOut)
	total := sumOutputs(tx)
	assert.True(total.IsEqual(tx[0].Coins.Plus(tx[1].Coins)), "sumOutputs: total coins are not equal")
}

func TestAdjustBy(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	//adjustByInputs/adjustByOutputs
	//sending transaction from accIn to accOut
	initBalIn := et.accIn.Account.Balance
	initBalOut := et.accOut.Account.Balance
	et.acc2State(et.accIn, et.accOut)

	txIn := types.Accs2TxInputs(1, et.accIn)
	txOut := types.Accs2TxOutputs(et.accOut)
	accMap, _ := getInputs(et.state(), txIn)
	accMap, _ = getOrMakeOutputs(et.state(), accMap, txOut)

	adjustByInputs(et.state(), accMap, txIn)
	adjustByOutputs(et.state(), accMap, txOut)

	inAddr := et.accIn.Account.PubKey.Address()
	outAddr := et.accOut.Account.PubKey.Address()
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
	et := newExecTest()

	//ExecTx
	tx := types.MakeSendTx(1, et.accOut, et.accIn)
	et.acc2State(et.accIn)
	et.acc2State(et.accOut)
	et.signSendTx(tx, et.accIn)

	//Bad Balance
	et.accIn.Balance = types.Coins{ThetaWei: 2}
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

func TestCalculateThetaReward(t *testing.T) {
	assert := assert.New(t)

	res := calculateThetaReward(1e17, true)
	assert.True(res.ThetaWei > 0)
}

func TestNonEmptyPubKey(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	_, userPubKey, err := crypto.TEST_GenerateKeyPairWithSeed("user")
	assert.Nil(err)
	userAddr := userPubKey.Address()
	et.state().SetAccount(userAddr, &types.Account{
		LastUpdatedBlockHeight: 1,
	})

	// ----------- Test 1: Both acc.PubKey and txInput.PubKey are empty -----------

	accInit, res := getAccount(et.state(), userAddr)
	assert.True(res.IsOK())
	assert.Nil(accInit.PubKey)

	txInput1 := types.TxInput{
		Address:  userAddr,
		Sequence: 1,
	} // Empty PubKey

	acc, res := getInput(et.state(), txInput1)
	assert.Equal(result.CodeEmptyPubKeyWithSequence1, res.Code)
	assert.True(acc == nil)

	// ----------- Test 2: acc.PubKey is empty, and txInput.PubKey is not empty -----------

	accInit, res = getAccount(et.state(), userAddr)
	assert.True(res.IsOK())
	assert.Nil(accInit.PubKey)

	txInput2 := types.TxInput{
		Address:  userAddr,
		PubKey:   userPubKey,
		Sequence: 2,
	}

	acc, res = getInput(et.state(), txInput2)
	assert.True(res.IsOK())
	assert.False(acc.PubKey.IsEmpty())
	assert.Equal(acc.PubKey, userPubKey)

	// ----------- Test 3: acc.PubKey is not empty, but txInput.PubKey is empty -----------

	et.state().SetAccount(userAddr, &types.Account{
		PubKey:                 userPubKey,
		LastUpdatedBlockHeight: 1,
	})

	accInit, res = getAccount(et.state(), userAddr)
	assert.True(res.IsOK())
	assert.False(accInit.PubKey.IsEmpty())

	txInput3 := types.TxInput{
		Address:  userAddr,
		Sequence: 3,
	} // Empty PubKey

	acc, res = getInput(et.state(), txInput3)
	assert.True(res.IsOK())
	assert.False(acc.PubKey.IsEmpty())
	assert.Equal(acc.PubKey, userPubKey)

	// ----------- Test 4: txInput contains another PubKey, should be ignored -----------

	_, userPubKey2, err := crypto.TEST_GenerateKeyPairWithSeed("lol")
	assert.Nil(err)

	accInit, res = getAccount(et.state(), userAddr)
	assert.True(res.IsOK())
	assert.False(accInit.PubKey.IsEmpty())

	txInput4 := types.TxInput{
		Address:  userAddr,
		Sequence: 4,
		PubKey:   userPubKey2,
	}

	acc, res = getInput(et.state(), txInput4)
	assert.True(res.IsOK())
	assert.False(acc.PubKey.IsEmpty())
	assert.Equal(userPubKey, acc.PubKey)     // acc.PukKey should not change
	assert.NotEqual(userPubKey2, acc.PubKey) // acc.PukKey should not change
}

func TestCoinbaseTx(t *testing.T) {
	assert := assert.New(t)
	et := newExecTest()

	va1 := et.accProposer
	va1.Balance = types.Coins{ThetaWei: 1e11}
	et.acc2State(va1)

	va2 := et.accVal2
	va2.Balance = types.Coins{ThetaWei: 3e11}
	et.acc2State(va2)

	user1 := types.MakeAcc("user 1")
	user1.Balance = types.Coins{ThetaWei: 1e11}
	et.acc2State(user1)

	et.fastforwardTo(1e7)

	var tx *types.CoinbaseTx
	var res result.Result

	//Regular check
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			va2.Account.PubKey.Address(), types.Coins{ThetaWei: 951},
		}},
		BlockHeight: 1e7,
	}
	tx.Proposer.Signature = va1.Sign(tx.SignBytes(et.chainID))

	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsOK(), res.String())

	//Error if reward Theta amount is incorrect
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			va2.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}},
		BlockHeight: 1e7,
	}
	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsError(), res.String())

	//Error if reward Gamma amount is incorrect
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			va2.Account.PubKey.Address(), types.Coins{ThetaWei: 951, GammaWei: 1},
		}},
		BlockHeight: 1e7,
	}
	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsError(), res.String())

	//Error if Validator 2 is not rewarded
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}},
		BlockHeight: 1e7,
	}
	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsError(), res.String())

	//Error if non-validator is rewarded
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			va2.Account.PubKey.Address(), types.Coins{ThetaWei: 951},
		}, {
			user1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}},
		BlockHeight: 1e7,
	}
	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsError(), res.String())

	//Error if validator address is changed
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			user1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}},
		BlockHeight: 1e7,
	}
	res = et.executor.coinbaseTxExec.sanityCheck(et.chainID, et.state(), tx)
	assert.True(res.IsError(), res.String())

	//Process should update validator account
	tx = &types.CoinbaseTx{
		Proposer: types.TxInput{
			Address: va1.PubKey.Address(), PubKey: va1.PubKey},
		Outputs: []types.TxOutput{{
			va1.Account.PubKey.Address(), types.Coins{ThetaWei: 317},
		}, {
			va2.Account.PubKey.Address(), types.Coins{ThetaWei: 951},
		}},
		BlockHeight: 1e7,
	}

	_, res = et.executor.coinbaseTxExec.process(et.chainID, et.state(), tx)
	assert.True(res.IsOK(), res.String())

	va1balance := et.state().GetAccount(va1.Account.PubKey.Address()).Balance
	assert.Equal(int64(100000000317), va1balance.ThetaWei)
	// validator's Gamma is also updated.
	assert.Equal(int64(189999981000), va1balance.GammaWei)

	va2balance := et.state().GetAccount(va2.Account.PubKey.Address()).Balance
	assert.Equal(int64(300000000951), va2balance.ThetaWei)
	assert.Equal(int64(569999943000), va2balance.GammaWei)

	user1balance := et.state().GetAccount(user1.Account.PubKey.Address()).Balance
	assert.Equal(int64(100000000000), user1balance.ThetaWei)
	// user's Gamma is not updated.
	assert.Equal(int64(0), user1balance.GammaWei)
}
