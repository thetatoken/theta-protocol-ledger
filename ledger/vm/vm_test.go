package vm

import (
	"bytes"
	"encoding/hex"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/store/database/backend"
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
	assert.Equal(int64(2000-123), account2.Balance.TFuelWei.Int64())

	contractAcc := store.GetAccount(contractAddress)
	assert.NotNil(contractAcc)
	assert.Equal(int64(0), contractAcc.Balance.ThetaWei.Int64())
	assert.Equal(int64(123), contractAcc.Balance.TFuelWei.Int64())
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
