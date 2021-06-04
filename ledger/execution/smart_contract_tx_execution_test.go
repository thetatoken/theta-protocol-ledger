package execution

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/ledger/vm"
)

func TestSimpleSmartContractDeploymentAndExecution(t *testing.T) {
	assert := assert.New(t)
	et, privAccounts := setupForSmartContract(assert, 2)
	et.fastforwardBy(1000)

	deployerPrivAcc := &privAccounts[0]
	callerPrivAcc := &privAccounts[1]

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
	deploymentCode, _ := hex.DecodeString("600a600c600039600a6000f3600360135360016013f3")

	// ASM:
	// push 0x3
	// push 0x13
	// mstore8
	// push 0x1
	// push 0x13
	// return
	smartContractCode, _ := hex.DecodeString("600360135360016013f3")

	// Step 1. Deploy a smart contract
	valueAmount := int64(9723)
	gasLimit := uint64(90000)
	contractAddr := deploySmartContract(et, deployerPrivAcc, valueAmount, gasLimit, deploymentCode, smartContractCode, 1, assert)

	// Step 2. Execute the smart contact
	gasLimit = uint64(30000)
	data := common.Bytes(nil)
	vmRet, vmErr, _ := callSmartContract(et, contractAddr, callerPrivAcc, gasLimit, data, 1, assert)
	assert.Nil(vmErr)
	assert.Equal(common.Bytes{0x3}, vmRet)
	executeSmartContract(et, contractAddr, callerPrivAcc, gasLimit, data, 1, assert)
}

