// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package vm

import (
	"math/big"
	"sync/atomic"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/crypto/bls"
	"github.com/thetatoken/theta/ledger/state"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/ledger/vm/params"
)

type (
	// CanTransferFunc is the signature of a transfer guard function
	CanTransferFunc func(StateDB, common.Address, *big.Int) bool
	// TransferFunc is the signature of a transfer function
	TransferFunc func(StateDB, common.Address, common.Address, *big.Int)
	// GetHashFunc returns the nth block hash in the blockchain
	// and is used by the BLOCKHASH EVM op code.
	GetHashFunc func(uint64) common.Hash
)

func SupportThetaTransferInEVM(blockHeight uint64) bool {
	return blockHeight >= common.HeightSupportThetaTokenInSmartContract
}

func SupportWrappedTheta(blockHeight uint64) bool {
	return blockHeight >= common.HeightSupportWrappedTheta
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// CanTransferTheta checks whether there are enough funds in the address' account to make a Theta transfer.
func CanTransferTheta(db StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetThetaBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}

// TransferTheta subtracts the given amount of Theta from sender and adds amount to recipient using the given Db
func TransferTheta(db StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubThetaBalance(sender, amount)
	db.AddThetaBalance(recipient, amount)
}

func parseBLSSummary(summary []byte) (holderAddress common.Address, blsPubkey *bls.PublicKey, blsPop *bls.Signature, holderSig *crypto.Signature, ok bool) {
	if len(summary) != 229 && len(summary) != 261 {
		return
	}

	holderAddress = common.BytesToAddress(summary[:20])
	blsPubkey, err := bls.PublicKeyFromBytes(summary[20:68])
	if err != nil {
		return
	}
	blsPop, err = bls.SignatureFromBytes(summary[68:164])
	if err != nil {
		return
	}
	holderSig, err = crypto.SignatureFromBytes(summary[164:])
	if err != nil {
		return
	}

	ok = true
	return
}

func CheckBLSSummary(summary []byte) bool {
	guardianAddr, blsPubkey, blsPop, holderSig, ok := parseBLSSummary(summary)
	if !ok {
		return false
	}

	return checkBlsSummary(blsPubkey, blsPop, holderSig, guardianAddr)
}

func checkBlsSummary(blsPubkey *bls.PublicKey, blsPop *bls.Signature, holderSig *crypto.Signature, guardianAddr common.Address) bool {
	if blsPubkey.IsEmpty() {
		return false
	}
	if blsPop.IsEmpty() {
		return false
	}
	if holderSig == nil || holderSig.IsEmpty() {
		return false
	}

	if !holderSig.Verify(blsPop.ToBytes(), guardianAddr) {
		return false
	}

	if !blsPop.PopVerify(blsPubkey) {
		return false
	}

	return true
}

// StakeToGuardian stake Theta to given guardian node.
func StakeToGuardian(db StateDB, sender common.Address, guardianSummary []byte, amount *big.Int) bool {
	// if amount.Cmp(core.MinGuardianStakeDeposit) < 0 {
	// 	return false
	// }
	if db.GetThetaBalance(sender).Cmp(amount) < 0 {
		return false
	}

	guardianAddr, blsPubkey, blsPop, holderSig, ok := parseBLSSummary(guardianSummary)
	if !ok {
		return false
	}

	view := db.(*state.StoreView)
	gcp := view.GetGuardianCandidatePool()
	if !gcp.Contains(guardianAddr) {
		if !checkBlsSummary(blsPubkey, blsPop, holderSig, guardianAddr) {
			return false
		}
	}

	err := gcp.DepositStake(sender, guardianAddr, amount, blsPubkey, view.GetBlockHeight())
	if err != nil {
		return false
	}

	view.UpdateGuardianCandidatePool(gcp)
	db.SubThetaBalance(sender, amount)

	return true
}

// UnstakeFromGuardian unstake from Guardians.
func UnstakeFromGuardian(db StateDB, addr common.Address, guardianAddr common.Address) bool {
	view := db.(*state.StoreView)
	gcp := view.GetGuardianCandidatePool()
	currentHeight := view.Height()
	err := gcp.WithdrawStake(addr, guardianAddr, currentHeight)
	if err != nil {
		return false
	}

	view.UpdateGuardianCandidatePool(gcp)
	return true
}

// StakeToEEN stake to given EEN node.
func StakeToEEN(db StateDB, sender common.Address, summary []byte, amount *big.Int) bool {
	// minEliteEdgeNodeStake := core.MinEliteEdgeNodeStakeDeposit
	// maxEliteEdgeNodeStake := core.MaxEliteEdgeNodeStakeDeposit

	// if amount.Cmp(minEliteEdgeNodeStake) < 0 {
	// 	return false
	// }

	eenAddr, blsPubkey, blsPop, holderSig, ok := parseBLSSummary(summary)
	if !ok {
		return false
	}

	view := db.(*state.StoreView)

	// currentStake := big.NewInt(0)

	eenp := state.NewEliteEdgeNodePool(view, false)
	een := eenp.Get(eenAddr)
	// if een != nil {
	// 	currentStake = een.TotalStake()
	// }

	// expectedStake := big.NewInt(0).Add(currentStake, amount)
	// if expectedStake.Cmp(maxEliteEdgeNodeStake) > 0 {
	// 	return false
	// }

	if db.GetBalance(sender).Cmp(amount) < 0 {
		return false
	}

	if een == nil && !checkBlsSummary(blsPubkey, blsPop, holderSig, eenAddr) {
		return false
	}

	err := eenp.DepositStake(sender, eenAddr, amount, blsPubkey, view.GetBlockHeight())
	if err != nil {
		return false
	}

	db.SubBalance(sender, amount)

	return true
}

// UnstakeFromEEN unstake from EEN.
func UnstakeFromEEN(db StateDB, addr common.Address, eenAddr common.Address) bool {
	view := db.(*state.StoreView)

	eenp := state.NewEliteEdgeNodePool(view, false)

	currentHeight := view.Height()

	withdrawnStake, err := eenp.WithdrawStake(addr, eenAddr, currentHeight)
	if err != nil || withdrawnStake == nil {
		return false
	}

	returnHeight := withdrawnStake.ReturnHeight
	stakesToBeReturned := view.GetEliteEdgeNodeStakeReturns(returnHeight)
	stakesToBeReturned = append(stakesToBeReturned, state.StakeWithHolder{
		Holder: eenAddr,
		Stake:  *withdrawnStake,
	})
	view.SetEliteEdgeNodeStakeReturns(returnHeight, stakesToBeReturned)

	return true
}

func getPrecompiledContracts(blockHeight uint64) map[common.Address]PrecompiledContract {
	var precompiles map[common.Address]PrecompiledContract
	if blockHeight < common.HeightSupportThetaTokenInSmartContract {
		precompiles = PrecompiledContractsByzantium
	} else if blockHeight < common.HeightSupportWrappedTheta {
		precompiles = PrecompiledContractsThetaSupport
	} else {
		precompiles = PrecompiledContractsWrappedThetaSupport
	}
	return precompiles
}

// run runs the given contract and takes care of running precompiles with a fallback to the byte code interpreter.
func run(evm *EVM, contract *Contract, input []byte, readOnly bool) ([]byte, error) {
	if contract.CodeAddr != nil {
		blockHeight := evm.StateDB.GetBlockHeight()
		precompiles := getPrecompiledContracts(blockHeight)
		if p := precompiles[*contract.CodeAddr]; p != nil {
			return RunPrecompiledContract(evm, p, input, contract)
		}
	}
	for _, interpreter := range evm.interpreters {
		if interpreter.CanRun(contract.Code) {
			if evm.interpreter != interpreter {
				// Ensure that the interpreter pointer is set back
				// to its current value upon return.
				defer func(i Interpreter) {
					evm.interpreter = i
				}(evm.interpreter)
				evm.interpreter = interpreter
			}
			return interpreter.Run(contract, input, readOnly)
		}
	}
	return nil, ErrNoCompatibleInterpreter
}

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// CanTransfer returns whether the account contains
	// sufficient ether to transfer the value
	CanTransfer CanTransferFunc
	// Transfer transfers ether from one account to the other
	Transfer TransferFunc
	// GetHash returns the hash corresponding to n
	GetHash GetHashFunc

	// Message information
	Origin   common.Address // Provides information for ORIGIN
	GasPrice *big.Int       // Provides information for GASPRICE

	// Block information
	Coinbase    common.Address // Provides information for COINBASE
	GasLimit    uint64         // Provides information for GASLIMIT
	BlockNumber *big.Int       // Provides information for NUMBER
	Time        *big.Int       // Provides information for TIME
	Difficulty  *big.Int       // Provides information for DIFFICULTY
}

