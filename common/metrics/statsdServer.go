package metrics

import (
	"fmt"
	"sync"
	"time"

	"github.com/smira/go-statsd"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
)

type StatsdClient struct {
	client *statsd.Client
	init   bool
	InSync bool //TODO : lock
	mu     *sync.Mutex
}

var client *statsd.Client

const sleepTime time.Duration = time.Second * 10
const flushDuration time.Duration = time.Second * 10

func (sc *StatsdClient) NewStatsdClient(sync bool) *StatsdClient {
	return nil
}

//init statsd client and start heartbeat functions
func InitStatsdClient() *StatsdClient {
	re := &StatsdClient{}
	re.mu = &sync.Mutex{}
	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		client = statsd.NewClient(mserver+":8125", statsd.MetricPrefix("theta."), statsd.FlushInterval(flushDuration))
		re.client = client
		re.init = true
		ticker := time.NewTicker(sleepTime)
		go re.reportOnlineAndSync(ticker)
	} else {
		fmt.Printf("metrics server is not in config file")
	}
	return re
}

//report online & sync
func (sc *StatsdClient) reportOnlineAndSync(ticker *time.Ticker) {
	for {
		select {
		case <-ticker.C:
			sc.client.Incr("guardian.online", 1)
			sc.mu.Lock()
			if sc.InSync {
				sc.client.Incr("guardian.inSync", 1)
			}
			sc.mu.Unlock()
		}
	}
}

func (sc *StatsdClient) SetInSync(b bool) {
	sc.mu.Lock()
	sc.InSync = b
	sc.mu.Unlock()
}
