package messenger

import (
	"fmt"
	"net"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/p2p/netutil"
)

const (
	defaultExternalPort = 7650
	tryListenSeconds    = 5
)

//
// PeerListener models a listener for peer connections
//
type PeerListener struct {
	netListener  net.Listener
	internalAddr *netutil.NetAddress
	externalAddr *netutil.NetAddress
	netconns     chan net.Conn

	config PeerListenerConfig
}

//
// PeerListenerConfig specifies the configuration for the PeerListener instance
//
type PeerListenerConfig struct {
	numBufferedConnections int
}

// CreatePeerListener creates a new peer listener instance
func CreatePeerListener(protocol string, localAddr string, skipUPNP bool, config PeerListenerConfig) PeerListener {
	localAddrIP, localAddrPort := splitHostPort(localAddr)
	netListener := initiateNetListener(protocol, localAddr)
	netListenerIP, netListenerPort := splitHostPort(netListener.Addr().String())
	log.Infof("[p2p] Local network listener, ip: %v, port: %v", netListenerIP, netListenerPort)

	internalNetAddr := getInternalNetAddress(localAddr)
	externalNetAddr := getExternalNetAddress(localAddrIP, localAddrPort, netListenerPort, skipUPNP)

	peerListener := PeerListener{
		netListener:  netListener,
		internalAddr: internalNetAddr,
		externalAddr: externalNetAddr,
		netconns:     make(chan net.Conn, config.numBufferedConnections),
	}

	return peerListener
}

// CreateDefaultPeerListenerConfig returns the default configuration for the listeners
func CreateDefaultPeerListenerConfig() PeerListenerConfig {
	return PeerListenerConfig{
		numBufferedConnections: 10,
	}
}

// OnStart is called when the PeerListener instance starts
func (pl *PeerListener) OnStart() error {
	go pl.listenRoutine()
	return nil
}

// OnStop is called when the PeerListener instance stops
func (pl *PeerListener) OnStop() {
	pl.netListener.Close()
}

func (pl *PeerListener) listenRoutine() {
	for {
		netconn, err := pl.netListener.Accept()
		if err != nil {
			close(pl.netconns)
			panic(fmt.Sprintf("[p2p] net listener error: %v", err))
		}

		pl.netconns <- netconn
	}
}

// Connections returns all the network connections attached
func (pl *PeerListener) Connections() <-chan net.Conn {
	return pl.netconns
}

// InternalAddress returns the internal address of the current node
func (pl *PeerListener) InternalAddress() *netutil.NetAddress {
	return pl.internalAddr
}

// ExternalAddress returns the external address of the current node
func (pl *PeerListener) ExternalAddress() *netutil.NetAddress {
	return pl.externalAddr
}

// NetListener returns the attached network listener
func (pl *PeerListener) NetListener() net.Listener {
	return pl.netListener
}

func (pl *PeerListener) String() string {
	return fmt.Sprintf("PeerListener(@%v)", pl.externalAddr)
}

func splitHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		panic(fmt.Sprintf("[p2p] failed to split host and port for: %v, err: %v", addr, err))
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		panic(fmt.Sprintf("[p2p] failed to extract port for: %v, err: %v", addr, err))
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
		panic(fmt.Sprintf("[p2p] Failed to initiate net listener: %v", err))
	}

	return netListener
}

func getInternalNetAddress(localAddr string) *netutil.NetAddress {
	internalAddr, err := netutil.NewNetAddressString(localAddr)
	if err != nil {
		panic(fmt.Sprintf("[p2p] Failed to get internal network address: %v", err))
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
		panic(fmt.Sprintf("[p2p] Could not determine external address!"))
	}

	return externalAddr
}

func getUPNPExternalAddress(externalPort, internalPort int) *netutil.NetAddress {
	log.Infof("[p2p] Getting UPNP external address")
	nat, err := netutil.Discover()
	if err != nil {
		log.Infof("[p2p] Could not perform UPNP discover: %v", err)
		return nil
	}

	ext, err := nat.GetExternalAddress()
	if err != nil {
		log.Infof("[p2p] Could not get UPNP external address: %v", err)
		return nil
	}

	if externalPort == 0 { // Cannot get external port from UPNP, use the default port
		externalPort = defaultExternalPort
	}

	externalPort, err = nat.AddPortMapping("tcp", externalPort, internalPort, "theta", 0)
	if err != nil {
		log.Infof("[p2p] Could not add UPNP port mapping: %v", err)
		return nil
	}

	log.Infof("[p2p] Got UPNP external address: %v", ext)
	return netutil.NewNetAddressIPPort(ext, uint16(externalPort))
}

func getNaiveExternalAddress(port int) *netutil.NetAddress {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		panic(fmt.Sprintf("[p2p] Could not fetch interface addresses: %v", err))
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