// EVM is the Ethereum Virtual Machine base object and provides
// the necessary tools to run a contract on the given state with
// the provided context. It should be noted that any error
// generated through any of the calls should be considered a
// revert-state-and-consume-all-gas operation, no checks on
// specific errors should ever be performed. The interpreter makes
// sure that any errors generated are to be considered faulty code.
//
// The EVM should never be reused and is not thread safe.
type EVM struct {
	// Context provides auxiliary blockchain related information
	Context
	// StateDB gives access to the underlying state
	StateDB StateDB
	// Depth is the current call stack
	depth int

	// chainConfig contains information about the current chain
	chainConfig *params.ChainConfig
	// chain rules contains the chain rules for the current epoch
	chainRules params.Rules
	// virtual machine configuration options used to initialise the
	// evm.
	vmConfig Config
	// global (to this context) ethereum virtual machine
	// used throughout the execution of the tx.
	interpreters []Interpreter
	interpreter  Interpreter
	// abort is used to abort the EVM calling operations
	// NOTE: must be set atomically
	abort int32
	// callGasTemp holds the gas available for the current call. This is needed because the
	// available gas is calculated in gasCall* according to the 63/64 rule and later
	// applied in opCall*.
	callGasTemp uint64
}

// NewEVM returns a new EVM. The returned EVM is not thread safe and should
// only ever be used *once*.
func NewEVM(ctx Context, statedb StateDB, chainConfig *params.ChainConfig, vmConfig Config) *EVM {
	evm := &EVM{
		Context:     ctx,
		StateDB:     statedb,
		vmConfig:    vmConfig,
		chainConfig: chainConfig,
		// chainRules:   chainConfig.Rules(ctx.BlockNumber),
		interpreters: make([]Interpreter, 0, 1),
	}

	// vmConfig.EVMInterpreter will be used by EVM-C, it won't be checked here
	// as we always want to have the built-in EVM as the failover option.
	evm.interpreters = append(evm.interpreters, NewEVMInterpreter(evm, vmConfig))
	evm.interpreter = evm.interpreters[0]

	return evm
}

