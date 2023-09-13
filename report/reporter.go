package report

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/blockchain"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	dp "github.com/thetatoken/theta/dispatcher"
	"github.com/thetatoken/theta/version"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "reporter"})
var reportPeersPort string = ":9000"
var setPeersSuffix string = "/peers/set"
var peerUrl string
var rpcJSON = []byte(`{"jsonrpc": "2.0", "method": "theta.GetStatus", "params": [{}], "id": 0}`)

const sleepTime time.Duration = time.Second * 60 * 10

type Reporter struct {
	init   bool
	id     string
	ipAddr string

	consensus *consensus.ConsensusEngine
	disp      *dp.Dispatcher
	chain     *blockchain.Chain
	ticker    *time.Ticker

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// NewReporter instantiates a reporter instance
func NewReporter(disp *dp.Dispatcher, consensus *consensus.ConsensusEngine, chain *blockchain.Chain) *Reporter {
	peerUrl = "http://" + viper.GetString(common.CfgMetricsServer) + reportPeersPort + setPeersSuffix
	ipAddr, err := util.GetPublicIP()
	if err != nil {
		logger.Warnf("Reporter failed to retrieve the node's IP address: %v", err)
	}
	var ok bool = true
	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		logger.Infof("metrics server is not in config file")
		ok = false
	}

	rp := &Reporter{
		init:      ok,
		id:        disp.ID(),
		ipAddr:    ipAddr,
		consensus: consensus,
		disp:      disp,
		chain:     chain,
		ticker:    time.NewTicker(sleepTime),
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
	s := rp.consensus.GetSummary()
	latestFinalizedHash := s.LastFinalizedBlock
	addition := ""
	if !latestFinalizedHash.IsEmpty() {
		block, err := rp.chain.FindBlock(latestFinalizedHash)
		if err == nil {
			addition = fmt.Sprintf(`,"LatestFinalizedBlockHeight":%d,"syncing":"%v"`, common.JSONUint64(block.Height), isSyncing(block))
		}
	}
	result := fmt.Sprintf(`"version":"%s", "git_hash":"%s", "address":"%s", "chain_id":"%s", "OS":"%s"%s`, version.Version, version.GitHash, rp.consensus.ID(), rp.chain.ChainID, runtime.GOOS, addition)
	return result
}

func (rp *Reporter) handlePeers() {
	url := "http://" + viper.GetString(common.CfgMetricsServer) + reportPeersPort + setPeersSuffix
	jsonStr := fmt.Sprintf("{\"id\":\"%s\", \"ip\": \"%s\", \"peers\" : [%s], %s}",
		rp.id, rp.ipAddr, rp.peersToString(), rp.statusToString())
	data := []byte(jsonStr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		logger.Warnf("Reporter failed to create request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Warnf("Reporter failed to create request: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Debug("response Status:", resp.Status)
	log.Debug("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	log.Debug("response Body:", string(body))
}

func (rp *Reporter) peersToString() string {
	p := rp.disp.Peers(true) // skip edge nodes
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

func isSyncing(block *core.ExtendedBlock) bool {
	currentTime := big.NewInt(time.Now().Unix())
	maxDiff := new(big.Int).SetUint64(30) // thirty seconds, about 5 blocks
	threshold := new(big.Int).Sub(currentTime, maxDiff)
	isSyncing := block.Timestamp.Cmp(threshold) < 0
	return isSyncing
}
