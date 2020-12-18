package rpc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"

	"net/rpc"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/dispatcher"
	"github.com/thetatoken/theta/ledger"
	"github.com/thetatoken/theta/mempool"
	"github.com/thetatoken/theta/rpc/lib/rpc-codec/jsonrpc2"
	"golang.org/x/net/netutil"
	"golang.org/x/net/websocket"
)

var logger *log.Entry

type ThetaRPCService struct {
	mempool    *mempool.Mempool
	ledger     *ledger.Ledger
	dispatcher *dispatcher.Dispatcher
	chain      *blockchain.Chain
	consensus  *consensus.ConsensusEngine

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
func NewThetaRPCServer(mempool *mempool.Mempool, ledger *ledger.Ledger, dispatcher *dispatcher.Dispatcher,
	chain *blockchain.Chain, consensus *consensus.ConsensusEngine) *ThetaRPCServer {
	t := &ThetaRPCServer{
		ThetaRPCService: &ThetaRPCService{
			wg: &sync.WaitGroup{},
		},
	}

	t.mempool = mempool
	t.ledger = ledger
	t.dispatcher = dispatcher
	t.chain = chain
	t.consensus = consensus

	s := rpc.NewServer()
	s.RegisterName("theta", t.ThetaRPCService)

	t.handler = s

	t.router = mux.NewRouter()
	t.router.Handle("/", &defaultHTTPHandler{})
	t.router.Handle("/rpc", corsMiddleware(TimeoutHandler(jsonrpc2.HTTPHandler(s), viper.GetDuration(common.CfgRPCTimeoutSecs)*time.Second, "")))
	t.router.Handle("/ws", websocket.Handler(func(ws *websocket.Conn) {
		s.ServeCodec(jsonrpc2.NewServerCodec(ws, s))
	}))

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
	address := viper.GetString(common.CfgRPCAddress)
	port := viper.GetString(common.CfgRPCPort)
	l, err := net.Listen("tcp", address+":"+port)
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Fatal("Failed to create listener")
	} else {
		logger.WithFields(log.Fields{"address": address, "port": port}).Info("RPC server started")
	}
	defer l.Close()

	ll := netutil.LimitListener(l, viper.GetInt(common.CfgRPCMaxConnections))
	t.listener = ll

	logger.Info(t.server.Serve(ll))
}

func corsMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//Allow CORS here By * or specific origin
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// Stop notifies all goroutines to stop without blocking.
func (t *ThetaRPCServer) Stop() {
	t.cancel()
}

// Wait blocks until all goroutines stop.
func (t *ThetaRPCServer) Wait() {
	t.wg.Wait()
}

type defaultHTTPHandler struct {
}

func (dh *defaultHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Theta Node is up and running!")
}

//
// Adapted from https://golang.org/src/net/http/server.go
//

// TimeoutHandler returns a Handler that runs h with the given time limit.
//
// The new Handler calls h.ServeHTTP to handle each request, but if a
// call runs for longer than its time limit, the handler responds with
// a 503 Service Unavailable error and the given message in its body.
// (If msg is empty, a suitable default message will be sent.)
// After such a timeout, writes by h to its ResponseWriter will return
// ErrHandlerTimeout.
//
// TimeoutHandler supports the Pusher interface but does not support
// the Hijacker or Flusher interfaces.
func TimeoutHandler(h http.Handler, dt time.Duration, msg string) http.Handler {
	return &timeoutHandler{
		handler: h,
		body:    msg,
		dt:      dt,
	}
}

type timeoutHandler struct {
	handler http.Handler
	body    string
	dt      time.Duration

	// When set, no context will be created and this context will
	// be used instead.
	testContext context.Context
}

func (h *timeoutHandler) errorBody() string {
	if h.body != "" {
		return h.body
	}
	return "{\"error\": {\"message\":\"Timeout\"}}"
}

func (h *timeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.testContext
	if ctx == nil {
		var cancelCtx context.CancelFunc
		ctx, cancelCtx = context.WithTimeout(r.Context(), h.dt)
		defer cancelCtx()
	}
	r = r.WithContext(ctx)
	done := make(chan struct{})
	tw := &timeoutWriter{
		w:   w,
		h:   make(http.Header),
		req: r,
	}
	panicChan := make(chan interface{}, 1)

	buf, bodyErr := ioutil.ReadAll(r.Body)
	if bodyErr != nil {
		http.Error(w, bodyErr.Error(), http.StatusInternalServerError)
		return
	}

	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
	r.Body = rdr2

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()
		h.handler.ServeHTTP(tw, r)
		close(done)
	}()
	select {
	case p := <-panicChan:
		panic(p)
	case <-done:
		tw.mu.Lock()
		defer tw.mu.Unlock()

		dst := w.Header()
		for k, vv := range tw.h {
			dst[k] = vv
		}
		if !tw.wroteHeader {
			tw.code = http.StatusOK
		}
		w.WriteHeader(tw.code)
		w.Write(tw.wbuf.Bytes())
	case <-ctx.Done():
		tw.mu.Lock()
		defer tw.mu.Unlock()
		w.WriteHeader(http.StatusServiceUnavailable)
		io.WriteString(w, h.errorBody())
		tw.timedOut = true
		logger.Errorf("HTTP request timed out: requestBody=%q", rdr1)
	}
}

type timeoutWriter struct {
	w    http.ResponseWriter
	h    http.Header
	wbuf bytes.Buffer
	req  *http.Request

	mu          sync.Mutex
	timedOut    bool
	wroteHeader bool
	code        int
}

var _ http.Pusher = (*timeoutWriter)(nil)

// Push implements the Pusher interface.
func (tw *timeoutWriter) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := tw.w.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return http.ErrNotSupported
}

func (tw *timeoutWriter) Header() http.Header { return tw.h }

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	if tw.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if !tw.wroteHeader {
		tw.writeHeaderLocked(http.StatusOK)
	}
	return tw.wbuf.Write(p)
}

func (tw *timeoutWriter) writeHeaderLocked(code int) {
	switch {
	case tw.timedOut:
		return
	case tw.wroteHeader:
	default:
		tw.wroteHeader = true
		tw.code = code
	}
}

func (tw *timeoutWriter) WriteHeader(code int) {
	tw.mu.Lock()
	defer tw.mu.Unlock()
	tw.writeHeaderLocked(code)
}