// Cancel cancels any running EVM operation. This may be called concurrently and
// it's safe to be called multiple times.
func (evm *EVM) Cancel() {
	atomic.StoreInt32(&evm.abort, 1)
}

// Interpreter returns the current interpreter
func (evm *EVM) Interpreter() Interpreter {
	return evm.interpreter
}

// Call executes the contract associated with the addr with the given input as
// parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
func (evm *EVM) Call(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int, thetaValue *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	blockHeight := evm.StateDB.GetBlockHeight()

	// Fail if we're trying to transfer more than the available balance
	if !CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	if SupportThetaTransferInEVM(blockHeight) && !CanTransferTheta(evm.StateDB, caller.Address(), thetaValue) {
		return nil, gas, ErrInsufficientThetaBlance
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	if !evm.StateDB.Exist(addr) {

		precompiles := getPrecompiledContracts(blockHeight)
		if precompiles[addr] == nil && value.Sign() == 0 {
			// Calling a non existing account, don't do anything, but ping the tracer
			if evm.vmConfig.Debug && evm.depth == 0 {
				evm.vmConfig.Tracer.CaptureStart(caller.Address(), addr, false, input, gas, value)
				evm.vmConfig.Tracer.CaptureEnd(ret, 0, 0, nil)
			}
			return nil, gas, nil
		}

		if !SupportThetaTransferInEVM(blockHeight) { // just for backward compatibility
			evm.StateDB.CreateAccount(addr)
		} else { // should not wipe out the Theta/TFuel balance sent to the contract address prior to contract creation
			evm.StateDB.CreateAccountWithPreviousBalance(addr)
		}
	}
	Transfer(evm.StateDB, caller.Address(), to.Address(), value)

	if SupportThetaTransferInEVM(blockHeight) {
		TransferTheta(evm.StateDB, caller.Address(), to.Address(), thetaValue)
	}

	// Initialise a new contract and set the code that is to be used by the EVM.
	// The contract is a scoped environment for this execution context only.
	contract := NewContract(caller, to, value, thetaValue, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false)

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// CallCode executes the contract associated with the addr with the given input
// as parameters. It also handles any necessary value transfer required and takes
// the necessary steps to create accounts and reverses the state in case of an
// execution error or failed value transfer.
//
// CallCode differs from Call in the sense that it executes the given address'
// code with the caller as context.
func (evm *EVM) CallCode(caller ContractRef, addr common.Address, input []byte, gas uint64, value *big.Int, thetaValue *big.Int) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}

	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}
	// Fail if we're trying to transfer more than the available balance
	if !CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, gas, ErrInsufficientBalance
	}

	blockHeight := evm.StateDB.GetBlockHeight()
	if SupportWrappedTheta(blockHeight) && !CanTransferTheta(evm.StateDB, caller.Address(), thetaValue) {
		return nil, gas, ErrInsufficientThetaBlance
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)
	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, value, thetaValue, gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// DelegateCall executes the contract associated with the addr with the given input
