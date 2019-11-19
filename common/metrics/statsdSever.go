package metrics

import (
	"fmt"
	"time"

	"github.com/smira/go-statsd"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
)

type StatsdClient struct {
	client *statsd.Client
	init   bool
}

var client *statsd.Client

const sleepTime time.Duration = time.Second * 5
const flushDuration time.Duration = time.Second

func (sc *StatsdClient) NewStatsdClient(sync bool) *StatsdClient {
	return nil
}

//block function to init statsd client and start heartbeat functions
func InitStatsdClient(sync bool) *StatsdClient {
	re := &StatsdClient{}
	if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
		client = statsd.NewClient(mserver+":8125", statsd.MetricPrefix("theta."), statsd.FlushInterval(flushDuration))
		re.client = client
		re.init = true
		defer client.Close()
		c := make(chan bool)
		go reportOnline(client)
		if sync {
			go reportSync(client)
		}
		done := <-c
		if done {
			return nil
		}
	} else {
		fmt.Printf("metrics server is not in config file")
	}
	return re
}

//report online
func reportOnline(client *statsd.Client) {
	// TODO: check if this is a guardian node
	for {
		client.Incr("guardian.online", 1)
		time.Sleep(flushDuration)
	}
}

//report sync
func reportSync(client *statsd.Client) {
	for {
		client.Incr("guardian.inSync", 1)
		time.Sleep(flushDuration)
	}
}
