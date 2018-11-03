package vm

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/big"
	"strconv"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/state"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/store/database/backend"
)

func TestVMBasic(t *testing.T) {
	assert := assert.New(t)

	// ASM:
	// push 0x1
	// push 0x2
	// add
	// push 0x13
	// mstore8
	// push 0x1
	// push 0x13
	// return
	source := "600160020160135360016013f3"
	code, _ := hex.DecodeString(source)

	context := Context{}
	store := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	evm := NewEVM(context, store, nil, Config{})

	contract := &Contract{
		Code: code,
		Gas:  math.MaxUint64,
	}
	ret, err := evm.interpreter.Run(contract, []byte{}, false)
	assert.Nil(err)
	assert.Equal([]byte{0x3}, ret)
}

func TestVMStore(t *testing.T) {
	assert := assert.New(t)

	// ASM:
	// push 0x3
	// push 0x12
	// sstore
	source := "6003601255"
	code, _ := hex.DecodeString(source)

	context := Context{}
	store := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	evm := NewEVM(context, store, nil, Config{})

	contract := NewContract(
		AccountRef(common.HexToAddress("1133")),
		AccountRef(common.HexToAddress("2266")),
		new(big.Int),
		math.MaxUint64)
	contract.Code = code
	ret, err := evm.interpreter.Run(contract, []byte{}, false)
	assert.Nil(err)
	assert.Equal([]byte(nil), ret)

	loc := common.BigToHash(big.NewInt(0x12))
	actual := store.GetState(contract.Address(), loc)
	assert.Equal(common.BigToHash(big.NewInt(0x3)), actual)
}

func TestVMCreate(t *testing.T) {
	assert := assert.New(t)

	// ASM:
	// push 0x3
	// push 0x12
	// sstore
	source := "6003601255"
	code, _ := hex.DecodeString(source)
	addr := common.HexToAddress("1133")
	context := Context{}
	store := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	account := store.GetOrCreateAccount(addr)
	account.Balance = types.NewCoins(1000, 2000)
	store.SetAccount(addr, account)

	evm := NewEVM(context, store, nil, Config{})
	_, contractAddress, gas, err := evm.Create(AccountRef(addr), code, math.MaxUint64, big.NewInt(123))

	assert.Nil(err)
	assert.True(gas < math.MaxUint64)

	account2 := store.GetAccount(addr)
	assert.Equal(int64(1000), account2.Balance.ThetaWei.Int64())
	assert.Equal(int64(2000-123), account2.Balance.GammaWei.Int64())

	contractAcc := store.GetAccount(contractAddress)
	assert.NotNil(contractAcc)
	assert.Equal(int64(0), contractAcc.Balance.ThetaWei.Int64())
	assert.Equal(int64(123), contractAcc.Balance.GammaWei.Int64())
	ccode := store.GetCode(contractAddress)
	assert.Nil(ccode)

	loc := common.BigToHash(big.NewInt(0x12))
	actual := store.GetState(contractAddress, loc)
	assert.Equal(common.BigToHash(big.NewInt(0x3)), actual)
}

func TestContractDeployment(t *testing.T) {
	assert := assert.New(t)

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

	addr := common.HexToAddress("1133")
	context := Context{}
	store := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	account := store.GetOrCreateAccount(addr)
	account.Balance = types.NewCoins(1000, 2000)
	store.SetAccount(addr, account)

	evm := NewEVM(context, store, nil, Config{})
	_, contractAddress, _, err := evm.Create(AccountRef(addr), deployCode, math.MaxUint64, big.NewInt(123))

	assert.Nil(err)
	ccode := store.GetCode(contractAddress)
	assert.True(bytes.Equal(code, ccode))

	ret, leftOverGas, err := evm.Call(AccountRef(addr), contractAddress, nil, math.MaxUint64, big.NewInt(123))
	assert.Nil(err)
	assert.True(leftOverGas < math.MaxUint64)
	assert.Equal([]byte{0x3}, ret)
}

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
	deployerAddr := deployerAcc.PubKey.Address()
	valueAmount := 9723
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
	callerAddr := callerAcc.PubKey.Address()
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