// ------------ Solidity Source Code of the Contract under Test ------------ //
//
// pragma solidity ^0.4.18;
//
// library SafeMath {
//     function sub(uint a, uint b) internal pure returns (uint) {
//         assert(b <= a);
//         return a - b;
//     }
//
//     function add(uint a, uint b) internal pure returns (uint) {
//         uint c = a + b;
//         assert(c >= a);
//         return c;
//     }
// }
//
// contract TestCustomToken {
//     using SafeMath for uint;
//     mapping (address => uint) balances;
//     address public constant ADMIN = 0x4d4ce78b09F8A06C0d3063a315dC9c011F6e876E;
//
//     function mint() public {
//         require(msg.sender == ADMIN);
//         balances[ADMIN] = balances[ADMIN].add(10000);
//     }
//
//     function balanceOf(address _owner) public constant returns (uint balance) {
//         return balances[_owner];
//     }
//
//     function transfer(address _to, uint _value) public returns (bool success) {
//         require(balances[msg.sender] >= _value && _value > 0);
//         balances[msg.sender] = balances[msg.sender].sub(_value);
//         balances[_to] = balances[_to].add(_value);
//         return true;
//     }
// }
func TestCustomTokenSmartContract(t *testing.T) {
	assert := assert.New(t)

	//
	// Step 0. Preparation
	//
	et, privAccounts := setupForSmartContract(assert, 4)
	et.fastforwardBy(1000)

	adminPrivAcc := &privAccounts[0]
	deployerPrivAcc := &privAccounts[1]
	user1PrivAcc := &privAccounts[2]
	user2PrivAcc := &privAccounts[3]

	adminAddr := adminPrivAcc.Address
	assert.Equal(common.HexToAddress("0x4d4ce78b09F8A06C0d3063a315dC9c011F6e876E"), adminAddr)
	deployerAddr := deployerPrivAcc.Address
	user1Addr := user1PrivAcc.Address
	user2Addr := user2PrivAcc.Address
	log.Infof("Admin    Address: %v", adminAddr)
	log.Infof("Deployer Address: %v", deployerAddr)
	log.Infof("User1    Address: %v", user1Addr)
	log.Infof("User2    Address: %v", user2Addr)

	var cbc contractByteCode
	err := loadJSONTest("testdata/custom_token_transfer.json", &cbc)
	assert.Nil(err)
	deploymentCode, err := hex.DecodeString(cbc.DeploymentCode)
	assert.Nil(err)
	smartContractCode, err := hex.DecodeString(cbc.Code)
	assert.Nil(err)

	mintTokenData, _ := hex.DecodeString("1249c58b") // signature of the mint() API
	getQueryTokenBalanceData := func(addr common.Address) common.Bytes {
		queryToken := "70a08231000000000000000000000000" // signature of the balanceOf() API
		dataStr := queryToken + addr.String()[2:]
		log.Infof("Token balance query data: %v", dataStr)
		data, _ := hex.DecodeString(dataStr)
		return data
	}
	getTransferTokenData := func(to common.Address, amount uint64) common.Bytes {
		transferToken := "a9059cbb000000000000000000000000" // signature of the transfer() API
		dataStr := transferToken + to.String()[2:] + fmt.Sprintf("%064x", amount)
		log.Infof("Token transfer transaction data: %v", dataStr)
		data, _ := hex.DecodeString(dataStr)
		return data
	}
	vmRevertErr := fmt.Errorf("evm: execution reverted")
	zeroBalanceStr := fmt.Sprintf("%064x", 0)
	oneTimeMintingAmount := uint64(10000)

	//
	// Step 1. Deploy the smart contract
	//
	valueAmount := int64(0)
	gasLimit := uint64(500000)
	contractAddr := deploySmartContract(et, deployerPrivAcc, valueAmount, gasLimit, deploymentCode, smartContractCode, 1, assert)

	//
	// Step 2. Contract access permission test via token minting
	//

	// User1 tries to mint tokens, should not succeed
	gasLimit = uint64(90000)
	vmRet, vmErr, _ := callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, mintTokenData, 1, assert)
	assert.Equal(vmRevertErr, vmErr)
	vmRet = executeSmartContract(et, contractAddr, user1PrivAcc, gasLimit, mintTokenData, 1, assert)

	queryUser1TokenBalanceData := getQueryTokenBalanceData(user1Addr)
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 2, assert)
	assert.Nil(vmErr)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	queryUser2TokenBalanceData := getQueryTokenBalanceData(user2Addr)
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Nil(vmErr)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	queryAdminTokenBalanceData := getQueryTokenBalanceData(adminAddr)
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, adminPrivAcc, gasLimit, queryAdminTokenBalanceData, 1, assert)
	assert.Nil(vmErr)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))

	// Admin mints tokens, only the Admin account possesses the minted tokens
	gasLimit = uint64(60000)
	vmRet = executeSmartContract(et, contractAddr, adminPrivAcc, gasLimit, mintTokenData, 1, assert)
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 2, assert)
	assert.Nil(vmErr)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Nil(vmErr)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, adminPrivAcc, gasLimit, queryAdminTokenBalanceData, 2, assert)
	assert.Nil(vmErr)
	assert.Equal(fmt.Sprintf("%064x", oneTimeMintingAmount), hex.EncodeToString(vmRet))

	//
	// Step 3. Test token transfer among addresses
	//

	// transfer from User1 to User2, should not succeed, since User1 has zero token balance
	gasLimit = uint64(60000)
	transferAmount1 := uint64(1000)
	transferToUser2Data := getTransferTokenData(user2Addr, transferAmount1)
	vmRet = executeSmartContract(et, contractAddr, user1PrivAcc, gasLimit, transferToUser2Data, 2, assert)

	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 3, assert)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))

	// transfer from Admin to User2
	gasLimit = uint64(60000)
	expectedAdminCustomTokenBalance := oneTimeMintingAmount - transferAmount1
	expectedUser2CustomeTokenBalance := transferAmount1
	vmRet = executeSmartContract(et, contractAddr, adminPrivAcc, gasLimit, transferToUser2Data, 2, assert)

	vmRet, vmErr, _ = callSmartContract(et, contractAddr, adminPrivAcc, gasLimit, queryAdminTokenBalanceData, 3, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedAdminCustomTokenBalance), hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 3, assert)
	assert.Equal(zeroBalanceStr, hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedUser2CustomeTokenBalance), hex.EncodeToString(vmRet))

	// now User2 has some tokens, he can transfer to User1
	transferAmount2 := uint64(100)
	expectedUser2CustomeTokenBalance = transferAmount1 - transferAmount2
	expectedUser1CustomeTokenBalance := transferAmount2
	transferToUser1Data := getTransferTokenData(user1Addr, transferAmount2)
	vmRet = executeSmartContract(et, contractAddr, user2PrivAcc, gasLimit, transferToUser1Data, 1, assert)

	vmRet, vmErr, _ = callSmartContract(et, contractAddr, adminPrivAcc, gasLimit, queryAdminTokenBalanceData, 3, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedAdminCustomTokenBalance), hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 3, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedUser1CustomeTokenBalance), hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedUser2CustomeTokenBalance), hex.EncodeToString(vmRet))

	// User1 tries to transfer more than he has, should fail. Token Balance of the accounts should not change
	transferAmount3 := uint64(1000000)
	transferToUser2AgainData := getTransferTokenData(user2Addr, transferAmount3)
	vmRet = executeSmartContract(et, contractAddr, user1PrivAcc, gasLimit, transferToUser2AgainData, 3, assert)

	vmRet, vmErr, _ = callSmartContract(et, contractAddr, adminPrivAcc, gasLimit, queryAdminTokenBalanceData, 4, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedAdminCustomTokenBalance), hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user1PrivAcc, gasLimit, queryUser1TokenBalanceData, 3, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedUser1CustomeTokenBalance), hex.EncodeToString(vmRet))
	vmRet, vmErr, _ = callSmartContract(et, contractAddr, user2PrivAcc, gasLimit, queryUser2TokenBalanceData, 1, assert)
	assert.Equal(fmt.Sprintf("%064x", expectedUser2CustomeTokenBalance), hex.EncodeToString(vmRet))
}

