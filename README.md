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

## Run unit tests
Make sure `jq` is installed:
```
brew install jq
```
Run unit tests:
```
make test_unit
```
## Run a local private net
Use the following commands to launch a private net with a single validator node.
```
cd $UKULELE_HOME
cp -r ./integration/testnet ../testnet
ukulele start --config=../testnet/node2 | tee ../node2.log
```
Send Theta token between addresses by executing the following commands in another terminal. When the prompt asks for password, simply enter `qwertyuiop`
```
banjo tx send --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --gamma=900000 --seq=1
```
The balance of an address can be queried with the following command
```
banjo query account --address=2E833968E5bB786Ae419c4d13189fB081Cc43bab
```



