package metrics

import (
	"sync"
	"time"

	pr "github.com/libp2p/go-libp2p-core/peer"
	log "github.com/sirupsen/logrus"
	statsd "github.com/smira/go-statsd"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	msgl "github.com/thetatoken/theta/p2pl/messenger"
)

type StatsdClient struct {
	client *statsd.Client
	init   bool
	InSync bool
	mu     *sync.Mutex
	ID     string
	IP     string

	Msn *msgl.Messenger
}

var client *statsd.Client
var logger *log.Entry = log.WithFields(log.Fields{"prefix": "statsd"})

const sleepTime time.Duration = time.Second * 10
const flushDuration time.Duration = time.Second * 10

func (sc *StatsdClient) NewStatsdClient(sync bool) *StatsdClient {
	return nil
}

//init statsd client and start heartbeat functions
func InitStatsdClient(Msn *msgl.Messenger) *StatsdClient {
	re := &StatsdClient{}
	re.mu = &sync.Mutex{}
	re.Msn = Msn
	re.ID = Msn.ID()
	re.IP, _ = Msn.GetPublicIP()
	logger.Infof("xj1 ID is %s, IP is %s \n", re.ID, re.IP)
	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		client = statsd.NewClient(mserver+":8125", statsd.MetricPrefix("theta."), statsd.FlushInterval(flushDuration))
		re.client = client
		re.init = true
		ticker := time.NewTicker(sleepTime)
		go re.reportOnlineAndSync(ticker)
	} else {
		logger.Infof("metrics server is not in config file")
	}
	return re
}

//report online & sync
func (sc *StatsdClient) reportOnlineAndSync(ticker *time.Ticker) {
	var peerIDs *[]pr.ID
	for {
		select {
		case <-ticker.C:
			sc.client.Incr("guardian.online", 1)
			sc.mu.Lock()
			if sc.InSync {
				sc.client.Incr("guardian.inSync", 1)
			}
			peerIDs = sc.Msn.GetPeerIDs()
			logger.Infof(" get IDs %v\n", peerIDs)
			sc.mu.Unlock()
		}
	}
}

func (sc *StatsdClient) handlePeers() {
}

func (sc *StatsdClient) SetInSync(b bool) {
	sc.mu.Lock()
	sc.InSync = b
	sc.mu.Unlock()
}

func (sc *StatsdClient) SetIP(addr string) {
	sc.mu.Lock()
	sc.mu.Unlock()
}
