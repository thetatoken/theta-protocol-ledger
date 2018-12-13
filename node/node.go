package node

import (
	"context"
	"sync"

	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	dp "github.com/thetatoken/ukulele/dispatcher"
	ld "github.com/thetatoken/ukulele/ledger"
	mp "github.com/thetatoken/ukulele/mempool"
	"github.com/thetatoken/ukulele/netsync"
	"github.com/thetatoken/ukulele/p2p"
	"github.com/thetatoken/ukulele/rpc"
	"github.com/thetatoken/ukulele/store"
	"github.com/thetatoken/ukulele/store/database"
	"github.com/thetatoken/ukulele/store/kvstore"
)

type Node struct {
	Store            store.Store
	Chain            *blockchain.Chain
	Consensus        *consensus.ConsensusEngine
	ValidatorManager core.ValidatorManager
	SyncManager      *netsync.SyncManager
	Dispatcher       *dp.Dispatcher
	Ledger           core.Ledger
	Mempool          *mp.Mempool
	RPC              *rpc.ThetaRPCServer

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

type Params struct {
	ChainID    string
	PrivateKey *crypto.PrivateKey
	Root       *core.Block
	Validators *core.ValidatorSet
	Network    p2p.Network
	DB         database.Database
}

func NewNode(params *Params) *Node {
	store := kvstore.NewKVStore(params.DB)
	chain := blockchain.NewChain(params.ChainID, store, params.Root)
	validatorManager := consensus.NewFixedValidatorManager(params.Validators)
	dispatcher := dp.NewDispatcher(params.Network)
	consensus := consensus.NewConsensusEngine(params.PrivateKey, store, chain, dispatcher, validatorManager)
	syncMgr := netsync.NewSyncManager(chain, consensus, params.Network, dispatcher, consensus)
	mempool := mp.CreateMempool(dispatcher)
	ledger := ld.NewLedger(params.ChainID, params.DB, consensus, validatorManager, mempool)
	consensus.SetLedger(ledger)
	mempool.SetLedger(ledger)
	txMsgHandler := mp.CreateMempoolMessageHandler(mempool)
	params.Network.RegisterMessageHandler(txMsgHandler)

	node := &Node{
		Store:            store,
		Chain:            chain,
		Consensus:        consensus,
		ValidatorManager: validatorManager,
		SyncManager:      syncMgr,
		Dispatcher:       dispatcher,
		Ledger:           ledger,
		Mempool:          mempool,
	}

	if viper.GetBool(common.CfgRPCEnabled) {
		node.RPC = rpc.NewThetaRPCServer(mempool, ledger, chain, consensus)
	}

	return node
}

// Start starts sub components and kick off the main loop.
func (n *Node) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	n.ctx = c
	n.cancel = cancel

	n.Consensus.Start(n.ctx)
	n.SyncManager.Start(n.ctx)
	n.Dispatcher.Start(n.ctx)
	n.Mempool.Start(n.ctx)

	if viper.GetBool(common.CfgRPCEnabled) {
		n.RPC.Start(n.ctx)
	}
}

// Stop notifies all sub components to stop without blocking.
func (n *Node) Stop() {
	n.cancel()
}

// Wait blocks until all sub components stop.
func (n *Node) Wait() {
	n.Consensus.Wait()
	n.SyncManager.Wait()
	if n.RPC != nil {
		n.RPC.Wait()
	}
}
