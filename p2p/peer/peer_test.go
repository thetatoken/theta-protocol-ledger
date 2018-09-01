package peer

import (
	"crypto/ecdsa"
	"fmt"
	"net"
	"strconv"

	"github.com/thetatoken/ukulele/crypto"
	cn "github.com/thetatoken/ukulele/p2p/connection"
	nu "github.com/thetatoken/ukulele/p2p/netutil"
	p2ptypes "github.com/thetatoken/ukulele/p2p/types"
)

func getRandPubKey() ecdsa.PublicKey {
	randPrivKey, err := crypto.GenerateKey()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate a random private key: %v", err))
	}
	return randPrivKey.PublicKey
}

func getNetconn(port int) net.Conn {
	netconn, err := net.Dial("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to create a net connection: %v", err))
	}
	return netconn
}

func getListener(port int) net.Listener {
	listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		panic(fmt.Sprintf("Failed to listen: %v", err))
	}

	return listener
}

func newIncomingNetconn(port int) net.Conn {
	go func() {
		netconn := getNetconn(port)
		defer netconn.Close()
	}()

	listener := getListener(port)
	netconn, err := listener.Accept()
	if err != nil {
		panic(fmt.Sprintf("Failed to listen to the netconn: %v", err))
	}
	defer netconn.Close()

	return netconn
}

func newOutboundPeer(ipAddr string, pubKey ecdsa.PublicKey) *Peer {
	netaddr, err := nu.NewNetAddressString(ipAddr)
	if err != nil {
		panic(fmt.Sprintf("Failed to create net address: %v", err))
	}
	peerConfig := GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	outboundPeer, err := CreateOutboundPeer(netaddr, peerConfig, connConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create outbound peer: %v", err))
	}
	outboundPeer.nodeInfo = p2ptypes.CreateNodeInfo(pubKey)
	return outboundPeer
}

func newInboundPeer(netconn net.Conn, pubKey ecdsa.PublicKey) *Peer {
	peerConfig := GetDefaultPeerConfig()
	connConfig := cn.GetDefaultConnectionConfig()
	inboundPeer, err := CreateInboundPeer(netconn, peerConfig, connConfig)
	if err != nil {
		panic(fmt.Sprintf("Failed to create outbound peer: %v", err))
	}
	inboundPeer.nodeInfo = p2ptypes.CreateNodeInfo(pubKey)
	return inboundPeer
}
