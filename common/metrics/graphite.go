package metrics

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
)

// GraphiteConfig provides a container with configuration parameters for
// the Graphite exporter
type GraphiteConfig struct {
	Addr          *net.TCPAddr  // Network address to connect to
	Registry      Registry      // Registry to be exported
	FlushInterval time.Duration // Flush interval
	DurationUnit  time.Duration // Time conversion unit for durations
	Prefix        string        // Prefix to be prepended to metric names
	Percentiles   []float64     // Percentiles to export from timers and histograms
}

// Graphite is a blocking exporter function which reports metrics in r
// to a graphite server located at addr, flushing them every d duration
// and prepending metric names with prefix.
func Graphite(r Registry, d time.Duration, prefix string, addr *net.TCPAddr) {
	GraphiteWithConfig(GraphiteConfig{
		Addr:          addr,
		Registry:      r,
		FlushInterval: d,
		DurationUnit:  time.Nanosecond,
		Prefix:        prefix,
		Percentiles:   []float64{0.5, 0.75, 0.95, 0.99, 0.999},
	})
}

// GraphiteWithConfig is a blocking exporter function just like Graphite,
// but it takes a GraphiteConfig instead.
func GraphiteWithConfig(c GraphiteConfig) {
	log.Printf("WARNING: This go-metrics client has been DEPRECATED! It has been moved to https://github.com/cyberdelia/go-metrics-graphite and will be removed from rcrowley/go-metrics on August 12th 2015")
	for range time.Tick(c.FlushInterval) {
		if err := graphite(&c); nil != err {
			log.Println(err)
		}
	}
}

// GraphiteOnce performs a single submission to Graphite, returning a
// non-nil error on failed connections. This can be used in a loop
// similar to GraphiteWithConfig for custom error handling.
func GraphiteOnce(c GraphiteConfig) error {
	log.Printf("WARNING: This go-metrics client has been DEPRECATED! It has been moved to https://github.com/cyberdelia/go-metrics-graphite and will be removed from rcrowley/go-metrics on August 12th 2015")
	return graphite(&c)
}

func graphite(c *GraphiteConfig) error {
	now := time.Now().Unix()
	du := float64(c.DurationUnit)
	conn, err := net.DialTCP("tcp", nil, c.Addr)
	if nil != err {
		return err
	}
	defer conn.Close()
	w := bufio.NewWriter(conn)
	c.Registry.Each(func(name string, i interface{}) {
		switch metric := i.(type) {
		case Counter:
			fmt.Fprintf(w, "%s.%s.count %d %d\n", c.Prefix, name, metric.Count(), now)
		case Gauge:
			fmt.Fprintf(w, "%s.%s.value %d %d\n", c.Prefix, name, metric.Value(), now)
		case GaugeFloat64:
			fmt.Fprintf(w, "%s.%s.value %f %d\n", c.Prefix, name, metric.Value(), now)
		case Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles(c.Percentiles)
			fmt.Fprintf(w, "%s.%s.count %d %d\n", c.Prefix, name, h.Count(), now)
			fmt.Fprintf(w, "%s.%s.min %d %d\n", c.Prefix, name, h.Min(), now)
			fmt.Fprintf(w, "%s.%s.max %d %d\n", c.Prefix, name, h.Max(), now)
			fmt.Fprintf(w, "%s.%s.mean %.2f %d\n", c.Prefix, name, h.Mean(), now)
			fmt.Fprintf(w, "%s.%s.std-dev %.2f %d\n", c.Prefix, name, h.StdDev(), now)
			for psIdx, psKey := range c.Percentiles {
				key := strings.Replace(strconv.FormatFloat(psKey*100.0, 'f', -1, 64), ".", "", 1)
				fmt.Fprintf(w, "%s.%s.%s-percentile %.2f %d\n", c.Prefix, name, key, ps[psIdx], now)
			}
		case Meter:
			m := metric.Snapshot()
			fmt.Fprintf(w, "%s.%s.count %d %d\n", c.Prefix, name, m.Count(), now)
			fmt.Fprintf(w, "%s.%s.one-minute %.2f %d\n", c.Prefix, name, m.Rate1(), now)
			fmt.Fprintf(w, "%s.%s.five-minute %.2f %d\n", c.Prefix, name, m.Rate5(), now)
			fmt.Fprintf(w, "%s.%s.fifteen-minute %.2f %d\n", c.Prefix, name, m.Rate15(), now)
			fmt.Fprintf(w, "%s.%s.mean %.2f %d\n", c.Prefix, name, m.RateMean(), now)
		case Timer:
			t := metric.Snapshot()
			ps := t.Percentiles(c.Percentiles)
			fmt.Fprintf(w, "%s.%s.count %d %d\n", c.Prefix, name, t.Count(), now)
			fmt.Fprintf(w, "%s.%s.min %d %d\n", c.Prefix, name, t.Min()/int64(du), now)
			fmt.Fprintf(w, "%s.%s.max %d %d\n", c.Prefix, name, t.Max()/int64(du), now)
			fmt.Fprintf(w, "%s.%s.mean %.2f %d\n", c.Prefix, name, t.Mean()/du, now)
			fmt.Fprintf(w, "%s.%s.std-dev %.2f %d\n", c.Prefix, name, t.StdDev()/du, now)
			for psIdx, psKey := range c.Percentiles {
				key := strings.Replace(strconv.FormatFloat(psKey*100.0, 'f', -1, 64), ".", "", 1)
				fmt.Fprintf(w, "%s.%s.%s-percentile %.2f %d\n", c.Prefix, name, key, ps[psIdx], now)
			}
			fmt.Fprintf(w, "%s.%s.one-minute %.2f %d\n", c.Prefix, name, t.Rate1(), now)
			fmt.Fprintf(w, "%s.%s.five-minute %.2f %d\n", c.Prefix, name, t.Rate5(), now)
			fmt.Fprintf(w, "%s.%s.fifteen-minute %.2f %d\n", c.Prefix, name, t.Rate15(), now)
			fmt.Fprintf(w, "%s.%s.mean-rate %.2f %d\n", c.Prefix, name, t.RateMean(), now)
		}
		w.Flush()
	})
	return nil
}

