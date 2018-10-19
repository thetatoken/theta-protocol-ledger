package rpc

import (
	"context"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/rpc/v2"
	json "github.com/gorilla/rpc/v2/json2"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/util"
	"github.com/thetatoken/ukulele/ledger"
	"github.com/thetatoken/ukulele/mempool"
	"golang.org/x/net/netutil"
)

var logger *log.Entry

func init() {
	logger = util.GetLoggerForModule("rpc")
}

// ThetaRPCServer is an instance of RPC service.
type ThetaRPCServer struct {
	mempool *mempool.Mempool
	ledger  *ledger.Ledger

	server   *http.Server
	handler  *rpc.Server
	router   *mux.Router
	listener net.Listener

	// Life cycle
	wg      *sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewThetaRPCServer creates a new instance of ThetaRPCServer.
func NewThetaRPCServer(mempool *mempool.Mempool, ledger *ledger.Ledger) *ThetaRPCServer {
	t := &ThetaRPCServer{
		wg: &sync.WaitGroup{},
	}

	t.mempool = mempool
	t.ledger = ledger

	t.handler = rpc.NewServer()
	t.handler.RegisterCodec(json.NewCodec(), "application/json")
	t.handler.RegisterCodec(json.NewCodec(), "application/json;charset=UTF-8")
	t.handler.RegisterService(t, "theta")

	t.router = mux.NewRouter()
	t.router.Handle("/rpc", t.handler)

	t.server = &http.Server{
		Handler: t.router,
	}

	return t
}

// Start creates the main goroutine.
func (t *ThetaRPCServer) Start(ctx context.Context) {
	c, cancel := context.WithCancel(ctx)
	t.ctx = c
	t.cancel = cancel

	go t.mainLoop()
}

func (t *ThetaRPCServer) mainLoop() {
	t.wg.Add(1)
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
