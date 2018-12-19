package messenger

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/thetatoken/ukulele/p2p/netutil"
	pr "github.com/thetatoken/ukulele/p2p/peer"
)

const (
	defaultExternalPort = 7650
	tryListenSeconds    = 5
)

//
// InboundPeerListener models a listener for inbound peer connections
//
type InboundPeerListener struct {
	discMgr *PeerDiscoveryManager

	netListener  net.Listener
	internalAddr *netutil.NetAddress
	externalAddr *netutil.NetAddress

	inboundCallback InboundCallback

	config InboundPeerListenerConfig

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

//
// InboundPeerListenerConfig specifies the configuration for the PeerListener instance
//
type InboundPeerListenerConfig struct {
	numBufferedConnections int
}

// InboundCallback is called when an inbound peer is created
type InboundCallback func(peer *pr.Peer, err error)

// createInboundPeerListener creates a new inbound peer listener instance
func createInboundPeerListener(discMgr *PeerDiscoveryManager, protocol string, localAddr string,
	skipUPNP bool, config InboundPeerListenerConfig) (InboundPeerListener, error) {
	localAddrIP, localAddrPort := splitHostPort(localAddr)
	netListener := initiateNetListener(protocol, localAddr)
	netListenerIP, netListenerPort := splitHostPort(netListener.Addr().String())
	logger.Infof("Local network listener, ip: %v, port: %v", netListenerIP, netListenerPort)

	internalNetAddr := getInternalNetAddress(localAddr)
	externalNetAddr := getExternalNetAddress(localAddrIP, localAddrPort, netListenerPort, skipUPNP)

	inboundPeerListener := InboundPeerListener{
		discMgr:      discMgr,
		netListener:  netListener,
		internalAddr: internalNetAddr,
		externalAddr: externalNetAddr,
		config:       config,
		wg:           &sync.WaitGroup{},
	}

	return inboundPeerListener, nil
}

// GetDefaultInboundPeerListenerConfig returns the default configuration for the listeners
func GetDefaultInboundPeerListenerConfig() InboundPeerListenerConfig {
	return InboundPeerListenerConfig{
		numBufferedConnections: 10,
	}
}

// Start is called when the InboundPeerListener instance starts
func (ipl *InboundPeerListener) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	ipl.ctx = c
	ipl.cancel = cancel

	ipl.wg.Add(1)
	go ipl.listenRoutine()

	return nil
}

// Stop is called when the InboundPeerListener instance stops
func (ipl *InboundPeerListener) Stop() {
	ipl.netListener.Close()
	ipl.cancel()
}

// Wait suspends the caller goroutine
func (ipl *InboundPeerListener) Wait() {
	ipl.wg.Wait()
}

// SetInboundCallback sets the inbound callback function
func (ipl *InboundPeerListener) SetInboundCallback(incb InboundCallback) {
	ipl.inboundCallback = incb
}

func (ipl *InboundPeerListener) listenRoutine() {
	defer ipl.wg.Done()

	for {
		netconn, err := ipl.netListener.Accept()
		if err != nil {
			panic(fmt.Sprintf("net listener error: %v", err))
		}

		peer, err := ipl.discMgr.connectWithInboundPeer(netconn, true)
		if ipl.inboundCallback != nil {
			ipl.inboundCallback(peer, err)
		}
	}
}

// InternalAddress returns the internal address of the current node
func (ipl *InboundPeerListener) InternalAddress() *netutil.NetAddress {
	return ipl.internalAddr
}

// ExternalAddress returns the external address of the current node
func (ipl *InboundPeerListener) ExternalAddress() *netutil.NetAddress {
	return ipl.externalAddr
}

// NetListener returns the attached network listener
func (ipl *InboundPeerListener) NetListener() net.Listener {
	return ipl.netListener
}

func (ipl *InboundPeerListener) String() string {
	return fmt.Sprintf("InboundPeerListener(@%v)", ipl.externalAddr)
}

func splitHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		panic(fmt.Sprintf("failed to split host and port for: %v, err: %v", addr, err))
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("failed to extract port for: %v, err: %v", addr, err))
	}
	return host, port
}

func initiateNetListener(protocol string, localAddr string) (netListener net.Listener) {
	var err error
	for i := 0; i < tryListenSeconds; i++ {
		netListener, err = net.Listen(protocol, localAddr)
		if err == nil {
			break
		} else if i < tryListenSeconds-1 {
			time.Sleep(time.Second)
		}
	}
	if err != nil {
		panic(fmt.Sprintf("Failed to initiate net listener: %v", err))
	}

	return netListener
}

func getInternalNetAddress(localAddr string) *netutil.NetAddress {
	internalAddr, err := netutil.NewNetAddressString(localAddr)
	if err != nil {
		panic(fmt.Sprintf("Failed to get internal network address: %v", err))
	}
	return internalAddr
}

func getExternalNetAddress(localAddrIP string, localAddrPort int, listenerPort int, skipUPNP bool) *netutil.NetAddress {
	var externalAddr *netutil.NetAddress
	if !skipUPNP {
		// If the lAddrIP is INADDR_ANY, try UPNP
		if localAddrIP == "" || localAddrIP == "0.0.0.0" {
			externalAddr = getUPNPExternalAddress(localAddrPort, listenerPort)
		}
	}
	// Otherwise just use the local address
	if externalAddr == nil {
		externalAddr = getNaiveExternalAddress(listenerPort)
	}
	if externalAddr == nil {
		panic(fmt.Sprintf("Could not determine external address!"))
	}

	return externalAddr
}

func getUPNPExternalAddress(externalPort, internalPort int) *netutil.NetAddress {
	logger.Infof("Getting UPNP external address")
	nat, err := netutil.Discover()
	if err != nil {
		logger.Infof("Could not perform UPNP discover: %v", err)
		return nil
	}

	ext, err := nat.GetExternalAddress()
	if err != nil {
		logger.Infof("Could not get UPNP external address: %v", err)
		return nil
	}

	if externalPort == 0 { // Cannot get external port from UPNP, use the default port
		externalPort = defaultExternalPort
	}

	externalPort, err = nat.AddPortMapping("tcp", externalPort, internalPort, "theta", 0)
	if err != nil {
		logger.Infof("Could not add UPNP port mapping: %v", err)
		return nil
	}

	logger.Infof("Got UPNP external address: %v", ext)
	return netutil.NewNetAddressIPPort(ext, uint16(externalPort))
}

func getNaiveExternalAddress(port int) *netutil.NetAddress {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(fmt.Sprintf("Could not fetch interface addresses: %v", err))
	}

	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		v4 := ipnet.IP.To4()
		if v4 == nil || v4[0] == 127 {
			continue
		} // loopback
		return netutil.NewNetAddressIPPort(ipnet.IP, uint16(port))
	}
	return nil
}
