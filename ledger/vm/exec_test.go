package vm

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database/backend"
)

func TestVMExecute(t *testing.T) {
	assert := assert.New(t)

	storeView := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	privAccounts := prepareInitState(storeView, 2)
	deployerAcc := privAccounts[0].Account
	callerAcc := privAccounts[1].Account

	// ASM:
	// push 0x3
	// push 0x13
	// mstore8
	// push 0x1
	// push 0x13
	// return
	code, _ := hex.DecodeString("600360135360016013f3")

	// ASM:
	// push 0xa
	// push 0xc
	// push 0x0
	// codecopy
	// push 0xa
	// push 0x0
	// return
	// push 0x3
	// push 0x13
	// mstore8
	// push 0x1
	// push 0x13
	// return
	deployCode, _ := hex.DecodeString("600a600c600039600a6000f3600360135360016013f3")

	// First deploy a smart contract
	deployerAddr := deployerAcc.Address
	valueAmount := int64(9723)
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address: deployerAddr,
			Coins:   types.NewCoins(0, valueAmount),
		},
		GasLimit: 60000,
		GasPrice: big.NewInt(5000),
		Data:     deployCode,
	}
	vmRet, contractAddr, gasUsed, vmErr := Execute(deploySCTx, storeView)
	assert.Nil(vmErr)
	retrievedCode := storeView.GetCode(contractAddr)
	assert.True(bytes.Equal(code, retrievedCode))

	storeView.Save()

	// Note: the gas fee deduction is handled by the ledger executor, so NO gas deduction here
	retrievedDeployerAcc := storeView.GetAccount(deployerAddr)
	deployerTransferredValue := deployerAcc.Balance.Minus(retrievedDeployerAcc.Balance)
	assert.True(types.NewCoins(0, valueAmount).IsEqual(deployerTransferredValue))

	contractBalance := storeView.GetBalance(contractAddr)
	assert.True(big.NewInt(int64(valueAmount)).Cmp(contractBalance) == 0)

	log.Infof("Deploy Contract -- contractAddr: %v, gasUsed: %v, vmRet: %v", contractAddr.Hex(), gasUsed, hex.EncodeToString(vmRet))

	// Call the smart contract
	callerAddr := callerAcc.Address
	callSCTX := &types.SmartContractTx{
		From:     types.TxInput{Address: callerAddr},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: 60000,
		GasPrice: big.NewInt(5000),
		Data:     nil,
	}
	vmRet, _, gasUsed, vmErr = Execute(callSCTX, storeView)
	assert.Nil(vmErr)
	assert.Equal(common.Bytes{0x3}, vmRet)

	log.Infof("Call   Contract -- contractAddr: %v, gasUsed: %v, vmRet: %v, ", contractAddr.Hex(), gasUsed, hex.EncodeToString(vmRet))

	storeView.Save()
	retrievedCallerAcc := storeView.GetAccount(callerAddr)
	callerTransferredValue := callerAcc.Balance.Minus(retrievedCallerAcc.Balance)
	assert.True(types.NewCoins(0, 0).IsEqual(callerTransferredValue)) // Caller transferred no value, also Gas fee should NOT be deducted
}