//--------------------new functions---------------------
var gs *GraphiteServer

type GraphiteServer struct {
	Addr       string // Network address to connect to
	TCPaddr    *net.TCPAddr
	Hostname   string
	Prefix     string
	repeatable []*MetricJob
}

type MetricJob struct {
	FlushInterval time.Duration // Flush interval
	Prefix        string        // Prefix to be prepended to metric names
}

func GetGraphiteServer() *GraphiteServer {
	if gs == nil {
		if mserver := viper.GetString(common.CfgMetricsServer); mserver != "" {
			addr, err := net.ResolveTCPAddr("tcp", mserver+":2003")
			if err != nil {
				log.Printf("can not connect to server")
				return nil
			}
			gs = &GraphiteServer{
				Addr:       mserver,
				TCPaddr:    addr,
				repeatable: make([]*MetricJob, 0),
			}
			// fmt.Printf("xj1 addr is %v, msserver is %s \n", addr, mserver)
		}
		hostname, err := os.Hostname()
		if err != nil {
			// Use random string if hostname is not available.
			b := make([]byte, 10)
			rand.Read(b)
			hostname = hex.EncodeToString(b)
		}
		hostname = strings.Replace(hostname, ".", "_", -1)
		gs.Hostname = hostname
		prefix := fmt.Sprintf("%s.Theta.%s", viper.GetString(common.CfgLibP2PRendezvous), hostname)
		gs.Prefix = prefix
	}
	return gs
}

func (gs *GraphiteServer) addJob() error {
	return nil
}

func (gs *GraphiteServer) OneTimeJob(key string, value string) error {
	conn, err := net.DialTCP("tcp", nil, gs.TCPaddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	message := fmt.Sprintf("%s.%s %s %d\n", gs.Prefix, key, value, time.Now().Unix())
	if _, err := conn.Write([]byte(message)); err != nil {
		return err
	}
	fmt.Printf("xj2 message is %s \n", message)
	return nil
}

func (gs *GraphiteServer) String() string {
	return fmt.Sprintf("GraphServer: address is %s, prefix is %s. \n", gs.Addr, gs.Prefix)
}
