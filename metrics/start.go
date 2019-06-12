package metrics

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	mlib "github.com/thetatoken/theta/common/metrics"
)

func Start(ctx context.Context) {
	mserver := viper.GetString(common.CfgMetricsServer)
	if mserver == "" {
		return
	}

	go reportProcessInfo()
	go reportHeartBeat()
	go sendToGraphite(mserver)
}

func reportProcessInfo() {
	mlib.CollectProcessMetrics(5 * time.Second)
}

func reportHeartBeat() {
	c := mlib.GetOrRegisterGauge(MHeartBeat, nil)
	for {
		c.Update(1)
		time.Sleep(time.Second)
	}
}

func sendToGraphite(mserver string) {
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

	mlib.Graphite(mlib.DefaultRegistry, 5*time.Second, prefix, addr)
}
