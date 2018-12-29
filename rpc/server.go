package rpc

import (
	"context"
	"net"
	"net/http"
	"sync"

	"net/rpc"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/blockchain"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/consensus"
	"github.com/thetatoken/ukulele/ledger"
	"github.com/thetatoken/ukulele/mempool"
	"github.com/thetatoken/ukulele/rpc/lib/rpc-codec/jsonrpc2"
	"golang.org/x/net/netutil"
)

var logger *log.Entry

type ThetaRPCService struct {
	mempool   *mempool.Mempool
	ledger    *ledger.Ledger
	chain     *blockchain.Chain
	consensus *consensus.ConsensusEngine

	// Life cycle
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// ThetaRPCServer is an instance of RPC service.
type ThetaRPCServer struct {
	*ThetaRPCService

	server   *http.Server
	handler  *rpc.Server
	router   *mux.Router
	listener net.Listener
}

// NewThetaRPCServer creates a new instance of ThetaRPCServer.
func NewThetaRPCServer(mempool *mempool.Mempool, ledger *ledger.Ledger, chain *blockchain.Chain, consensus *consensus.ConsensusEngine) *ThetaRPCServer {
	t := &ThetaRPCServer{
		ThetaRPCService: &ThetaRPCService{
			wg: &sync.WaitGroup{},
		},
	}

	t.mempool = mempool
	t.ledger = ledger
	t.chain = chain
	t.consensus = consensus

	s := rpc.NewServer()
	s.RegisterName("theta", t.ThetaRPCService)

	t.handler = s

	t.router = mux.NewRouter()
	t.router.Handle("/rpc", jsonrpc2.HTTPHandler(s))

	t.server = &http.Server{
		Handler: t.router,
	}

	logger = util.GetLoggerForModule("rpc")

	return t
}

// Start creates the main goroutine.
func (t *ThetaRPCServer) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	t.ctx = c
	t.cancel = cancel

	t.wg.Add(1)
	go t.mainLoop()

	t.wg.Add(1)
	go t.txCallback()
}

func (t *ThetaRPCServer) mainLoop() {
	defer t.wg.Done()

	go t.serve()

	<-t.ctx.Done()
	t.stopped = true
	t.server.Shutdown(t.ctx)
}

func (t *ThetaRPCServer) serve() {
	port := viper.GetString(common.CfgRPCPort)
	l, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Fatal("Failed to create listener")
	} else {
		logger.WithFields(log.Fields{"port": port}).Info("RPC server started")
	}
	defer l.Close()

	ll := netutil.LimitListener(l, viper.GetInt(common.CfgRPCMaxConnections))
	t.listener = ll

	logger.Fatal(t.server.Serve(ll))
}

// Stop notifies all goroutines to stop without blocking.
func (t *ThetaRPCServer) Stop() {
	t.cancel()
}

// Wait blocks until all goroutines stop.
func (t *ThetaRPCServer) Wait() {
	t.wg.Wait()
}