// --------------------------- Test Untils --------------------------- //

func deploySmartContract(et *execTest, deployerPrivAcc *types.PrivAccount,
	valueAmount int64, gasLimit uint64, deploymentCode, smartContractCode common.Bytes,
	sequence uint64, assert *assert.Assertions) (contractAddr common.Address) {
	deployerAcc := deployerPrivAcc.Account
	deployerAddr := deployerAcc.Address
	gasPrice := types.MinimumGasPriceJune2021
	deploySCTx := &types.SmartContractTx{
		From: types.TxInput{
			Address:  deployerAddr,
			Coins:    types.NewCoins(0, valueAmount),
			Sequence: sequence,
		},
		GasLimit: gasLimit,
		GasPrice: new(big.Int).SetUint64(gasPrice),
		Data:     deploymentCode,
	}
	signBytes := deploySCTx.SignBytes(et.chainID)
	deploySCTx.From.Signature = deployerPrivAcc.Sign(signBytes)

	// Dry run to get the smart contract address when it is actually deployed
	parentBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    1,
			Timestamp: 1601599331,
		},
	}
	stateCopy, err := et.state().Delivered().Copy()
	assert.Nil(err)
	_, contractAddr, gasUsed, vmErr := vm.Execute(parentBlock, deploySCTx, stateCopy)
	assert.Nil(vmErr)
	log.Infof("[Deployment] gas used: %v", gasUsed)

	// The actual on-chain deplpoyment
	res := et.executor.getTxExecutor(deploySCTx).sanityCheck(et.chainID, et.state().Delivered(), deploySCTx)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(deploySCTx).process(et.chainID, et.state().Delivered(), deploySCTx)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	// Check if the smart contract code has actually been deployed on-chain
	retrievedCode := et.state().Delivered().GetCode(contractAddr)
	assert.True(bytes.Equal(smartContractCode, retrievedCode))

	// Check the amount of coins transferred to the smart contract
	retrievedSmartContractAccount := et.state().Delivered().GetAccount(contractAddr)
	assert.NotNil(retrievedSmartContractAccount)
	expectedTransferredValue := types.NewCoins(0, valueAmount)
	assert.True(expectedTransferredValue.IsEqual(retrievedSmartContractAccount.Balance))
	log.Infof("[Deployment] expected transferred value: %v, actual transferred value: %v",
		expectedTransferredValue, retrievedSmartContractAccount.Balance)

	// Check the deployment gas fee
	retrievedDeployerAcc := et.state().Delivered().GetAccount(deployerAddr)
	deploymentFee := types.NewCoins(0, int64(gasUsed)*int64(gasPrice))
	expectedTotalDeploymentCost := expectedTransferredValue.Plus(deploymentFee)
	deploymentAccBalanceReduction := deployerAcc.Balance.Minus(retrievedDeployerAcc.Balance)
	assert.Equal(expectedTotalDeploymentCost, deploymentAccBalanceReduction)
	log.Infof("[Deployment] expected gas cost: %v, actual gas cost: %v",
		expectedTotalDeploymentCost, deploymentAccBalanceReduction)

	return contractAddr
}

