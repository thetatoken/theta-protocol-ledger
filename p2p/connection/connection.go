package connection

import (
	"bufio"
	"net"
	"time"

	"github.com/thetatoken/ukulele/p2p/netutil"
)

//
// Connection models the connection between the current node and a peer node.
// One link can contain multiple channels
//
type Connection struct {
	conn      net.Conn
	bufReader *bufio.Reader
	bufWriter *bufio.Writer

	channels []*Channel
	//onReceive receiveCbFunc
	//onError   errorCbFunc
	errored uint32

	quit chan struct{}
	//flushTimer *ThrottleTimer // flush writes as necessary but throttled
	//pingTimer  *RepeatTimer   // send pings periodically

	config ConnectionConfig
}

type ConnectionConfig struct {
	DialTimeout time.Duration
}

// createConnection creates a Connection instance
func createConnection(conn net.Conn) *Connection {
	return nil
}

func dial(addr *netutil.NetAddress, config *ConnectionConfig) (net.Conn, error) {
	conn, err := addr.DialTimeout(config.DialTimeout * time.Second)
	if err != nil {
		return nil, err
	}
	return conn, nil
}
