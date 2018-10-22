# Theta Ledger Protocol

The Theta Ledger is a decentralized ledger designed for the video streaming industry. It powers the Theta token economy which incentives end users to share their redundant bandwidth and storage resources, and encourage them to engage more actively with video platforms and content creators. The ledger employs a novel multi-level BFT consensus engine, which supports high transaction throughput, fast block confirmation, and allows mass participation in the consensus process. Off-chain payment support is built directly into the ledger through the resource-oriented micropayment pool, which is designed specifically to achieve the “pay-per-byte” granularity for streaming use cases. Moreover, the ledger storage system leverages the microservice architecture and reference counting based history pruning techniques, and is thus able to adapt to different computing environments, ranging from high-end data center server clusters to commodity PCs and laptops. For more technical details, please refer to our technical whitepaper.

## Setup

Install Go and set environment variables `GOPATH` and `PATH`.

Clone this repo into your GOPATH. The path should look like this: `$GOPATH/src/github.com/thetatoken/ukulele`

```
git clone git@github.com:thetatoken/theta-protocol-ledger-2.git $GOPATH/src/github.com/thetatoken/ukulele
```

Install [glide](https://github.com/Masterminds/glide). Then in code base, do this to download all dependencies:

```
cd $GOPATH/src/github.com/thetatoken/ukulele
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