func callSmartContract(et *execTest, contractAddr common.Address, callerPrivAcc *types.PrivAccount,
	gasLimit uint64, data common.Bytes, sequence uint64, assert *assert.Assertions) (vmRet common.Bytes, vmErr error, gasUsed uint64) {
	callerAcc := callerPrivAcc.Account
	callerAddr := callerAcc.Address
	gasPrice := types.MinimumGasPriceJune2021
	callSCTX := &types.SmartContractTx{
		From: types.TxInput{
			Address:  callerAddr,
			Sequence: sequence,
		},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: gasLimit,
		GasPrice: new(big.Int).SetUint64(gasPrice),
		Data:     data,
	}
	signBytes := callSCTX.SignBytes(et.chainID)
	callSCTX.From.Signature = callerPrivAcc.Sign(signBytes)

	// Dry run to call the contract
	stateCopy, err := et.state().Delivered().Copy()
	assert.Nil(err)

	parentBlock := &core.Block{
		BlockHeader: &core.BlockHeader{
			Height:    1,
			Timestamp: 1601599331,
		},
	}
	vmRet, execContractAddr, gasUsed, vmErr := vm.Execute(callSCTX, stateCopy)
	assert.Equal(contractAddr, execContractAddr)
	log.Infof("[Call      ] gas used: %v", gasUsed)

	return vmRet, vmErr, gasUsed
}

func executeSmartContract(et *execTest, contractAddr common.Address, callerPrivAcc *types.PrivAccount,
	gasLimit uint64, data common.Bytes, sequence uint64, assert *assert.Assertions) (vmRet common.Bytes) {

	_, _, gasUsed := callSmartContract(et, contractAddr, callerPrivAcc, gasLimit, data, sequence, assert)

	callerAcc := callerPrivAcc.Account
	callerAddr := callerAcc.Address
	retrievedCallerAccBeforeExec := et.state().Delivered().GetAccount(callerAddr)
	gasPrice := types.MinimumGasPriceJune2021
	execSCTX := &types.SmartContractTx{
		From: types.TxInput{
			Address:  callerAddr,
			Sequence: sequence,
		},
		To:       types.TxOutput{Address: contractAddr},
		GasLimit: gasLimit,
		GasPrice: new(big.Int).SetUint64(gasPrice),
		Data:     data,
	}
	signBytes := execSCTX.SignBytes(et.chainID)
	execSCTX.From.Signature = callerPrivAcc.Sign(signBytes)

	// Execute the on-chain smart contract
	res := et.executor.getTxExecutor(execSCTX).sanityCheck(et.chainID, et.state().Delivered(), execSCTX)
	assert.True(res.IsOK(), res.Message)
	_, res = et.executor.getTxExecutor(execSCTX).process(et.chainID, et.state().Delivered(), execSCTX)
	assert.True(res.IsOK(), res.Message)

	et.state().Commit()

	// Check the smart contract execution gas fee
	retrievedCallerAccAfterExec := et.state().Delivered().GetAccount(callerAddr)
	expectedSCExecGasFee := types.NewCoins(0, int64(gasUsed)*int64(gasPrice))
	callerAccBalanceReduction := retrievedCallerAccBeforeExec.Balance.Minus(retrievedCallerAccAfterExec.Balance)
	assert.Equal(expectedSCExecGasFee, callerAccBalanceReduction)
	log.Infof("[Execution ] expected gas cost: %v, actual gas cost: %v",
		expectedSCExecGasFee, callerAccBalanceReduction)

	return vmRet
}