// as parameters. It reverses the state in case of an execution error.
//
// DelegateCall differs from CallCode in the sense that it executes the given address'
// code with the caller as context and the caller is set to the caller of the caller.
func (evm *EVM) DelegateCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		snapshot = evm.StateDB.Snapshot()
		to       = AccountRef(caller.Address())
	)

	// Initialise a new contract and make initialise the delegate values
	contract := NewContract(caller, to, nil, nil, gas).AsDelegate()
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	ret, err = run(evm, contract, input, false)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

// StaticCall executes the contract associated with the addr with the given input
// as parameters while disallowing any modifications to the state during the call.
// Opcodes that attempt to perform such modifications will result in exceptions
// instead of performing the modifications.
func (evm *EVM) StaticCall(caller ContractRef, addr common.Address, input []byte, gas uint64) (ret []byte, leftOverGas uint64, err error) {
	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, gas, nil
	}
	// Fail if we're trying to execute above the call depth limit
	if evm.depth > int(params.CallCreateDepth) {
		return nil, gas, ErrDepth
	}

	var (
		to       = AccountRef(addr)
		snapshot = evm.StateDB.Snapshot()
	)
	// Initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, to, new(big.Int), new(big.Int), gas)
	contract.SetCallCode(&addr, evm.StateDB.GetCodeHash(addr), evm.StateDB.GetCode(addr))

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in Homestead this also counts for code storage gas errors.
	ret, err = run(evm, contract, input, true)
	if err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	return ret, contract.Gas, err
}

type codeAndHash struct {
	code []byte
	hash common.Hash
}

func (c *codeAndHash) Hash() common.Hash {
	if c.hash == (common.Hash{}) {
		c.hash = crypto.Keccak256Hash(c.code)
	}
	return c.hash
}

