package report

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	statsd "github.com/smira/go-statsd"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	dp "github.com/thetatoken/theta/dispatcher"
)

var client *statsd.Client
var logger *log.Entry = log.WithFields(log.Fields{"prefix": "statsd"})
var reportPeersPort string = ":9000"
var reportStatsdPort string = ":8125"
var setPeersSuffix string = "/peers/set"

const step int64 = 60
const sleepTime time.Duration = time.Second * 60
const flushDuration time.Duration = time.Second * 60

type Reporter struct {
	client *statsd.Client
	inSync bool
	mu     *sync.Mutex
	id     string
	ipAddr string

	disp   *dp.Dispatcher
	ticker *time.Ticker

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewReporter instantiates a reporter instance
func NewReporter(disp *dp.Dispatcher) *Reporter {
	ipAddr, err := util.GetPublicIP()
	if err != nil {
		logger.Errorf("Reporter failed to retrieve the node's IP address: %v", err)
	}

	var client *statsd.Client
	if mserver := viper.GetString(common.CfgMetricsServer) + reportStatsdPort; mserver != "" {
		client = statsd.NewClient(mserver, statsd.MetricPrefix("theta."), statsd.FlushInterval(flushDuration))
	} else {
		logger.Infof("metrics server is not in config file")
	}

	rp := &Reporter{
		client: client,
		inSync: false,
		mu:     &sync.Mutex{},
		id:     disp.ID(),
		ipAddr: ipAddr,
		disp:   disp,
		ticker: time.NewTicker(sleepTime),
	}

	logger.Infof("node ID is %s, IP Address is %s \n", rp.id, rp.ipAddr)

	return rp
}

// Start is called when the reporter starts
func (rp *Reporter) Start(ctx context.Context) error {
	if rp.client == nil {
		return fmt.Errorf("Failed to start the stats reporter, rp.client == nil")
	}

	go rp.reportOnlineAndSync()

	return nil
}

// Stop is called when the reporter stops
func (rp *Reporter) Stop() {
	rp.cancel()
}

// Wait suspends the caller goroutine
func (rp *Reporter) Wait() {
	rp.wg.Wait()
}

//report online & sync
func (rp *Reporter) reportOnlineAndSync() {
	// var peerIDs *[]pr.ID
	for {
		select {
		case <-rp.ticker.C:
			rp.client.Incr("guardian.online", step)
			rp.mu.Lock()
			if rp.inSync {
				rp.client.Incr("guardian.inSync", step)
			}
			rp.mu.Unlock()
			rp.handlePeers()
		}
	}
}

func (rp *Reporter) handlePeers() {
	url := "http://" + viper.GetString(common.CfgMetricsServer) + reportPeersPort + setPeersSuffix
	jsonStr := fmt.Sprintf("{\"id\":\"%s\", \"ip\": \"%s\", \"peers\" : [%s]}", rp.id, rp.ipAddr, rp.peersToString())
	data := []byte(jsonStr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		logger.Errorf("Reporter failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("Reporter failed to create request: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Debug("response Status:", resp.Status)
	log.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("response Body:", string(body))
}

func (rp *Reporter) SetInSync(inSync bool) {
	rp.mu.Lock()
	rp.inSync = inSync
	rp.mu.Unlock()
}

func (rp *Reporter) peersToString() string {
	p := rp.disp.Peers()
	var sb strings.Builder
	sb.WriteString("[")
	for i, peer := range p {
		if i > 0 {
			sb.WriteString(",\"")
		} else {
			sb.WriteString("\"")
		}
		sb.WriteString(peer)
		sb.WriteString("\"")
	}
	sb.WriteString("]")
	log.Debug("peers is : %v, stringbuilder is : %s \n", p, sb.String())
	return sb.String()
}
