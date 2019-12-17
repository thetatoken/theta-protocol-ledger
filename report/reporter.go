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

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	dp "github.com/thetatoken/theta/dispatcher"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "reporter"})
var reportPeersPort string = ":9000"
var setPeersSuffix string = "/peers/set"
var peerUrl string
var rpcJSON = []byte(`{"jsonrpc": "2.0", "method": "theta.GetStatus", "params": [{}], "id": 0}`)

const sleepTime time.Duration = time.Second * 60
const rpcUrl = "http://localhost:16888/rpc"

type Reporter struct {
	init   bool
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
	peerUrl = "http://" + viper.GetString(common.CfgMetricsServer) + reportPeersPort + setPeersSuffix
	ipAddr, err := util.GetPublicIP()
	if err != nil {
		logger.Errorf("Reporter failed to retrieve the node's IP address: %v", err)
	}
	var ok bool = true
	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		logger.Infof("metrics server is not in config file")
		ok = false
	}

	rp := &Reporter{
		init:   ok,
		id:     disp.ID(),
		ipAddr: ipAddr,
		disp:   disp,
		ticker: time.NewTicker(sleepTime),
	}

	logger.Infof("node ID is %s, IP Address is %s", rp.id, rp.ipAddr)

	return rp
}

// Start is called when the reporter starts
func (rp *Reporter) Start(ctx context.Context) error {
	if !rp.init {

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
	for {
		select {
		case <-rp.ticker.C:
			rp.handlePeers()
		}
	}
}

func (rp *Reporter) statusToString() string {
	req, err := http.NewRequest("POST", rpcUrl, bytes.NewBuffer(rpcJSON))
	if err != nil {
		logger.Errorf("Reporter failed to send getting node status request: %v", err)
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Errorf("Reporter failed to get node status: %v", err)
		return ""
	}
	defer resp.Body.Close()
	log.Debug("response Status:", resp.Status)
	log.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	bodyStr := string(body)
	log.Debug("response Body:", bodyStr)
	start := strings.Index(bodyStr[1:], "{")
	result := bodyStr[start+2 : len(bodyStr)-3]
	return result
}

func (rp *Reporter) handlePeers() {
	url := "http://" + viper.GetString(common.CfgMetricsServer) + reportPeersPort + setPeersSuffix
	jsonStr := fmt.Sprintf("{\"id\":\"%s\", \"ip\": \"%s\", \"peers\" : [%s], %s}",
		rp.id, rp.ipAddr, rp.peersToString(), rp.statusToString())
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

func (rp *Reporter) peersToString() string {
	p := rp.disp.Peers()
	var sb strings.Builder
	for i, peer := range p {
		if i > 0 {
			sb.WriteString(",\"")
		} else {
			sb.WriteString("\"")
		}
		sb.WriteString(peer)
		sb.WriteString("\"")
	}
	log.Debug("peers is : %v, stringbuilder is : %s \n", p, sb.String())
	return sb.String()
}