// create creates a new contract using code as deployment code.
func (evm *EVM) create(caller ContractRef, codeAndHash *codeAndHash, gas uint64, value *big.Int, thetaValue *big.Int, address common.Address) ([]byte, common.Address, uint64, error) {
	// Depth check execution. Fail if we're trying to execute above the
	// limit.
	if evm.depth > int(params.CallCreateDepth) {
		return nil, common.Address{}, gas, ErrDepth
	}
	if !CanTransfer(evm.StateDB, caller.Address(), value) {
		return nil, common.Address{}, gas, ErrInsufficientBalance
	}

	blockHeight := evm.StateDB.GetBlockHeight()
	if SupportThetaTransferInEVM(blockHeight) && !CanTransferTheta(evm.StateDB, caller.Address(), thetaValue) {
		return nil, common.Address{}, gas, ErrInsufficientThetaBlance
	}
	nonce := evm.StateDB.GetNonce(caller.Address())
	evm.StateDB.SetNonce(caller.Address(), nonce+1)

	// Ensure there's no existing contract already at the designated address
	contractHash := evm.StateDB.GetCodeHash(address)
	if evm.StateDB.GetNonce(address) != 0 || (contractHash != (common.Hash{}) && contractHash != types.EmptyCodeHash) {
		return nil, common.Address{}, 0, ErrContractAddressCollision
	}
	// Create a new account on the state
	snapshot := evm.StateDB.Snapshot()

	if !SupportThetaTransferInEVM(blockHeight) { // just for backward compatibility
		evm.StateDB.CreateAccount(address)
	} else { // should not wipe out the Theta/TFuel balance sent to the contract address prior to contract creation
		evm.StateDB.CreateAccountWithPreviousBalance(address)
	}
	Transfer(evm.StateDB, caller.Address(), address, value)

	if SupportThetaTransferInEVM(blockHeight) {
		TransferTheta(evm.StateDB, caller.Address(), address, thetaValue)
	}

	// initialise a new contract and set the code that is to be used by the
	// EVM. The contract is a scoped environment for this execution context
	// only.
	contract := NewContract(caller, AccountRef(address), value, thetaValue, gas)
	contract.SetCodeOptionalHash(&address, codeAndHash)

	if evm.vmConfig.NoRecursion && evm.depth > 0 {
		return nil, address, gas, nil
	}

	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureStart(caller.Address(), address, true, codeAndHash.code, gas, value)
	}
	start := time.Now()

	ret, err := run(evm, contract, nil, false)

	// check whether the max code size has been exceeded
	maxCodeSizeExceeded := len(ret) > params.MaxCodeSize
	// if the contract creation ran successfully and no errors were returned
	// calculate the gas required to store the code. If the code could not
	// be stored due to not enough gas set an error and let it be handled
	// by the error checking condition below.
	if err == nil && !maxCodeSizeExceeded {
		createDataGas := uint64(len(ret)) * params.CreateDataGas
		if contract.UseGas(createDataGas) {
			evm.StateDB.SetCode(address, ret)
		} else {
			err = ErrCodeStoreOutOfGas
		}
	}

	// When an error was returned by the EVM or when setting the creation code
	// above we revert to the snapshot and consume any gas remaining. Additionally
	// when we're in homestead this also counts for code storage gas errors.
	if maxCodeSizeExceeded || err != nil {
		evm.StateDB.RevertToSnapshot(snapshot)
		if err != errExecutionReverted {
			contract.UseGas(contract.Gas)
		}
	}
	// Assign err if contract code size exceeds the max while the err is still empty.
	if maxCodeSizeExceeded && err == nil {
		err = errMaxCodeSizeExceeded
	}
	if evm.vmConfig.Debug && evm.depth == 0 {
		evm.vmConfig.Tracer.CaptureEnd(ret, gas-contract.Gas, time.Since(start), err)
	}
	return ret, address, contract.Gas, err

}

// Create creates a new contract using code as deployment code.
func (evm *EVM) Create(caller ContractRef, code []byte, gas uint64, value *big.Int, thetaValue *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	contractAddr = crypto.CreateAddress(caller.Address(), evm.StateDB.GetNonce(caller.Address()))
	return evm.create(caller, &codeAndHash{code: code}, gas, value, thetaValue, contractAddr)
}

// Create2 creates a new contract using code as deployment code.
//
// The different between Create2 with Create is Create2 uses sha3(0xff ++ msg.sender ++ salt ++ sha3(init_code))[12:]
// instead of the usual sender-and-nonce-hash as the address where the contract is initialized at.
func (evm *EVM) Create2(caller ContractRef, code []byte, gas uint64, endowment *big.Int, thetaEndowment *big.Int, salt *big.Int) (ret []byte, contractAddr common.Address, leftOverGas uint64, err error) {
	codeAndHash := &codeAndHash{code: code}
	contractAddr = crypto.CreateAddress2(caller.Address(), common.BigToHash(salt), codeAndHash.Hash().Bytes())
	return evm.create(caller, codeAndHash, gas, endowment, thetaEndowment, contractAddr)
}

// ChainConfig returns the environment's chain configuration
func (evm *EVM) ChainConfig() *params.ChainConfig { return evm.chainConfig }
