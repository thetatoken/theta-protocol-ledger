# Theta Ledger Protocol

The Theta Ledger is a Proof-of-Stake decentralized ledger designed for the video streaming industry. It powers the Theta token economy which incentives end users to share their redundant bandwidth and storage resources, and encourage them to engage more actively with video platforms and content creators. The ledger employs a novel [multi-level BFT consensus engine](docs/multi-level-bft-tech-report.pdf), which supports high transaction throughput, fast block confirmation, and allows mass participation in the consensus process. Off-chain payment support is built directly into the ledger through the resource-oriented micropayment pool, which is designed specifically to achieve the “pay-per-byte” granularity for streaming use cases. Moreover, the ledger storage system leverages the microservice architecture and reference counting based history pruning techniques, and is thus able to adapt to different computing environments, ranging from high-end data center server clusters to commodity PCs and laptops. For more technical details, please refer to our [technical whitepaper](docs/theta-technical-whitepaper.pdf) and the [multi-level BFT technical report](docs/multi-level-bft-tech-report.pdf).

## Setup

Install Go and set environment variables `GOPATH` and `PATH`.

Clone this repo into your GOPATH. The path should look like this: `$GOPATH/src/github.com/thetatoken/ukulele`

```
git clone git@github.com:thetatoken/theta-protocol-ledger-2.git $GOPATH/src/github.com/thetatoken/ukulele
```

Install [glide](https://github.com/Masterminds/glide). Then in code base, do this to download all dependencies:

```
export UKULELE_HOME=$GOPATH/src/github.com/thetatoken/ukulele
cd $UKULELE_HOME
make get_vendor_deps
```

## Build
This should build the binaries and copy them into your `$GOPATH/bin`:
```
make build install
```

## Run Unit Tests
Make sure `jq` is installed:
```
brew install jq
```
Run unit tests:
```
make test_unit
```
## Run a Local Private Net
Use the following commands to launch a private net with a single validator node.
```
cd $UKULELE_HOME
cp -r ./integration/testnet ../testnet
ukulele start --config=../testnet/node2
```
Send Theta token between addresses by executing the following commands in another terminal. When the prompt asks for password, simply enter `qwertyuiop`
```
banjo tx send --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --gamma=20 --seq=1
```
The balance of an address can be queried with the following command
```
banjo query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab
```
## Deploy and Execute Smart Contracts
The Theta blockchain includes the [Ethereum Virtual Machine (EVM)](https://github.com/ethereum/wiki/wiki/Ethereum-Virtual-Machine-(EVM)-Awesome-List) which supports Turing-Complete smart contracts. The examples below demonstrate how to deploy the bytecode of a smart contract, and then execute the smart contract. For each step, we always perform a __dry run__ first with the `banjo call` command, which simulates the smart contract execution locally. For the actually deployment and execution, we use the `banjo tx` command instead.

First, let us udo a dry run for the smart contract deployment. The `data` parameter carries the bytecode of the smart contract to be deployed. The bytecode can be generated from [Solidity source code](https://solidity.readthedocs.io/en/v0.4.25/) using any EVM compatible compiler.
```
banjo call smart_contract --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=1680 --gas_price=3 --gas_limit=50000 --data=600a600c600039600a6000f3600360135360016013f3
```
This call should return a json similar to the one shown below. The `contract_address` parameter gives the address of the smart contract when it is actually deployed on to the blockchain.
```
{
    "contract_address": "0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337",
    "gas_used": 2024,
    "vm_error": null,
    "vm_return": "600360135360016013f3"
}
```
Now, let us deploy the contract with the  `banjo tx` command. Again, when the prompt asks for password, simply enter `qwertyuiop`
```
banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=1680 --gas_price=3 --gas_limit=50000 --data=600a600c600039600a6000f3600360135360016013f3 --seq=2
```
Wait 5 to 10 seconds for the transaction to be included in the blockchain. Then we can use the following query command to confirmed that the smart contract has been deployed, where the account address is the `contract_address` returned by the deployment dry run.
```
banjo query account --address=0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337
```
Now, we can do another dry run with `banjo call` to check the result and its gas usage if we were to call the smart contract. Note that the smart contract address is passed to the command with the `to` parameter.
```
banjo call smart_contract --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337 --gas_price=3 --gas_limit=50000
```
The call should return something like this
```
{
    "contract_address": "0x5c3159ddd2fe0f9862bc7b7d60c1875fa8f81337",
    "gas_used": 18,
    "vm_error": null,
    "vm_return": "03"
}
```
Finally, we can execute the smart contract with the following `banjo tx` command
```
banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x7ad6cea2bc3162e30a3c98d84f821b3233c22647 --gas_price=3 --gas_limit=50000 --seq=3
```
You might have noticed that both the smart contract deployment and execution use the `banjo tx smart_contract` command with similar parameters. The only difference is that the deployment command does not have the `to` parameter, while in the execution command, the `to` parameter is set to the smart contract address.