// This test case deploy the bytecode of the following Solidity contract on the
// blockchain and interfact with it
//
// pragma solidity ^0.4.18;
// contract SquareCalculator {
//     uint public value;
//
//     function SetValue(uint val) public {
//         value = val;
//     }
//
//     function CalculateSquare() constant public returns (uint) {
//         uint sqr = value * value;
//         assert(sqr / value == value); // overflow protection
//         return sqr;
//     }
// }
func TestVMExecutionInteractWithContract(t *testing.T) {
	assert := assert.New(t)

	storeView := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	privAccounts := prepareInitState(storeView, 2)
	deployerAcc := privAccounts[0].Account
	callerAcc := privAccounts[1].Account

	var cbc contractByteCode
	err := loadJSONTest("testdata/square_calculator.json", &cbc)
	assert.Nil(err)

	deploymentCode, err := hex.DecodeString(cbc.DeploymentCode)
	assert.Nil(err)
	code, err := hex.DecodeString(cbc.Code)
	assert.Nil(err)

	// First deploy a smart contract
	deployerAddr := deployerAcc.Address
	valueAmount := int64(0) // NOTE: For this contract, the value NEEDS to be ZERO
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address: deployerAddr,
			Coins:   types.NewCoins(0, valueAmount),
		},
		GasLimit: 200000,
		GasPrice: big.NewInt(50),
		Data:     deploymentCode,
	}
	vmRet, contractAddr, gasUsed, vmErr := Execute(deploySCTx, storeView)
	assert.Nil(vmErr)
	assert.True(bytes.Equal(code, vmRet))

	retrievedCode := storeView.GetCode(contractAddr)
	assert.True(bytes.Equal(code, retrievedCode))

	log.Infof("Deploy Contract -- contractAddr: %v, gasUsed: %v", contractAddr.Hex(), gasUsed)

	// Call the smart contract
	callerAddr := callerAcc.Address
	callSCTXTmpl := &types.SmartContractTx{
		From:     types.TxInput{Address: callerAddr},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: 50000,
		GasPrice: big.NewInt(50),
		Data:     nil,
	}

	// Set the value and then calculate its square
	value := uint64(18327)
	setValueCallTx := callSCTXTmpl
	setValueCallData, _ := hex.DecodeString("ed8b07060000000000000000000000000000000000000000000000000000000000004797") // "ed8b0706" is signature of the SetValue() interface, and 0x4797 is the hex of the value 18327
	setValueCallTx.Data = setValueCallData
	_, _, gasUsed, vmErr = Execute(setValueCallTx, storeView)
	assert.Nil(vmErr)
	log.Infof("Call   Contract -- SetValue: %v, gasUsed: %v", value, gasUsed)

	expectedSquare := new(big.Int).SetUint64(value * value)
	calculateSquareCallTx := callSCTXTmpl
	calculateSquareCallData, _ := hex.DecodeString("b5a0241a") // signature of the CalculateSquare() interface
	calculateSquareCallTx.Data = calculateSquareCallData
	vmRet, _, gasUsed, vmErr = Execute(setValueCallTx, storeView)
	calculatedSquare, success := new(big.Int).SetString(hex.EncodeToString(vmRet), 16)
	assert.True(success)
	assert.Equal(expectedSquare, calculatedSquare)
	log.Infof("Call   Contract -- calculatedSquare: %v, gasUsed: %v", calculatedSquare, gasUsed)
}

