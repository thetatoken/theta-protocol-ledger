# Theta Blockchain Ledger Protocol

The Theta Blockchain Ledger is a Proof-of-Stake decentralized ledger designed for the video streaming industry. It powers the Theta token economy which incentives end users to share their redundant bandwidth and storage resources, and encourage them to engage more actively with video platforms and content creators. The ledger employs a novel [multi-level BFT consensus engine](docs/multi-level-bft-tech-report.pdf), which supports high transaction throughput, fast block confirmation, and allows mass participation in the consensus process. Off-chain payment support is built directly into the ledger through the resource-oriented micropayment pool, which is designed specifically to achieve the “pay-per-byte” granularity for streaming use cases. Moreover, the ledger storage system leverages the microservice architecture and reference counting based history pruning techniques, and is thus able to adapt to different computing environments, ranging from high-end data center server clusters to commodity PCs and laptops. The ledger also supports Turing-Complete smart contracts, which enables rich user experiences for DApps built on top of the Theta Ledger. For more technical details, please refer to our [technical whitepaper](docs/theta-technical-whitepaper.pdf) and the [multi-level BFT technical report](docs/multi-level-bft-tech-report.pdf).

## Table of Contents
- [Setup](#setup)
- [Build and Install](#build-and-install)
- [Run Unit Tests](#run-unit-tests)
- [Launch a Local Private Net](#launch-a-local-private-net)
- [Deploy and Execute Smart Contracts](#deploy-and-execute-smart-contracts)
- [Off-Chain Micropayment Support](#off-chain-micropayment-support)

## Setup
Install Go and set environment variables `GOPATH` and `PATH` following the [official instructions](https://golang.org/doc/install)

Clone this repo into your `$GOPATH`. The path should look like this: `$GOPATH/src/github.com/thetatoken/ukulele`

```
git clone git@github.com:thetatoken/theta-protocol-ledger.git $GOPATH/src/github.com/thetatoken/ukulele
```

Install [glide](https://github.com/Masterminds/glide). Then execute the following commands to download all dependencies:

```
export UKULELE_HOME=$GOPATH/src/github.com/thetatoken/ukulele
cd $UKULELE_HOME
make get_vendor_deps
```
Also make sure `jq` is installed to run the unit tests. On Mac OS X, run the following command
```
brew install jq
```
On Ubuntu Linux, install `jq` with
```
sudo apt-get install jq
```

## Build and Install
This should build the binaries and copy them into your `$GOPATH/bin`. Two binaries `ukulele` and `banjo` are generated. `ukulele` can be regarded as the launcher of the Theta Ledger node, and `banjo` is a wallet with command line tools to interact with the ledger. 
```
make build install
```

## Run Unit Tests
Run unit tests with the command below
```
make test_unit
```

## Launch a Local Private Net
Open a terminal, and use the following commands to launch a private net with a single validator node.
```
cd $UKULELE_HOME
cp -r ./integration/testnet ../testnet
ukulele start --config=../testnet/node2
```
In another terminal, we can use the `banjo` command line tool to send Theta tokens from one address to another by executing the following command. When the prompt asks for password, simply enter `qwertyuiop`
```
banjo tx send --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --gamma=20 --seq=1
```
The balance of an address can be retrieved with the following query command
```
banjo query account --address=9F1233798E905E173560071255140b4A8aBd3Ec6
```

## Deploy and Execute Smart Contracts
The Theta Ledger provides a Turing-Complete smart contract runtime environment compatible with the [Ethereum Virtual Machine](https://github.com/ethereum/wiki/wiki/Ethereum-Virtual-Machine-(EVM)-Awesome-List) (EVM). [Solidity](https://solidity.readthedocs.io/) based Ethereum smart contracts can be ported to the Theta Ledger with little effort. The example below demonstrates how to deploy and execute an example smart contract `SquareCalculator` on the local private net we just launched.

```
pragma solidity ^0.4.18;

contract SquareCalculator {
    uint public value;
    
    function SetValue(uint val) public {
        value = val;
    }
    
    function CalculateSquare() constant public returns (uint) {
        uint sqr = value * value;
        assert(sqr / value == value); // overflow protection
        return sqr;
    }
}
```
Using any Solidity compiler, such as [Remix](https://remix.ethereum.org), we can generate the __deployment bytecode__ of the smart contract as shown below.
```
608060405234801561001057600080fd5b50610148806100206000396000f300608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633fa4f2451461005c578063b5a0241a14610087578063ed8b0706146100b2575b600080fd5b34801561006857600080fd5b506100716100df565b6040518082815260200191505060405180910390f35b34801561009357600080fd5b5061009c6100e5565b6040518082815260200191505060405180910390f35b3480156100be57600080fd5b506100dd60048036038101908080359060200190929190505050610112565b005b60005481565b6000806000546000540290506000546000548281151561010157fe5b0414151561010b57fe5b8091505090565b80600081905550505600a165627a7a72305820459c07c1668e919ca760d663b8df04e80634c53ebd49393dca83e81c58ae2a660029
```
Now let us deploy the bytecode on the Theta Ledger, and then use it to calculate squares. Note that for each of steps below, we may perform a __dry run__ first with the `banjo call` command, which simulates the smart contract execution locally. For the actually deployment and execution, we use the `banjo tx` command instead.

First, let us do a dry run for the smart contract deployment. The `data` parameter carries the deployment bytecode of the smart contract as provided above.
```
banjo call smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=0 --gas_price=1000000000wei --gas_limit=200000 --data=608060405234801561001057600080fd5b50610148806100206000396000f300608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633fa4f2451461005c578063b5a0241a14610087578063ed8b0706146100b2575b600080fd5b34801561006857600080fd5b506100716100df565b6040518082815260200191505060405180910390f35b34801561009357600080fd5b5061009c6100e5565b6040518082815260200191505060405180910390f35b3480156100be57600080fd5b506100dd60048036038101908080359060200190929190505050610112565b005b60005481565b6000806000546000540290506000546000548281151561010157fe5b0414151561010b57fe5b8091505090565b80600081905550505600a165627a7a72305820459c07c1668e919ca760d663b8df04e80634c53ebd49393dca83e81c58ae2a660029
```
This call should return a json similar to the one shown below. The `contract_address` parameter gives the address of the smart contract when it is actually deployed on to the blockchain. The `gas_used` field is the amount of gas to be consumed if we deploy the smart contract.
```
{
    "contract_address": "0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337",
    "gas_used": 139293,
    "vm_error": "",
    "vm_return": "608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633fa4f2451461005c578063b5a0241a14610087578063ed8b0706146100b2575b600080fd5b34801561006857600080fd5b506100716100df565b6040518082815260200191505060405180910390f35b34801561009357600080fd5b5061009c6100e5565b6040518082815260200191505060405180910390f35b3480156100be57600080fd5b506100dd60048036038101908080359060200190929190505050610112565b005b60005481565b6000806000546000540290506000546000548281151561010157fe5b0414151561010b57fe5b8091505090565b80600081905550505600a165627a7a72305820459c07c1668e919ca760d663b8df04e80634c53ebd49393dca83e81c58ae2a660029"
}
```
Now, let us deploy the contract with the following command. Again, when the prompt asks for password, simply enter `qwertyuiop`
```
banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=0 --gas_price=1000000000wei --gas_limit=200000 --data=608060405234801561001057600080fd5b50610148806100206000396000f300608060405260043610610057576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff1680633fa4f2451461005c578063b5a0241a14610087578063ed8b0706146100b2575b600080fd5b34801561006857600080fd5b506100716100df565b6040518082815260200191505060405180910390f35b34801561009357600080fd5b5061009c6100e5565b6040518082815260200191505060405180910390f35b3480156100be57600080fd5b506100dd60048036038101908080359060200190929190505050610112565b005b60005481565b6000806000546000540290506000546000548281151561010157fe5b0414151561010b57fe5b8091505090565b80600081905550505600a165627a7a72305820459c07c1668e919ca760d663b8df04e80634c53ebd49393dca83e81c58ae2a660029 --seq=2
```
Wait for a few seconds for the transaction to be included in the blockchain. Then we can use the following query command to confirmed that the smart contract has been deployed, where the account address is the `contract_address` returned by the deployment dry run.
```
banjo query account --address=0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337
```
Now, let us call the `SetValue()` function of the deployed smart contract with the following `banjo tx` command. Note that the smart contract address is passed to the command with the `to` parameter. And the `data` parameter is the concatenation of `ed8b0706`, the signature of the function `SetValue()`, and an integer `0x3` for which we want to calculate the square.  
```
banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337 --gas_price=1000000000wei --gas_limit=50000 --data=ed8b07060000000000000000000000000000000000000000000000000000000000000003 --seq=3
```
Again, wait for a couple seconds for the transaction to be included in the blockchain, and then we can query the square result with the following command, where the `data` parameter `b5a0241a` is the signature of the `CalculateSquare()` function.
```
banjo call smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337 --gas_price=1000000000wei --gas_limit=50000 --data=b5a0241a
```
The `vm_return` field in the returned json should be `0000000000000000000000000000000000000000000000000000000000000009`, which is simply the square of `0x3`, the value we set previously.

You might have noticed that both the smart contract deployment and execution use the `banjo tx smart_contract` command with similar parameters. The only difference is that the deployment command does not have the `to` parameter, while in the execution command, the `to` parameter is set to the smart contract address.

## Off-Chain Micropayment Support
In order to handle the sheer amount of micropayments for the bandwidth sharing reward, the Theta Ledger provides native support for off-chain payment through the [resource oriented micropayment pool](https://medium.com/theta-network/building-the-theta-protocol-part-iv-d7cce583aad1) concept. The micropayment pool allows a sender to pay to multiple recipients with off-chain transactions without the sender being able to double spend.

Below is an example. To get started, the sender creates a resource oriented micropayment pool for a live video stream with resource_id `rid1000001` by reserving some Gamma tokens for 1002 blocktimes. She can use this micropayment pool to pay multiple relay nodes that provides the desired video stream.
```
banjo tx reserve --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --fund=100 --collateral=101 --duration=1002 --resource_ids=rid1000001 --seq=4
```
After this transaction has been processed. We can query the `from` account to confirm the creation of the micropayment pool.
```
banjo query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab
```
The return should look like the json below. As we can see, 100 Gamma (= 100000000000000000000 GammaWei) were reserved for the off-chain payment with 101 Gamma collateral for resourceID `rid1000001`. If the sender overspends the reserved fund, her collateral will be entirely slashed.
```
{
    "address": "2E833968E5bB786Ae419c4d13189fB081Cc43bab",
    ...
    "reserved_funds": [
        {
            "collateral": {
                "gammawei": 101000000000000000000,
                "thetawei": 0
            },
            "end_block_height": 1588,
            "initial_fund": {
                "gammawei": 100000000000000000000,
                "thetawei": 0
            },
            "reserve_sequence": 4,
            "resource_ids": [
                "rid1000001"
            ],
            "transfer_records": [],
            "used_fund": {
                "gammawei": 0,
                "thetawei": 0
            }
        }
    ],
    ...
}
```
From the reserved fund, the sender can send tokens to multiple parties with a special off-chain [Service Payment Transaction](https://github.com/thetatoken/theta-protocol-ledger/blob/ed3d616eca7e3de2c19f63351716aba7547a1e4c/ledger/types/tx.go#L321). Before the reserved fund expires (1002 blocktimes), whenever a recipient wants to receive the tokens, he simply signs the last received service payment transaction, and [submits the signed raw transaction to the Ledger node](https://github.com/thetatoken/theta-protocol-ledger/blob/ed3d616eca7e3de2c19f63351716aba7547a1e4c/rpc/tx.go#L11). A sender might send the recipient multiple off-chain transactions before the recipient signs and submits the last transaction to receive the full amount. This mechanism achieves the "pay-per-byte" granularity, and yet could reduce the amount of on-chain transactions by several orders of magnitude. For more details, please refer to the "Off-Chain Micropayment Support" section of our [technical whitepaper](docs/theta-technical-whitepaper.pdf).
