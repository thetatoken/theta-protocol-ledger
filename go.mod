module github.com/thetatoken/theta

require (
	github.com/aerospike/aerospike-client-go v1.36.0
	github.com/bgentry/speakeasy v0.1.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.6.2
	github.com/golang/protobuf v1.5.2
	github.com/golang/snappy v0.0.0-20180518054509-2e65f85255db
	github.com/gorilla/mux v1.7.3
	github.com/hashicorp/golang-lru v0.5.4
	github.com/herumi/bls-eth-go-binary v0.0.0-20200107021104-147ed25f233e
	github.com/huin/goupnp v1.0.2
	github.com/ipfs/go-datastore v0.5.0
	github.com/jackpal/gateway v1.0.5
	github.com/jackpal/go-nat-pmp v1.0.2
	github.com/karalabe/hid v0.0.0-20180420081245-2b4488a37358
	github.com/koron/go-ssdp v0.0.2
	github.com/libp2p/go-libp2p v0.18.0
	github.com/libp2p/go-libp2p-connmgr v0.1.1
	github.com/libp2p/go-libp2p-core v0.14.0
	github.com/libp2p/go-libp2p-crypto v0.1.0
	github.com/libp2p/go-libp2p-kad-dht v0.2.0
	github.com/libp2p/go-libp2p-peerstore v0.6.0
	github.com/libp2p/go-libp2p-pubsub v0.1.1
	github.com/mattn/go-isatty v0.0.14
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mongodb/mongo-go-driver v0.0.17
	github.com/multiformats/go-multiaddr v0.5.0
	github.com/pborman/uuid v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday v2.0.0+incompatible // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.5.0
	github.com/stretchr/testify v1.7.0
	github.com/syndtr/goleveldb v1.0.0
	github.com/thetatoken/theta/common v0.0.0
	github.com/thetatoken/theta/rpc/lib/rpc-codec/jsonrpc2 v0.0.0
	github.com/tidwall/pretty v1.0.0 // indirect
	github.com/xdg/scram v0.0.0-20180814205039-7eeb5667e42c // indirect
	github.com/xdg/stringprep v0.0.0-20180714160509-73f8eece6fdc // indirect
	github.com/ybbus/jsonrpc v1.1.1
	github.com/yuin/gopher-lua v0.0.0-20180827083657-b942cacc89fe // indirect
	golang.org/x/crypto v0.0.0-20210813211128-0a44fdfbc16e
	golang.org/x/net v0.0.0-20210813160813-60bc85c4be6d
	golang.org/x/sys v0.0.0-20220412071739-889880a91fd5
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/mgo.v2 v2.0.0-20180705113604-9856a29383ce
)

replace github.com/thetatoken/theta/rpc/lib/rpc-codec/jsonrpc2 v0.0.0 => ./rpc/lib/rpc-codec/jsonrpc2/

replace github.com/thetatoken/theta/common v0.0.0 => ./common

go 1.13