// The test case below is based on a production TimeLockedSafe Ethereum smart contract
// https://etherscan.io/tx/0xaf55bd233997065737195ee88e871d8bc282c44a5c11144f40865d699d8b8244
// The deplyment_code hex string in testdata/time_locked_safe.json is the "Input Data" of the above transaction
func TestVMExecutionDeployComplexContract(t *testing.T) {
	assert := assert.New(t)

	storeView := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	privAccounts := prepareInitState(storeView, 2)
	deployerAcc := privAccounts[0].Account
	callerAcc := privAccounts[1].Account

	var cbc contractByteCode
	err := loadJSONTest("testdata/time_locked_safe.json", &cbc)
	assert.Nil(err)

	deploymentCode, err := hex.DecodeString(cbc.DeploymentCode)
	assert.Nil(err)
	code, err := hex.DecodeString(cbc.Code)
	assert.Nil(err)

	// First deploy a smart contract
	deployerAddr := deployerAcc.Address
	valueAmount := int64(0) // NOTE: For this contract, the value NEEDS to be ZERO
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address: deployerAddr,
			Coins:   types.NewCoins(0, valueAmount),
		},
		GasLimit: 2000000,
		GasPrice: big.NewInt(50),
		Data:     deploymentCode,
	}
	vmRet, contractAddr, gasUsed, vmErr := Execute(deploySCTx, storeView)
	assert.Nil(vmErr)
	assert.True(bytes.Equal(code, vmRet))

	retrievedCode := storeView.GetCode(contractAddr)
	assert.True(bytes.Equal(code, retrievedCode))

	log.Infof("Deploy Contract -- contractAddr: %v, gasUsed: %v", contractAddr.Hex(), gasUsed)

	// Call the smart contract
	callerAddr := callerAcc.Address
	callSCTXTmpl := &types.SmartContractTx{
		From:     types.TxInput{Address: callerAddr},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: 50000,
		GasPrice: big.NewInt(50),
		Data:     nil,
	}

	monthlyWithdrawLimitInWeiCallTx := callSCTXTmpl
	monthlyWithdrawLimitInWeiCallData, _ := hex.DecodeString("03216695") // signature of the monthlyWithdrawLimitInWei() interface
	monthlyWithdrawLimitInWeiCallTx.Data = monthlyWithdrawLimitInWeiCallData
	vmRet, _, gasUsed, vmErr = Execute(monthlyWithdrawLimitInWeiCallTx, storeView)
	assert.Nil(vmErr)
	monthlyWithdrawLimitInWei, success := new(big.Int).SetString(hex.EncodeToString(vmRet), 16)
	assert.True(success)
	expectedWithdrawLimit := new(big.Int).SetUint64(2500000000000000000)
	expectedWithdrawLimit.Mul(expectedWithdrawLimit, new(big.Int).SetUint64(100000000))
	assert.Equal(expectedWithdrawLimit, monthlyWithdrawLimitInWei)
	log.Infof("Call   Contract -- monthlyWithdrawLimitInWei: %v", monthlyWithdrawLimitInWei)

	lockingPeriodInMonthsCallTx := callSCTXTmpl
	lockingPeriodInMonthsCallData, _ := hex.DecodeString("32aeaddf") // signature of the lockingPeriodInMonths() interface
	lockingPeriodInMonthsCallTx.Data = lockingPeriodInMonthsCallData
	vmRet, _, gasUsed, vmErr = Execute(lockingPeriodInMonthsCallTx, storeView)
	assert.Nil(vmErr)
	lockingPeriodInMonths, success := new(big.Int).SetString(hex.EncodeToString(vmRet), 16)
	assert.True(success)
	assert.Equal(new(big.Int).SetInt64(12), lockingPeriodInMonths)
	log.Infof("Call   Contract -- lockingPeriodInMonths: %v", lockingPeriodInMonths)

	tokenAddressCallTx := callSCTXTmpl
	tokenAddressCallData, _ := hex.DecodeString("fc0c546a") // signature of the token() interface
	tokenAddressCallTx.Data = tokenAddressCallData
	vmRet, _, gasUsed, vmErr = Execute(tokenAddressCallTx, storeView)
	assert.Nil(vmErr)
	expectedTokenAddrBytes, _ := hex.DecodeString("3883f5e181fccaF8410FA61e12b59BAd963fb645")
	expectedTokenAddr := common.BytesToAddress(expectedTokenAddrBytes)
	retrievedTokenAddr := common.BytesToAddress(vmRet)
	assert.Equal(expectedTokenAddr, retrievedTokenAddr)
	log.Infof("Call   Contract -- retrievedTokenAddr: %v", retrievedTokenAddr)
}

