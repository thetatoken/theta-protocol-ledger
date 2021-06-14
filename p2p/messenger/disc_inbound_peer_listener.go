package messenger

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/viper"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2p/netutil"
	pr "github.com/thetatoken/theta/p2p/peer"

	gonetutil "golang.org/x/net/netutil"
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

	bootstrapNodePurgePeerTimer time.Time

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
func createInboundPeerListener(discMgr *PeerDiscoveryManager, protocol string, localAddr string, externalPort int,
	skipUPNP bool, config InboundPeerListenerConfig) (InboundPeerListener, error) {
	localAddrIP, _ := splitHostPort(localAddr)
	netListener := initiateNetListener(protocol, localAddr)
	netListenerIP, netListenerPort := splitHostPort(netListener.Addr().String())
	logger.Infof("Local network listener, ip: %v, port: %v", netListenerIP, netListenerPort)

	internalNetAddr := getInternalNetAddress(localAddr)
	externalNetAddr := getExternalNetAddress(localAddrIP, externalPort, netListenerPort, skipUPNP)

	inboundPeerListener := InboundPeerListener{
		discMgr:      discMgr,
		netListener:  netListener,
		internalAddr: internalNetAddr,
		externalAddr: externalNetAddr,
		config:       config,

		bootstrapNodePurgePeerTimer: time.Now(),
		wg:                          &sync.WaitGroup{},
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

	seedPeerOnly := viper.GetBool(common.CfgP2PSeedPeerOnly)
	maxNumPeers := GetDefaultPeerDiscoveryManagerConfig().MaxNumPeers
	logger.Infof("InboundPeerListener listen routine started, seedPeerOnly set to %v", seedPeerOnly)

	//purgeAllNonSeedPeersInterval := time.Duration(viper.GetInt(common.CfgP2PBootstrapNodePurgePeerInterval)) * time.Second
	for {

		// if viper.GetBool(common.CfgP2PIsBootstrapNode) {
		// 	now := time.Now()
		// 	if now.Sub(ipl.bootstrapNodePurgePeerTimer) > purgeAllNonSeedPeersInterval {
		// 		ipl.bootstrapNodePurgePeerTimer = now
		// 		ipl.purgeAllNonSeedPeers()
		// 	}
		// }

		netconn, err := ipl.netListener.Accept()
		if err != nil {
			logger.Fatalf("net listener error: %v", err)
		}

		remoteAddr := netutil.NewNetAddress(netconn.RemoteAddr())
		if seedPeerOnly {
			isNotASeedPeer := !ipl.discMgr.seedPeerConnector.isASeedPeerIgnoringPort(remoteAddr)
			if isNotASeedPeer {
				logger.Debugf("%v is not a seed peer, ignore inbound connection request", remoteAddr.String())
				netconn.Close()
				continue
			} else {
				logger.Infof("Accept inbound connection from seed peer %v", remoteAddr.String())
			}
		} else {
			skipEdgeNode := !viper.GetBool(common.CfgP2PIsBootstrapNode)
			numPeers := int(ipl.discMgr.peerTable.GetTotalNumPeers(skipEdgeNode))
			if numPeers >= maxNumPeers {
				if viper.GetBool(common.CfgP2PConnectionFIFO) {
					purgedPeer := ipl.discMgr.peerTable.PurgeOldestPeer()
					if purgedPeer != nil {
						purgedPeer.Stop()
						logger.Infof("Purged old peer %v to make room for inbound connection request from %v", purgedPeer.ID(), remoteAddr.String())
					}
				} else {
					logger.Debugf("Max peers limit %v reached, ignore inbound connection request from %v", maxNumPeers, remoteAddr.String())
					netconn.Close()
					continue
				}
			}
		}

		go func(netconn net.Conn) {
			peer, err := ipl.discMgr.connectWithInboundPeer(netconn, true)
			if err != nil {
				netconn.Close()
			}
			if ipl.inboundCallback != nil {
				ipl.inboundCallback(peer, err)
			}
		}(netconn)
	}
}

func (ipl *InboundPeerListener) purgeAllNonSeedPeers() {
	logger.Infof("Purge all non-seed peers")

	allPeers := ipl.discMgr.peerTable.GetAllPeers(false)
	for _, peer := range *allPeers {
		if !peer.IsSeed() {
			ipl.discMgr.peerTable.DeletePeer(peer.ID())
			peer.Stop()
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
		logger.Fatalf("failed to split host and port for: %v, err: %v", addr, err)
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		logger.Fatalf("failed to extract port for: %v, err: %v", addr, err)
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
		logger.Fatalf("Failed to initiate net listener: %v", err)
	}

	ll := gonetutil.LimitListener(netListener, viper.GetInt(common.CfgP2PMaxConnections))
	netListener = ll

	return netListener
}

func getInternalNetAddress(localAddr string) *netutil.NetAddress {
	internalAddr, err := netutil.NewNetAddressString(localAddr)
	if err != nil {
		logger.Fatalf("Failed to get internal network address: %v", err)
	}
	return internalAddr
}

func getExternalNetAddress(localAddrIP string, externalPort int, listenerPort int, skipUPNP bool) *netutil.NetAddress {
	var externalAddr *netutil.NetAddress
	if !skipUPNP {
		// If the lAddrIP is INADDR_ANY, try UPNP
		if localAddrIP == "" || localAddrIP == "0.0.0.0" {
			externalAddr = getUPNPExternalAddress(externalPort, listenerPort)
		}
	}
	// Otherwise just use the local address
	if externalAddr == nil {
		externalAddr = getNaiveExternalAddress(listenerPort)
	}
	if externalAddr == nil {
		logger.Fatalf("Could not determine external address!")
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
		logger.Fatalf("Could not fetch interface addresses: %v", err)
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
