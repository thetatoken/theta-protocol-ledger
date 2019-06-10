package node

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/metrics"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	"github.com/thetatoken/theta/crypto"
	dp "github.com/thetatoken/theta/dispatcher"
	ld "github.com/thetatoken/theta/ledger"
	mp "github.com/thetatoken/theta/mempool"
	"github.com/thetatoken/theta/netsync"
	"github.com/thetatoken/theta/p2p"
	"github.com/thetatoken/theta/rpc"
	"github.com/thetatoken/theta/snapshot"
	"github.com/thetatoken/theta/store"
	"github.com/thetatoken/theta/store/database"
	"github.com/thetatoken/theta/store/kvstore"
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
	ChainID      string
	PrivateKey   *crypto.PrivateKey
	Root         *core.Block
	Network      p2p.Network
	DB           database.Database
	SnapshotPath string
}

func NewNode(params *Params) *Node {
	store := kvstore.NewKVStore(params.DB)
	chain := blockchain.NewChain(params.ChainID, store, params.Root)
	validatorManager := consensus.NewRotatingValidatorManager()
	dispatcher := dp.NewDispatcher(params.Network)
	consensus := consensus.NewConsensusEngine(params.PrivateKey, store, chain, dispatcher, validatorManager)

	currentHeight := consensus.GetLastFinalizedBlock().Height
	if currentHeight <= params.Root.Height {
		snapshotPath := params.SnapshotPath
		if _, err := snapshot.ImportSnapshot(snapshotPath, params.DB); err != nil {
			log.Fatalf("Failed to load snapshot: %v, err: %v", snapshotPath, err)
		}
	}

	syncMgr := netsync.NewSyncManager(chain, consensus, params.Network, dispatcher, consensus)
	mempool := mp.CreateMempool(dispatcher)
	ledger := ld.NewLedger(params.ChainID, params.DB, chain, consensus, validatorManager, mempool)
	validatorManager.SetConsensusEngine(consensus)
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

	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		metrics.Enabled = true

		addr, _ := net.ResolveTCPAddr("tcp", mserver)

		// Use chainID.hostname.Theta as prefix.
		hostname, err := os.Hostname()
		if err != nil {
			// Use random string if hostname is not available.
			b := make([]byte, 10)
			rand.Read(b)
			hostname = hex.EncodeToString(b)
		}
		hostname = strings.Replace(hostname, ".", "_", -1)
		chainID := viper.GetString(common.CfgGenesisChainID)
		if chainID == "" {
			chainID = "unknown"
		}
		prefix := fmt.Sprintf("%s.Theta.%s", chainID, hostname)

		go metrics.CollectProcessMetrics(5 * time.Second)
		go metrics.Graphite(metrics.DefaultRegistry, 5*time.Second, prefix, addr)

		// Report heartbeat.
		go func() {
			c := metrics.GetOrRegisterGauge(metrics.MHeartBeat, nil)
			for {
				c.Update(1)
				time.Sleep(time.Second)
			}
		}()
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