// The test case below is based on the production Theta ERC20 Token smart contract deployed on the Ethereum blockchain
// https://etherscan.io/tx/0x078358d68d132458fc964cfb19011f8e561da5c4ebb6e47b27032813d684861b
// The deplyment_code hex string in testdata/erc20_token.json is the "Input Data" of the above transaction
func TestVMExecutionDeployERC20TokenContract(t *testing.T) {
	assert := assert.New(t)

	storeView := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	privAccounts := prepareInitState(storeView, 2)
	deployerAcc := privAccounts[0].Account
	callerAcc := privAccounts[1].Account

	var cbc contractByteCode
	err := loadJSONTest("testdata/erc20_token.json", &cbc)
	assert.Nil(err)

	deploymentCode, err := hex.DecodeString(cbc.DeploymentCode)
	assert.Nil(err)
	code, err := hex.DecodeString(cbc.Code)
	assert.Nil(err)

	// First deploy a smart contract
	deployerAddr := deployerAcc.Address
	valueAmount := int64(0) // NOTE: For this contract, the value NEEDS to be ZERO
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address: deployerAddr,
			Coins:   types.NewCoins(0, valueAmount),
		},
		GasLimit: 3000000,
		GasPrice: big.NewInt(50),
		Data:     deploymentCode,
	}
	vmRet, contractAddr, gasUsed, vmErr := Execute(deploySCTx, storeView)
	assert.Nil(vmErr)
	assert.True(bytes.Equal(code, vmRet))

	retrievedCode := storeView.GetCode(contractAddr)
	assert.True(bytes.Equal(code, retrievedCode))

	log.Infof("Deploy Contract -- contractAddr: %v, gasUsed: %v", contractAddr.Hex(), gasUsed)

	// Call the smart contract
	callerAddr := callerAcc.Address
	callSCTXTmpl := &types.SmartContractTx{
		From:     types.TxInput{Address: callerAddr},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: 50000,
		GasPrice: big.NewInt(50),
		Data:     nil,
	}

	nameCallTx := callSCTXTmpl
	nameCallData, _ := hex.DecodeString("06fdde03") // signature of the name() interface
	nameCallTx.Data = nameCallData
	vmRet, _, gasUsed, vmErr = Execute(nameCallTx, storeView)
	assert.Nil(vmErr)
	name := string(vmRet[64:75])
	assert.Equal("Theta Token", name)
	log.Infof("Call   Contract -- name: %v", name)

	symbolCallTx := callSCTXTmpl
	symbolCallData, _ := hex.DecodeString("95d89b41") // signature of the symbol() interface
	symbolCallTx.Data = symbolCallData
	vmRet, _, gasUsed, vmErr = Execute(symbolCallTx, storeView)
	assert.Nil(vmErr)
	symbol := string(vmRet[64:69])
	assert.Equal("THETA", symbol)
	log.Infof("Call   Contract -- symbol: %v", symbol)
}

// ----------- Utilities ----------- //

func prepareInitState(storeView *state.StoreView, numAccounts int) (privAccounts []types.PrivAccount) {
	for i := 0; i < numAccounts; i++ {
		secret := "acc_secret_" + strconv.FormatInt(int64(i), 16)
		privAccount := types.MakeAccWithInitBalance(secret, types.NewCoins(90000000, 50000000000))
		privAccounts = append(privAccounts, privAccount)
		storeView.SetAccount(privAccount.Address, &privAccount.Account)
	}

	storeView.IncrementHeight()
	storeView.Save()

	return privAccounts
}

type contractByteCode struct {
	DeploymentCode string `json:"deployment_code"`
	Code           string `json:"code"`
}

func loadJSONTest(file string, val interface{}) error {
	content, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(content, val); err != nil {
		if syntaxerr, ok := err.(*json.SyntaxError); ok {
			line := findLine(content, syntaxerr.Offset)
			return fmt.Errorf("JSON syntax error at %v:%v: %v", file, line, err)
		}
		return fmt.Errorf("JSON unmarshal error in %v: %v", file, err)
	}
	return nil
}

func findLine(data []byte, offset int64) (line int) {
	line = 1
	for i, r := range string(data) {
		if int64(i) >= offset {
			return
		}
		if r == '\n' {
			line++
		}
	}
	return
}
