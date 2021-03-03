package messenger

import (
	"context"
	"fmt"
	"io/ioutil"
	"math/rand"
	"strconv"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/util"
	"github.com/thetatoken/theta/crypto"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	p2pcmn "github.com/thetatoken/theta/p2pl/common"

	"github.com/thetatoken/theta/p2pl/peer"

	"github.com/thetatoken/theta/p2pl"
	"github.com/thetatoken/theta/p2pl/transport"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	pr "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	cr "github.com/libp2p/go-libp2p-crypto"
	peerstore "github.com/libp2p/go-libp2p-peerstore"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
	dhtopts "github.com/libp2p/go-libp2p-kad-dht/opts"
	ps "github.com/libp2p/go-libp2p-pubsub"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	// "github.com/libp2p/go-libp2p/p2p/discovery"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	ma "github.com/multiformats/go-multiaddr"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2pl"})

//
// Messenger implements the Network interface
//
var _ p2pl.Network = (*Messenger)(nil)

const (
	// thetaP2PProtocolPrefix            = "/theta/1.0.0/"
	defaultPeerDiscoveryPulseInterval = 10 * time.Second
	connectInterval                   = 1000 // 1 sec
	lowConnectivityCheckInterval      = 60
	highConnectivityCheckInterval     = 10
)

type Messenger struct {
	host          host.Host
	msgHandlerMap map[common.ChannelIDEnum](p2pl.MessageHandler)
	config        MessengerConfig
	seedPeers     map[pr.ID]*pr.AddrInfo
	pubsub        *ps.PubSub
	dht           *kaddht.IpfsDHT
	needMdns      bool
	seedPeerOnly  bool

	peerTable    *peer.PeerTable
	newPeers     chan pr.ID
	peerDead     chan pr.ID
	newPeerError chan pr.ID

	protocolPrefix string

	msgBlockBufferPool  chan []byte
	msgNormalBufferPool chan []byte

	// Stats.
	statsEnabled bool
	statsLock    sync.Mutex
	statsCounter map[common.ChannelIDEnum]uint64

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

//
// MessengerConfig specifies the configuration for Messenger
//
type MessengerConfig struct {
	networkProtocol string
}

// GetDefaultMessengerConfig returns the default config for messenger, not necessary
func GetDefaultMessengerConfig() MessengerConfig {
	return MessengerConfig{
		networkProtocol: "tcp",
	}
}

func createP2PAddr(ip, port, networkProtocol string) (ma.Multiaddr, error) {
	ipv := "ip4"
	if strings.Index(ip, ":") > 0 {
		ipv = "ip6"
	}
	multiAddr, err := ma.NewMultiaddr(fmt.Sprintf("/%v/%v/%v/%v", ipv, ip, networkProtocol, port))
	if err != nil {
		return nil, err
	}
	return multiAddr, nil
}

// ID returns the ID of the current node
func (msgr *Messenger) ID() string {
	//return string(msgr.host.ID())
	return msgr.host.ID().Pretty()
}

// CreateMessenger creates an instance of Messenger
func CreateMessenger(pubKey *crypto.PublicKey, seedPeerMultiAddresses []string,
	port int, seedPeerOnly bool, msgrConfig MessengerConfig, needMdns bool, ctx context.Context) (*Messenger, error) {

	ctx, cancel := context.WithCancel(ctx)

	pt := peer.CreatePeerTable()

	bufferPoolSize := viper.GetInt(common.CfgBufferPoolSize)

	var protocolPrefix string
	if viper.GetString(common.CfgP2PProtocolPrefix) != "" {
		protocolPrefix = viper.GetString(common.CfgP2PProtocolPrefix)
	} else {
		protocolPrefix = "/theta/" + viper.GetString(common.CfgGenesisChainID) + "/" + viper.GetString(common.CfgP2PVersion) + "/"
	}

	messenger := &Messenger{
		peerTable:           &pt,
		newPeers:            make(chan pr.ID),
		peerDead:            make(chan pr.ID),
		newPeerError:        make(chan pr.ID),
		msgBlockBufferPool:  make(chan []byte, bufferPoolSize),
		msgNormalBufferPool: make(chan []byte, bufferPoolSize),
		msgHandlerMap:       make(map[common.ChannelIDEnum](p2pl.MessageHandler)),
		needMdns:            needMdns,
		seedPeerOnly:        seedPeerOnly,
		seedPeers:           make(map[pr.ID]*pr.AddrInfo),
		protocolPrefix:      protocolPrefix,
		config:              msgrConfig,
		statsCounter:        make(map[common.ChannelIDEnum]uint64),
		wg:                  &sync.WaitGroup{},
		ctx:                 ctx,
	}

	for i := 0; i < bufferPoolSize; i++ {
		messenger.msgBlockBufferPool <- make([]byte, p2pcmn.MaxBlockMessageSize)
		messenger.msgNormalBufferPool <- make([]byte, p2pcmn.MaxNormalMessageSize)
	}

	hostId, _, err := cr.GenerateEd25519Key(strings.NewReader(common.Bytes2Hex(pubKey.ToBytes())))
	if err != nil {
		return messenger, err
	}
	localNetAddress, err := createP2PAddr("0.0.0.0", strconv.Itoa(port), msgrConfig.networkProtocol)
	if err != nil {
		return messenger, err
	}

	var extMultiAddr ma.Multiaddr
	if !seedPeerOnly {
		externalIP, err := util.GetPublicIP()
		if err != nil {
			logger.Warnf("Cannot to get the node's external IP address, use 0.0.0.0: %v", err)
			externalIP = "0.0.0.0"
			//return messenger, err
		}

		extMultiAddr, err = createP2PAddr(externalIP, strconv.Itoa(port), msgrConfig.networkProtocol)
		if err != nil {
			return messenger, err
		}
	}

	addressFactory := func(addrs []ma.Multiaddr) []ma.Multiaddr {
		if extMultiAddr != nil {
			addrs = append(addrs, extMultiAddr)
		}
		return addrs
	}

	minNumPeers := viper.GetInt(common.CfgP2PMinNumPeers)
	maxNumPeers := viper.GetInt(common.CfgP2PMaxNumPeers)
	cm := connmgr.NewConnManager(minNumPeers, maxNumPeers, defaultPeerDiscoveryPulseInterval)
	host, err := libp2p.New(
		ctx,
		libp2p.EnableRelay(),
		libp2p.Identity(hostId),
		libp2p.ListenAddrs([]ma.Multiaddr{localNetAddress}...),
		libp2p.AddrsFactory(addressFactory),
		libp2p.ConnectionManager(cm),
	)
	if err != nil {
		cancel()
		return messenger, err
	}
	messenger.host = host

	// seeds
	for _, seedPeerMultiAddrStr := range seedPeerMultiAddresses {
		addr, err := ma.NewMultiaddr(seedPeerMultiAddrStr)
		if err != nil {
			cancel()
			return messenger, err
		}
		peer, err := peerstore.InfoFromP2pAddr(addr)
		if err != nil {
			cancel()
			return messenger, err
		}
		messenger.seedPeers[peer.ID] = peer
	}

	if !seedPeerOnly {
		// kad-dht
		dopts := []dhtopts.Option{
			dhtopts.Datastore(dsync.MutexWrap(ds.NewMapDatastore())),
			dhtopts.Protocols(
				protocol.ID(messenger.protocolPrefix + "dht"),
			),
		}

		dht, err := kaddht.New(ctx, host, dopts...)
		if err != nil {
			cancel()
			return messenger, err
		}
		host = rhost.Wrap(host, dht)
		messenger.dht = dht
	}

	// pubsub
	psOpts := []ps.Option{
		ps.WithMessageSigning(false),
		ps.WithStrictSignatureVerification(false),
	}
	pubsub, err := ps.NewGossipSub(ctx, host, psOpts...)
	if err != nil {
		cancel()
		return messenger, err
	}
	messenger.pubsub = pubsub

	host.Network().Notify((*PeerNotif)(messenger))

	logger.Infof("Created node %v, %v, seedPeerOnly: %v", host.ID(), host.Addrs(), seedPeerOnly)
	return messenger, nil
}

func (msgr *Messenger) isSeedPeer(pid pr.ID) bool {
	_, isSeed := msgr.seedPeers[pid]
	return isSeed
}

func (msgr *Messenger) processLoop(ctx context.Context) {
	defer func() {
		// Clean up go routines.
		allPeers := msgr.peerTable.GetAllPeers(false) // should clean up all peers, including edge nodes
		for _, peer := range *allPeers {
			peer.Stop()
			msgr.peerTable.DeletePeer(peer.ID())
		}
		msgr.cancel()
	}()

	for {
		select {
		case pid := <-msgr.newPeers:
			if msgr.peerTable.PeerExists(pid) {
				continue
			}

			if msgr.seedPeerOnly {
				if !msgr.isSeedPeer(pid) {
					msgr.host.Network().ClosePeer(pid)
					// msgr.host.Peerstore().UpdateAddrs(pid, peerstore.ConnectedAddrTTL, time.Duration(1 * time.Millisecond))
					continue
				}
			}

			if int(msgr.peerTable.GetTotalNumPeers(true)) >= viper.GetInt(common.CfgP2PMaxNumPeers) { // only account for blockchain nodes
				msgr.host.Network().ClosePeer(pid)
				continue
			}

			pr := msgr.host.Peerstore().PeerInfo(pid)
			if pr.ID == "" {
				continue
			}
			isOutbound := strings.Compare(msgr.host.ID().String(), pid.String()) > 0
			peer := peer.CreatePeer(pr, isOutbound)
			msgr.peerTable.AddPeer(peer)
			msgr.attachHandlersToPeer(peer)
			peer.Start(msgr.ctx)
			peer.OpenStreams()
			logger.Infof("Peer connected, id: %v, addrs: %v", pr.ID, pr.Addrs)
		case pid := <-msgr.newPeerError:
			peer := msgr.peerTable.GetPeer(pid)
			if peer == nil {
				continue
			}

			peer.Stop()
			msgr.peerTable.DeletePeer(pid)
			msgr.host.Network().ClosePeer(pid)
		case pid := <-msgr.peerDead:
			peer := msgr.peerTable.GetPeer(pid)
			if peer == nil {
				continue
			}

			if msgr.host.Network().Connectedness(pid) == network.Connected {
				// still connected, must be a duplicate connection being closed.
				// we respawn the writer as we need to ensure there is a stream active
				logger.Warnf("peer declared dead but still connected, should be a duplicated connection: %v", pid)
				continue
			}

			peer.Stop()
			msgr.peerTable.DeletePeer(pid)
			logger.Infof("Peer disconnected, id: %v, addrs: %v", peer.ID(), peer.Addrs())
		case <-ctx.Done():
			log.Debug("messenger processloop shutting down")
			return
		}
	}
}

func (msgr *Messenger) maintainConnectivityRoutine(ctx context.Context) {
	var seedsConnectivityCheckPulse, sufficientConnectionsCheckPulse *time.Ticker
	if msgr.seedPeerOnly {
		seedsConnectivityCheckPulse = time.NewTicker(highConnectivityCheckInterval * time.Second)
	} else {
		seedsConnectivityCheckPulse = time.NewTicker(lowConnectivityCheckInterval * time.Second)
	}
	sufficientConnectionsCheckPulse = time.NewTicker(lowConnectivityCheckInterval * time.Second)

	for {
		select {
		case <-seedsConnectivityCheckPulse.C:
			msgr.maintainSeedsConnectivity(ctx)
		case <-sufficientConnectionsCheckPulse.C:
			msgr.maintainSufficientConnections(ctx)
		}
	}
}

func (msgr *Messenger) maintainSeedsConnectivity(ctx context.Context) {
	if !msgr.seedPeerOnly {
		for _, pid := range *(msgr.peerTable.GetAllPeerIDs()) {
			if msgr.isSeedPeer(pid) {
				// don't proceed if there's at least one seed in peer table
				return
			}
		}
	}

	seedPeers := make([]*pr.AddrInfo, 0, len(msgr.seedPeers))
	for _, seedPeer := range msgr.seedPeers {
		seedPeers = append(seedPeers, seedPeer)
	}

	perm := rand.Perm(len(seedPeers))
	for _, idx := range perm {
		time.Sleep(time.Duration(rand.Int63n(connectInterval)) * time.Millisecond)
		seedPeer := seedPeers[idx]
		peer := msgr.peerTable.GetPeer(seedPeer.ID)
		if peer == nil { // if peer is not in peer table, then connect
			msgr.wg.Add(1)
			go func() {
				defer msgr.wg.Done()
				err := msgr.host.Connect(ctx, *seedPeer)
				if err == nil {
					logger.Infof("Successfully re-connected to seed peer: %v", seedPeer)
				} else {
					logger.Warnf("Failed to re-connect to seed peer %v, %v", seedPeer, err)
				}
			}()
		}
		if !msgr.seedPeerOnly {
			break // if not seed peer only, sufficient to have at least one connection
		}
	}
}

func (msgr *Messenger) maintainSufficientConnections(ctx context.Context) {
	diff := viper.GetInt(common.CfgP2PMinNumPeers) - int(msgr.peerTable.GetTotalNumPeers(true)) // only account for blockchain nodes
	if diff > 0 {
		var connections []*pr.AddrInfo
		for _, seed := range msgr.seedPeers {
			if !msgr.peerTable.PeerExists(seed.ID) {
				connections = append(connections, seed)
			}
		}
		if !msgr.seedPeerOnly {
			prevPeers, err := msgr.peerTable.RetrievePreviousPeers()
			if err == nil {
				for _, prevPeer := range prevPeers {
					if msgr.peerTable.PeerExists(prevPeer.ID) {
						continue
					}

					exists := false
					for _, seed := range connections {
						if seed.ID == prevPeer.ID {
							exists = true
							break
						}
					}
					if !exists {
						connections = append(connections, prevPeer)
					}
				}
			}
		}

		if len(connections) > 0 {
			perm := rand.Perm(len(connections))
			msgr.wg.Add(1)
			go func(i int) {
				defer msgr.wg.Done()
				j := perm[i]
				peer := connections[j]
				err := msgr.host.Connect(ctx, *peer)
				if err == nil {
					logger.Infof("Successfully re-connected to peer: %v", peer)
				} else {
					logger.Warnf("Failed to re-connect to peer %v, %v", peer, err)
				}
			}(perm[0])
		}
	}
}

// Start is called when the Messenger starts
func (msgr *Messenger) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	msgr.ctx = c
	msgr.cancel = cancel

	// seeds & previously persisted peers
	connections := make([]*pr.AddrInfo, 0)
	for _, seed := range msgr.seedPeers {
		connections = append(connections, seed)
	}
	if !msgr.seedPeerOnly {
		prevPeers, err := msgr.peerTable.RetrievePreviousPeers()
		if err == nil {
			for _, prevPeer := range prevPeers {
				exists := false
				for _, seed := range connections {
					if seed.ID == prevPeer.ID {
						exists = true
						break
					}
				}
				if !exists {
					connections = append(connections, prevPeer)
				}
			}
		}
	}

	logger.Infof("Connecting to: %v", connections)

	perm := rand.Perm(len(connections))
	for i := 0; i < len(perm); i++ { // create outbound peers in a random order
		time.Sleep(time.Duration(rand.Int63n(connectInterval)) * time.Millisecond)

		msgr.wg.Add(1)
		go func(i int) {
			defer msgr.wg.Done()

			j := perm[i]
			seedPeer := connections[j]
			err := msgr.host.Connect(ctx, *seedPeer)
			if err != nil {
				logger.Warnf("Failed to connect to peer %v: %v. connectedness: %v", seedPeer, err, msgr.host.Network().Connectedness(seedPeer.ID))
			}
		}(i)
	}

	// kad-dht
	if msgr.dht != nil {
		bcfg := kaddht.DefaultBootstrapConfig
		bcfg.Period = time.Duration(defaultPeerDiscoveryPulseInterval)
		if err := msgr.dht.BootstrapWithConfig(ctx, bcfg); err != nil {
			logger.Errorf("Failed to bootstrap DHT: %v", err)
		}
	}

	// // mDns
	// if msgr.needMdns {
	// 	mdnsService, err := discovery.NewMdnsService(ctx, msgr.host, defaultPeerDiscoveryPulseInterval, viper.GetString(common.CfgLibP2PRendezvous))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	mdnsService.RegisterNotifee(&discoveryNotifee{ctx, msgr.host})
	// }

	go msgr.processLoop(ctx)
	go msgr.maintainConnectivityRoutine(ctx)

	msgr.statsEnabled = viper.GetBool(common.CfgProfEnabled)
	if msgr.statsEnabled {
		go func() {
			t := time.NewTicker(3 * time.Second)

			for {
				<-t.C
				msgr.printStats()
			}
		}()
	}

	return nil
}

// Stop is called when the Messenger stops
func (msgr *Messenger) Stop() {
	if msgr.host.Peerstore() != nil && msgr.host.Peerstore().Peers() != nil {
		for _, pid := range msgr.host.Peerstore().Peers() {
			msgr.host.Network().ClosePeer(pid)
		}
	}

	msgr.cancel()
	logger.Infof("Messenger shut down %v", msgr.host.ID())
}

// Wait suspends the caller goroutine
func (msgr *Messenger) Wait() {
	msgr.wg.Wait()
}

// Publish publishes the given message to all the subscribers
func (msgr *Messenger) Publish(message p2ptypes.Message) error {
	logger.Debugf("Publishing messages...")

	msgHandler := msgr.msgHandlerMap[message.ChannelID]
	bytes, err := msgHandler.EncodeMessage(message.Content)
	if err != nil {
		logger.Errorf("Encoding error: %v", err)
		return err
	}

	err = msgr.pubsub.Publish(msgr.protocolPrefix+strconv.Itoa(int(message.ChannelID)), bytes)
	if err != nil {
		log.Errorf("Failed to publish to gossipsub topic: %v", err)
		return err
	}

	return nil
}

// Broadcast broadcasts the given message to all the connected peers
func (msgr *Messenger) Broadcast(message p2ptypes.Message, skipEdgeNode bool) (successes chan bool) {
	// TODO: support skipEdgeNode
	logger.Debugf("Broadcasting messages...")
	msgr.Publish(message)
	return make(chan bool)
}

// BroadcastToNeighbors broadcasts the given message to neighbors
func (msgr *Messenger) BroadcastToNeighbors(message p2ptypes.Message, maxNumPeersToBroadcast int, skipEdgeNode bool) (successes chan bool) {
	// TODO: support skipEdgeNode
	sampledPIDs := msgr.samplePeers(maxNumPeersToBroadcast, skipEdgeNode)
	for _, pid := range sampledPIDs {
		go func(pid string) {
			msgr.Send(pid, message)
		}(pid)
	}
	return make(chan bool)
}

// samplePeers randomly sample a subset of peers
func (msgr *Messenger) samplePeers(maxNumSampledPeers int, skipEdgeNode bool) []string {
	// TODO: support skipEdgeNode

	// Prioritize seed peers
	sampledPIDs, idx := []string{}, 0
	for seedPID := range msgr.seedPeers {
		// Note: the order of map loop-through is undeterminstic, which effectively shuffles the seed peers
		sampledPIDs = append(sampledPIDs, seedPID.String())
		idx++
		if idx >= maxNumSampledPeers {
			return sampledPIDs
		}
	}

	// Randomly sample the remaining peers
	neighbors := *msgr.peerTable.GetAllPeers(skipEdgeNode)
	neighborPIDs := []string{}
	for _, peer := range neighbors {
		pid := peer.ID()
		if pid == msgr.host.ID() || msgr.isSeedPeer(pid) {
			continue
		}
		neighborPIDs = append(neighborPIDs, pid.String())
	}

	numPeersToSample := maxNumSampledPeers - len(msgr.seedPeers) // numPeersToSample is guaranteed > 0
	sampledNeighbors := util.Sample(neighborPIDs, numPeersToSample)
	if numPeersToSample >= len(sampledNeighbors) {
		numPeersToSample = len(sampledNeighbors)
	}

	for i := 0; i < numPeersToSample; i++ {
		sampledPIDs = append(sampledPIDs, sampledNeighbors[i])
	}

	return sampledPIDs
}

// Send sends the given message to the specified peer
func (msgr *Messenger) Send(peerID string, message p2ptypes.Message) bool {
	prID, err := pr.IDB58Decode(peerID)
	if err != nil {
		return false
	}
	peer := msgr.peerTable.GetPeer(prID)
	if peer == nil {
		return false
	}

	success := peer.Send(message.ChannelID, message.Content)
	return success
}

// Peers returns the IDs of all peers
func (msgr *Messenger) Peers(skipEdgeNode bool) []string {
	// TODO: support skipEdgeNode
	allPeers := msgr.peerTable.GetAllPeers(skipEdgeNode)
	peerIDs := []string{}
	for _, peer := range *allPeers {
		peerID := peer.ID().Pretty()
		peerIDs = append(peerIDs, peerID)
	}
	return peerIDs
}

// PeerURLs returns the URLs of all peers
func (msgr *Messenger) PeerURLs(skipEdgeNode bool) []string {
	allPeers := msgr.peerTable.GetAllPeers(skipEdgeNode)
	peerURLs := []string{}
	for _, peer := range *allPeers {
		peerURLs = append(peerURLs, peer.AddrInfo().String())
	}
	return peerURLs
}

// PeerExists indicates if the given peerID is a neighboring peer
func (msgr *Messenger) PeerExists(peerID string) bool {
	prID, err := pr.IDB58Decode(peerID)
	if err != nil {
		return false
	}
	return msgr.peerTable.PeerExists(prID)
}

func (msgr *Messenger) recordReceivedBytes(cid common.ChannelIDEnum, size int) {
	if !msgr.statsEnabled {
		return
	}

	msgr.statsLock.Lock()
	defer msgr.statsLock.Unlock()

	old, ok := msgr.statsCounter[cid]
	if ok {
		msgr.statsCounter[cid] = old + uint64(size)
	} else {
		msgr.statsCounter[cid] = uint64(size)
	}
}

func (msgr *Messenger) printStats() {
	msgr.statsLock.Lock()
	defer msgr.statsLock.Unlock()

	ret := "Received bytes:"
	for k := byte(0); k <= byte(common.ChannelIDAggregatedEliteEdgeNodeVotes); k++ {
		v, ok := msgr.statsCounter[common.ChannelIDEnum(k)]
		if !ok {
			continue
		}
		ret += fmt.Sprintf(" channel %v: %.3f MB\t", k, util.BToMb(v))
	}
	logger.Debug(ret)
}

// RegisterMessageHandler registers the message handler
func (msgr *Messenger) RegisterMessageHandler(msgHandler p2pl.MessageHandler) {
	channelIDs := msgHandler.GetChannelIDs()
	for _, channelID := range channelIDs {
		if msgr.msgHandlerMap[channelID] != nil {
			logger.Errorf("Message handler is already added for channelID: %v", channelID)
			return
		}
		msgr.msgHandlerMap[channelID] = msgHandler

		msgr.registerStreamHandler(channelID)

		sub, err := msgr.pubsub.Subscribe(msgr.protocolPrefix + strconv.Itoa(int(channelID)))
		if err != nil {
			logger.Errorf("Failed to subscribe to channel %v, %v", channelID, err)
			continue
		}
		go func(channelID common.ChannelIDEnum) {
			defer sub.Cancel()

			var msg *ps.Message
			var err error

			for {
				msg, err = sub.Next(context.Background())

				if msgr.ctx != nil && msgr.ctx.Err() != nil {
					logger.Errorf("Context error %v", msgr.ctx.Err())
					return
				}
				if err != nil {
					logger.Errorf("Failed to get next message: %v", err)
					continue
				}

				if msg == nil || msg.GetFrom() == msgr.host.ID() {
					continue
				}

				message, err := msgHandler.ParseMessage(msg.GetFrom().String(), channelID, msg.Data)
				if err != nil {
					logger.Errorf("Failed to parse message, %v", err)
					return
				}

				msgr.recordReceivedBytes(channelID, len(msg.Data))

				msgHandler.HandleMessage(message)
			}
		}(channelID)
	}
}

func (msgr *Messenger) registerStreamHandler(channelID common.ChannelIDEnum) {
	logger.Debugf("Registered stream handler for channel %v", channelID)
	msgr.host.SetStreamHandler(protocol.ID(msgr.protocolPrefix+strconv.Itoa(int(channelID))), func(strm network.Stream) {
		peerID := strm.Conn().RemotePeer()

		if msgr.seedPeerOnly {
			if !msgr.isSeedPeer(peerID) {
				msgr.host.Network().ClosePeer(peerID)
				return
			}
		}

		if strings.Compare(msgr.host.ID().String(), peerID.String()) > 0 {
			logger.Warnf("Received stream from an outbound peer")
			return
		}

		remotePeer := msgr.peerTable.GetPeer(peerID)
		if remotePeer == nil {
			var addrInfo pr.AddrInfo
			addrInfo.ID = peerID
			addrInfo.Addrs = append(addrInfo.Addrs, strm.Conn().RemoteMultiaddr())
			remotePeer = peer.CreatePeer(addrInfo, true)
			msgr.peerTable.AddPeer(remotePeer)
			msgr.attachHandlersToPeer(remotePeer)
			remotePeer.Start(msgr.ctx)

			logger.Infof("Peer connected (via stream), id: %v, addrs: %v", remotePeer.ID, remotePeer.Addrs)
		}

		reuseStream := viper.GetBool(common.CfgP2PReuseStream)
		if reuseStream {
			errorHandler := func(interface{}) {
				remotePeer.StopStream(channelID)
			}
			stream := transport.NewBufferedStream(strm, errorHandler)
			stream.Start(msgr.ctx)
			go msgr.readPeerMessageRoutine(stream, peerID.String(), channelID)
			remotePeer.AcceptStream(channelID, stream)

		} else {
			rawPeerMsg, err := ioutil.ReadAll(strm)
			if err != nil {
				logger.Warnf("Failed to read stream, %v. channel: %v, peer: %v", err, channelID, peerID)
				return
			}
			msgHandler := msgr.msgHandlerMap[channelID]
			message, err := msgHandler.ParseMessage(peerID.String(), channelID, rawPeerMsg)
			if err != nil {
				logger.Errorf("Failed to parse message, %v. len(): %v, channel: %v, peer: %v, msg: %v", err, len(rawPeerMsg), channelID, peerID, rawPeerMsg)
				return
			}

			msgr.recordReceivedBytes(channelID, len(rawPeerMsg))

			msgHandler.HandleMessage(message)
		}
	})
}

func (msgr *Messenger) readPeerMessageRoutine(stream *transport.BufferedStream, peerID string, channelID common.ChannelIDEnum) {
	defer stream.Stop()

	for {
		if msgr.ctx != nil {
			select {
			case <-msgr.ctx.Done():
				return
			default:
			}
		}

		var msgBuffer []byte
		var bufferSize int
		var bufferPool chan []byte
		if channelID == common.ChannelIDBlock || channelID == common.ChannelIDProposal {
			bufferSize = p2pcmn.MaxBlockMessageSize
			bufferPool = msgr.msgBlockBufferPool
		} else {
			bufferSize = p2pcmn.MaxNormalMessageSize
			bufferPool = msgr.msgNormalBufferPool
		}

		msgBuffer, msgSize, err := stream.Read(bufferPool)
		if err != nil {
			logger.Warnf("Failed to read stream: %v", err)
			if msgBuffer != nil {
				bufferPool <- msgBuffer
			}
			return
		}

		if msgBuffer == nil {
			// Should not happen
			logger.Panic("msgBuffer cannot be nil")
		}
		if msgSize > bufferSize {
			logger.Errorf("Message ignored since it exceeds the peer message size limit, size: %v", msgSize)
			bufferPool <- msgBuffer
			continue
		}

		rawPeerMsg := msgBuffer[:msgSize]

		msgHandler := msgr.msgHandlerMap[channelID]
		message, err := msgHandler.ParseMessage(peerID, channelID, rawPeerMsg)
		bufferPool <- msgBuffer
		if err != nil {
			logger.Errorf("Failed to parse message, %v. msgSize: %v, len(): %v, channel: %v, peer: %v, msg: %v", err, msgSize, len(rawPeerMsg), channelID, peerID, rawPeerMsg)
			return
		}

		msgr.recordReceivedBytes(channelID, len(rawPeerMsg))

		msgHandler.HandleMessage(message)
	}
}

// attachHandlersToPeer attaches the registerred message/stream handlers to the given peer
func (msgr *Messenger) attachHandlersToPeer(peer *peer.Peer) {
	messageParser := func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
		peerID := peer.ID()
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			logger.Errorf("Failed to setup message parser for channelID %v", channelID)
		}
		message, err := msgHandler.ParseMessage(peerID.String(), channelID, rawMessageBytes)

		msgr.recordReceivedBytes(channelID, len(rawMessageBytes))

		return message, err
	}
	peer.SetMessageParser(messageParser)

	messageEncoder := func(channelID common.ChannelIDEnum, message interface{}) (common.Bytes, error) {
		msgHandler := msgr.msgHandlerMap[channelID]
		return msgHandler.EncodeMessage(message)
	}
	peer.SetMessageEncoder(messageEncoder)

	receiveHandler := func(message p2ptypes.Message) error {
		channelID := message.ChannelID
		msgHandler := msgr.msgHandlerMap[channelID]
		if msgHandler == nil {
			logger.Errorf("Failed to setup message handler for peer %v on channelID %v", message.PeerID, channelID)
		}
		err := msgHandler.HandleMessage(message)
		return err
	}
	peer.SetReceiveHandler(receiveHandler)

	streamCreator := func(channelID common.ChannelIDEnum) (*transport.BufferedStream, error) {
		strm, err := msgr.host.NewStream(msgr.ctx, peer.ID(), protocol.ID(msgr.protocolPrefix+strconv.Itoa(int(channelID))))
		if err != nil {
			logger.Debugf("Stream open failed: %v. peer: %v, addrs: %v", err, peer.ID(), peer.Addrs())
			return nil, err
		}
		if strm == nil {
			logger.Errorf("Can't open stream. peer: %v, addrs: %v", peer.ID(), peer.Addrs())
			return nil, nil
		}

		errorHandler := func(interface{}) {
			peer.StopStream(channelID)
		}
		stream := transport.NewBufferedStream(strm, errorHandler)
		stream.Start(msgr.ctx)
		go msgr.readPeerMessageRoutine(stream, peer.ID().String(), channelID)
		return stream, nil
	}
	peer.SetStreamCreator(streamCreator)

	rawStreamCreator := func(channelID common.ChannelIDEnum) (network.Stream, error) {
		stream, err := msgr.host.NewStream(msgr.ctx, peer.ID(), protocol.ID(msgr.protocolPrefix+strconv.Itoa(int(channelID))))
		if err != nil {
			logger.Debugf("Stream open failed: %v. peer: %v, addrs: %v", err, peer.ID(), peer.Addrs())
			return nil, err
		}
		if stream == nil {
			logger.Errorf("Can't open stream. peer: %v, addrs: %v", peer.ID(), peer.Addrs())
			return nil, nil
		}

		return stream, nil
	}
	peer.SetRawStreamCreator(rawStreamCreator)
}