// The test case below is based on a production Ethereum smart contract
// https://etherscan.io/tx/0xaf55bd233997065737195ee88e871d8bc282c44a5c11144f40865d699d8b8244
// The deplyCode hex string is the "Input Data" of the above transaction
func TestVMDeployComplexContract(t *testing.T) {
	assert := assert.New(t)

	storeView := state.NewStoreView(0, common.Hash{}, backend.NewMemDatabase())
	privAccounts := prepareInitState(storeView, 2)
	deployerAcc := privAccounts[0].Account
	//callerAcc := privAccounts[1].Account

	code, _ := hex.DecodeString("6060604052600436106100f1576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806303216695146100f65780631453671d1461011f5780631581b6001461015857806327bcc9ca146101ad5780632e1a7d4d146101c2578063313ce567146101fd57806332aeaddf1461022657806378e979251461024f5780638aa5b2c3146102785780638efe6dc41461029b578063c1c3eccf146102be578063c8b18b5b146102e7578063c9cda91f14610310578063d7ca7cc514610349578063e5b0ee4d1461036c578063fc0c546a1461038f578063fc6f9468146103e4575b600080fd5b341561010157600080fd5b610109610439565b6040518082815260200191505060405180910390f35b341561012a57600080fd5b610156600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061043f565b005b341561016357600080fd5b61016b6104e0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156101b857600080fd5b6101c0610506565b005b34156101cd57600080fd5b6101e360048080359060200190919050506105a6565b604051808215151515815260200191505060405180910390f35b341561020857600080fd5b610210610882565b6040518082815260200191505060405180910390f35b341561023157600080fd5b610239610887565b6040518082815260200191505060405180910390f35b341561025a57600080fd5b61026261088d565b6040518082815260200191505060405180910390f35b341561028357600080fd5b6102996004808035906020019091905050610893565b005b34156102a657600080fd5b6102bc60048080359060200190919050506108fa565b005b34156102c957600080fd5b6102d1610961565b6040518082815260200191505060405180910390f35b34156102f257600080fd5b6102fa610967565b6040518082815260200191505060405180910390f35b341561031b57600080fd5b610347600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061096e565b005b341561035457600080fd5b61036a6004808035906020019091905050610a0f565b005b341561037757600080fd5b61038d6004808035906020019091905050610a76565b005b341561039a57600080fd5b6103a2610add565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156103ef57600080fd5b6103f7610b03565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b60045481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561049b57600080fd5b81600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050565b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561056257600080fd5b60008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6000806000806000806000806105c760065442610b2890919063ffffffff16565b96506105f260016105e462278d008a610b4190919063ffffffff16565b610b5c90919063ffffffff16565b9550600254861015151561060557600080fd5b61061c600354600254610b5c90919063ffffffff16565b945060009350848610156106405761063d8686610b2890919063ffffffff16565b93505b30925061065860045485610b7a90919063ffffffff16565b9150600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231846000604051602001526040518263ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050602060405180830381600087803b151561071f57600080fd5b6102c65a03f1151561073057600080fd5b505050604051805190509050816107508a83610b2890919063ffffffff16565b1015151561075d57600080fd5b600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff168b6000604051602001526040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b151561084c57600080fd5b6102c65a03f1151561085d57600080fd5b50505060405180519050151561087257600080fd5b6001975050505050505050919050565b601281565b60025481565b60065481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156108ef57600080fd5b816006819055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561095657600080fd5b816004819055505050565b60035481565b62278d0081565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156109ca57600080fd5b81600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610a6b57600080fd5b816002819055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610ad257600080fd5b816003819055505050565b600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000828211151515610b3657fe5b818303905092915050565b6000808284811515610b4f57fe5b0490508091505092915050565b6000808284019050838110151515610b7057fe5b8091505092915050565b6000806000841415610b8f5760009150610bae565b8284029050828482811515610ba057fe5b04141515610baa57fe5b8091505b50929150505600a165627a7a72305820942ef2dbfbfcb1b3ddf6ca937286223a5c06ce470adf2ab35350b65bbf1426650029")
	deployCode, _ := hex.DecodeString("6060604052341561000f57600080fd5b60405160c080610d8d8339810160405280805190602001909190805190602001909190805190602001909190805190602001909190805190602001909190805190602001909190505060008673ffffffffffffffffffffffffffffffffffffffff161415151561007e57600080fd5b60008573ffffffffffffffffffffffffffffffffffffffff16141515156100a457600080fd5b6012600a0a606402821115156100b957600080fd5b856000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555084600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555083600281905550826003819055508160048190555080600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555042600681905550505050505050610be1806101ac6000396000f3006060604052600436106100f1576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff16806303216695146100f65780631453671d1461011f5780631581b6001461015857806327bcc9ca146101ad5780632e1a7d4d146101c2578063313ce567146101fd57806332aeaddf1461022657806378e979251461024f5780638aa5b2c3146102785780638efe6dc41461029b578063c1c3eccf146102be578063c8b18b5b146102e7578063c9cda91f14610310578063d7ca7cc514610349578063e5b0ee4d1461036c578063fc0c546a1461038f578063fc6f9468146103e4575b600080fd5b341561010157600080fd5b610109610439565b6040518082815260200191505060405180910390f35b341561012a57600080fd5b610156600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061043f565b005b341561016357600080fd5b61016b6104e0565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156101b857600080fd5b6101c0610506565b005b34156101cd57600080fd5b6101e360048080359060200190919050506105a6565b604051808215151515815260200191505060405180910390f35b341561020857600080fd5b610210610882565b6040518082815260200191505060405180910390f35b341561023157600080fd5b610239610887565b6040518082815260200191505060405180910390f35b341561025a57600080fd5b61026261088d565b6040518082815260200191505060405180910390f35b341561028357600080fd5b6102996004808035906020019091905050610893565b005b34156102a657600080fd5b6102bc60048080359060200190919050506108fa565b005b34156102c957600080fd5b6102d1610961565b6040518082815260200191505060405180910390f35b34156102f257600080fd5b6102fa610967565b6040518082815260200191505060405180910390f35b341561031b57600080fd5b610347600480803573ffffffffffffffffffffffffffffffffffffffff1690602001909190505061096e565b005b341561035457600080fd5b61036a6004808035906020019091905050610a0f565b005b341561037757600080fd5b61038d6004808035906020019091905050610a76565b005b341561039a57600080fd5b6103a2610add565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b34156103ef57600080fd5b6103f7610b03565b604051808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200191505060405180910390f35b60045481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561049b57600080fd5b81600160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050565b600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561056257600080fd5b60008060006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff16021790555050565b6000806000806000806000806105c760065442610b2890919063ffffffff16565b96506105f260016105e462278d008a610b4190919063ffffffff16565b610b5c90919063ffffffff16565b9550600254861015151561060557600080fd5b61061c600354600254610b5c90919063ffffffff16565b945060009350848610156106405761063d8686610b2890919063ffffffff16565b93505b30925061065860045485610b7a90919063ffffffff16565b9150600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff166370a08231846000604051602001526040518263ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808273ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001915050602060405180830381600087803b151561071f57600080fd5b6102c65a03f1151561073057600080fd5b505050604051805190509050816107508a83610b2890919063ffffffff16565b1015151561075d57600080fd5b600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1663a9059cbb600160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff168b6000604051602001526040518363ffffffff167c0100000000000000000000000000000000000000000000000000000000028152600401808373ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16815260200182815260200192505050602060405180830381600087803b151561084c57600080fd5b6102c65a03f1151561085d57600080fd5b50505060405180519050151561087257600080fd5b6001975050505050505050919050565b601281565b60025481565b60065481565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156108ef57600080fd5b816006819055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff1614151561095657600080fd5b816004819055505050565b60035481565b62278d0081565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff161415156109ca57600080fd5b81600560006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610a6b57600080fd5b816002819055505050565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff168073ffffffffffffffffffffffffffffffffffffffff163373ffffffffffffffffffffffffffffffffffffffff16141515610ad257600080fd5b816003819055505050565b600560009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000809054906101000a900473ffffffffffffffffffffffffffffffffffffffff1681565b6000828211151515610b3657fe5b818303905092915050565b6000808284811515610b4f57fe5b0490508091505092915050565b6000808284019050838110151515610b7057fe5b8091505092915050565b6000806000841415610b8f5760009150610bae565b8284029050828482811515610ba057fe5b04141515610baa57fe5b8091505b50929150505600a165627a7a72305820942ef2dbfbfcb1b3ddf6ca937286223a5c06ce470adf2ab35350b65bbf1426650029000000000000000000000000bfe0bd83cad590a29ab618e94c2d2757b9010c6b00000000000000000000000003e130eafab61ca4d31923b4043db497a830d2bd000000000000000000000000000000000000000000000000000000000000000c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000cecb8f27f4200f3a0000000000000000000000000000003883f5e181fccaf8410fa61e12b59bad963fb645")

	// First deploy a smart contract
	deployerAddr := deployerAcc.PubKey.Address()
	valueAmount := 0 // NOTE: For this contract, the value NEEDS to be ZERO
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address: deployerAddr,
			Coins:   types.NewCoins(0, valueAmount),
		},
		GasLimit: 1000000,
		GasPrice: big.NewInt(5000),
		Data:     deployCode,
	}
	vmRet, contractAddr, gasUsed, vmErr := Execute(deploySCTx, storeView)
	assert.Nil(vmErr)

	retrievedCode := storeView.GetCode(contractAddr)
	assert.True(bytes.Equal(code, retrievedCode))

	log.Infof("Deploy Contract -- contractAddr: %v, gasUsed: %v, vmRet: %v", contractAddr.Hex(), gasUsed, hex.EncodeToString(vmRet))
}

// ----------- Utilities ----------- //

func prepareInitState(storeView *state.StoreView, numAccounts int) (privAccounts []types.PrivAccount) {
	for i := 0; i < numAccounts; i++ {
		secret := "acc_secret_" + strconv.FormatInt(int64(i), 16)
		privAccount := types.MakeAccWithInitBalance(secret, types.NewCoins(90000000, 50000000000))
		privAccounts = append(privAccounts, privAccount)
		storeView.SetAccount(privAccount.Account.PubKey.Address(), &privAccount.Account)
	}

	storeView.IncrementHeight()
	storeView.Save()

	return privAccounts
}
