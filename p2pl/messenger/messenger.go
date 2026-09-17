package messenger

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
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

// Messenger implements the Network interface
var _ p2pl.Network = (*Messenger)(nil)

const (
	// thetaP2PProtocolPrefix            = "/theta/1.0.0/"
	defaultPeerDiscoveryPulseInterval = 10 * time.Second
	connectInterval                   = 1000 // 1 sec
	lowConnectivityCheckInterval      = 60
	highConnectivityCheckInterval     = 10
	maxInboundStreamsPerPeer          = 16
	maxInboundReservedBytesPerPeer    = 3 * common.MaxBlockMessageSize
	maxPenalizedPeers                 = 4096
	malformedPeerPenaltyDuration      = time.Minute
	nonReusableStreamReadTimeout      = 10 * time.Second
)

var errP2PMessageTooLarge = errors.New("p2p message exceeds size limit")

type inboundStreamUsage struct {
	count         int
	reservedBytes int
	channels      map[common.ChannelIDEnum]int
}

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

	securityLock       sync.Mutex
	inboundStreamUsage map[pr.ID]*inboundStreamUsage
	penalizedPeers     map[pr.ID]time.Time

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// MessengerConfig specifies the configuration for Messenger
type MessengerConfig struct {
	networkProtocol        string
	disablePeerPersistence bool
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

	var pt peer.PeerTable
	if msgrConfig.disablePeerPersistence {
		pt = peer.CreateInMemoryPeerTable()
	} else {
		pt = peer.CreatePeerTable()
	}

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
		inboundStreamUsage:  make(map[pr.ID]*inboundStreamUsage),
		penalizedPeers:      make(map[pr.ID]time.Time),
		wg:                  &sync.WaitGroup{},
		ctx:                 ctx,
		cancel:              cancel,
	}

	for i := 0; i < bufferPoolSize; i++ {
		messenger.msgBlockBufferPool <- make([]byte, p2pcmn.MaxBlockMessageSize)
		messenger.msgNormalBufferPool <- make([]byte, p2pcmn.MaxNormalMessageSize)
	}

	hostId, _, err := cr.GenerateEd25519Key(strings.NewReader(common.Bytes2Hex(pubKey.ToBytes())))
	if err != nil {
		cancel()
		return messenger, err
	}
	localNetAddress, err := createP2PAddr("0.0.0.0", strconv.Itoa(port), msgrConfig.networkProtocol)
	if err != nil {
		cancel()
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
			cancel()
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

func (msgr *Messenger) IsSeedPeer(pid string) bool {
	_, isSeed := msgr.seedPeers[pr.ID(pid)]
	return isSeed
}

func (msgr *Messenger) acquireInboundStream(pid pr.ID, channelID common.ChannelIDEnum, reuseStream bool) bool {
	reservedBytes := common.MaxP2PMessageSize(channelID)

	msgr.securityLock.Lock()
	defer msgr.securityLock.Unlock()

	usage := msgr.inboundStreamUsage[pid]
	if usage == nil {
		usage = &inboundStreamUsage{channels: make(map[common.ChannelIDEnum]int)}
		msgr.inboundStreamUsage[pid] = usage
	}
	if reuseStream && usage.channels[channelID] > 0 {
		return false
	}
	if usage.count >= maxInboundStreamsPerPeer ||
		usage.reservedBytes > maxInboundReservedBytesPerPeer-reservedBytes {
		return false
	}

	usage.count++
	usage.reservedBytes += reservedBytes
	usage.channels[channelID]++
	return true
}

func (msgr *Messenger) releaseInboundStream(pid pr.ID, channelID common.ChannelIDEnum) {
	reservedBytes := common.MaxP2PMessageSize(channelID)

	msgr.securityLock.Lock()
	defer msgr.securityLock.Unlock()

	usage := msgr.inboundStreamUsage[pid]
	if usage == nil || usage.channels[channelID] == 0 {
		return
	}
	usage.count--
	usage.reservedBytes -= reservedBytes
	usage.channels[channelID]--
	if usage.channels[channelID] == 0 {
		delete(usage.channels, channelID)
	}
	if usage.count == 0 {
		delete(msgr.inboundStreamUsage, pid)
	}
}

func (msgr *Messenger) penalizeMalformedPeer(pid pr.ID, reason interface{}) {
	penaltyExpiresAt := msgr.recordMalformedPeerPenalty(pid, time.Now())

	logger.Warnf("Temporarily isolating peer %v after malformed P2P input until %v: %v",
		pid, penaltyExpiresAt, reason)
	if remotePeer := msgr.peerTable.GetPeer(pid); remotePeer != nil {
		remotePeer.Stop()
		msgr.peerTable.DeletePeer(pid)
	}
	_ = msgr.host.Network().ClosePeer(pid)
}

func (msgr *Messenger) recordMalformedPeerPenalty(pid pr.ID, now time.Time) time.Time {
	penaltyExpiresAt := now.Add(malformedPeerPenaltyDuration)

	msgr.securityLock.Lock()
	defer msgr.securityLock.Unlock()
	if msgr.penalizedPeers == nil {
		msgr.penalizedPeers = make(map[pr.ID]time.Time)
	}
	for penalizedPID, expiresAt := range msgr.penalizedPeers {
		if !now.Before(expiresAt) {
			delete(msgr.penalizedPeers, penalizedPID)
		}
	}
	if _, exists := msgr.penalizedPeers[pid]; !exists && len(msgr.penalizedPeers) >= maxPenalizedPeers {
		var earliestPID pr.ID
		var earliestExpiry time.Time
		for penalizedPID, expiresAt := range msgr.penalizedPeers {
			if earliestExpiry.IsZero() || expiresAt.Before(earliestExpiry) {
				earliestPID = penalizedPID
				earliestExpiry = expiresAt
			}
		}
		delete(msgr.penalizedPeers, earliestPID)
	}
	msgr.penalizedPeers[pid] = penaltyExpiresAt
	return penaltyExpiresAt
}

func (msgr *Messenger) isPeerPenalized(pid pr.ID, now time.Time) bool {
	msgr.securityLock.Lock()
	defer msgr.securityLock.Unlock()

	penaltyExpiresAt, ok := msgr.penalizedPeers[pid]
	if !ok {
		return false
	}
	if !now.Before(penaltyExpiresAt) {
		delete(msgr.penalizedPeers, pid)
		return false
	}
	return true
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
			if msgr.isPeerPenalized(pid, time.Now()) {
				_ = msgr.host.Network().ClosePeer(pid)
				continue
			}
			if msgr.peerTable.PeerExists(pid) {
				continue
			}

			if msgr.seedPeerOnly {
				if !msgr.IsSeedPeer(string(pid)) {
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
			msgr.attachHandlersToPeer(peer)
			peer.Start(msgr.ctx)
			msgr.peerTable.AddPeer(peer)
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
	defer seedsConnectivityCheckPulse.Stop()
	defer sufficientConnectionsCheckPulse.Stop()

	for {
		select {
		case <-seedsConnectivityCheckPulse.C:
			msgr.maintainSeedsConnectivity(ctx)
		case <-sufficientConnectionsCheckPulse.C:
			msgr.maintainSufficientConnections(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (msgr *Messenger) maintainSeedsConnectivity(ctx context.Context) {
	if !msgr.seedPeerOnly {
		for _, pid := range *(msgr.peerTable.GetAllPeerIDs()) {
			if msgr.IsSeedPeer(string(pid)) {
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
	if ctx != nil {
		go func(startCtx context.Context) {
			select {
			case <-startCtx.Done():
				msgr.cancel()
			case <-msgr.ctx.Done():
			}
		}(ctx)
	}

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
			err := msgr.host.Connect(msgr.ctx, *seedPeer)
			if err != nil {
				logger.Warnf("Failed to connect to peer %v: %v. connectedness: %v", seedPeer, err, msgr.host.Network().Connectedness(seedPeer.ID))
			}
		}(i)
	}

	// kad-dht
	if msgr.dht != nil {
		bcfg := kaddht.DefaultBootstrapConfig
		bcfg.Period = time.Duration(defaultPeerDiscoveryPulseInterval)
		if err := msgr.dht.BootstrapWithConfig(msgr.ctx, bcfg); err != nil {
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

	go msgr.processLoop(msgr.ctx)
	go msgr.maintainConnectivityRoutine(msgr.ctx)

	msgr.statsEnabled = viper.GetBool(common.CfgProfEnabled)
	if msgr.statsEnabled {
		go func(ctx context.Context) {
			t := time.NewTicker(3 * time.Second)
			defer t.Stop()

			for {
				select {
				case <-t.C:
					msgr.printStats()
				case <-ctx.Done():
					return
				}
			}
		}(msgr.ctx)
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
	if !common.IsP2PMessageSizeAllowed(message.ChannelID, len(bytes)) {
		return fmt.Errorf("refusing to publish oversized message, size: %v, limit: %v, channel: %v",
			len(bytes), common.MaxP2PMessageSize(message.ChannelID), message.ChannelID)
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
		if pid == msgr.host.ID() || msgr.IsSeedPeer(string(pid)) {
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
				maxMessageSize := common.MaxP2PMessageSize(channelID)
				if len(msg.Data) > maxMessageSize {
					logger.Errorf("Pubsub message ignored since it exceeds the peer message size limit, size: %v, limit: %v, channel: %v, peer: %v",
						len(msg.Data), maxMessageSize, channelID, msg.GetFrom())
					continue
				}

				message, err := msgHandler.ParseMessage(msg.GetFrom().String(), channelID, msg.Data)
				if err != nil {
					logger.Errorf("Failed to parse message, %v", err)
					continue
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
		if msgr.isPeerPenalized(peerID, time.Now()) {
			_ = strm.Reset()
			_ = msgr.host.Network().ClosePeer(peerID)
			return
		}

		if msgr.seedPeerOnly {
			if !msgr.IsSeedPeer(string(peerID)) {
				_ = strm.Reset()
				_ = msgr.host.Network().ClosePeer(peerID)
				return
			}
		}

		reuseStream := viper.GetBool(common.CfgP2PReuseStream)
		if reuseStream && strings.Compare(msgr.host.ID().String(), peerID.String()) > 0 {
			logger.Warnf("Received stream from an outbound peer")
			_ = strm.Reset()
			return
		}

		if !msgr.acquireInboundStream(peerID, channelID, reuseStream) {
			logger.Warnf("Rejected excess inbound stream for channel %v from peer %v", channelID, peerID)
			_ = strm.Reset()
			return
		}

		remotePeer := msgr.peerTable.GetPeer(peerID)
		if remotePeer == nil {
			var addrInfo pr.AddrInfo
			addrInfo.ID = peerID
			addrInfo.Addrs = append(addrInfo.Addrs, strm.Conn().RemoteMultiaddr())
			remotePeer = peer.CreatePeer(addrInfo, false)
			msgr.attachHandlersToPeer(remotePeer)
			remotePeer.Start(msgr.ctx)
			msgr.peerTable.AddPeer(remotePeer)

			logger.Infof("Peer connected (via stream), id: %v, addrs: %v", remotePeer.ID(), remotePeer.Addrs())
		}

		if reuseStream {
			errorHandler := func(reason interface{}) {
				msgr.penalizeMalformedPeer(peerID, reason)
			}
			stream := transport.NewBufferedStreamWithMaxMessageSize(
				strm, errorHandler, common.MaxP2PMessageSize(channelID))
			remotePeer.AcceptStream(channelID, stream)
			stream.Start(msgr.ctx)
			go func() {
				defer msgr.releaseInboundStream(peerID, channelID)
				msgr.readPeerMessageRoutine(stream, peerID, channelID)
			}()

		} else {
			defer msgr.releaseInboundStream(peerID, channelID)
			defer strm.Close()
			rawPeerMsg, err := readAllWithProgressDeadline(strm,
				common.MaxP2PMessageSize(channelID), nonReusableStreamReadTimeout)
			if err != nil {
				logger.Warnf("Failed to read stream, %v. channel: %v, peer: %v", err, channelID, peerID)
				if isProtocolReadError(err) {
					msgr.penalizeMalformedPeer(peerID, err)
				}
				return
			}
			msgHandler := msgr.msgHandlerMap[channelID]
			message, err := msgHandler.ParseMessage(peerID.String(), channelID, rawPeerMsg)
			if err != nil {
				logger.Errorf("Failed to parse message, %v. len(): %v, channel: %v, peer: %v", err, len(rawPeerMsg), channelID, peerID)
				msgr.penalizeMalformedPeer(peerID, err)
				return
			}

			msgr.recordReceivedBytes(channelID, len(rawPeerMsg))

			msgHandler.HandleMessage(message)
		}
	})
}

func (msgr *Messenger) readPeerMessageRoutine(stream *transport.BufferedStream, peerID pr.ID, channelID common.ChannelIDEnum) {
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
			bufferSize = common.MaxBlockMessageSize
			bufferPool = msgr.msgBlockBufferPool
		} else {
			bufferSize = common.MaxNormalMessageSize
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
		message, err := msgHandler.ParseMessage(peerID.String(), channelID, rawPeerMsg)
		bufferPool <- msgBuffer
		if err != nil {
			logger.Errorf("Failed to parse message, %v. msgSize: %v, len(): %v, channel: %v, peer: %v", err, msgSize, len(rawPeerMsg), channelID, peerID)
			msgr.penalizeMalformedPeer(peerID, err)
			return
		}

		msgr.recordReceivedBytes(channelID, len(rawPeerMsg))

		msgHandler.HandleMessage(message)
	}
}

func readAllWithLimit(reader io.Reader, limit int) ([]byte, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("invalid read limit %v", limit)
	}
	limitedReader := io.LimitReader(reader, int64(limit)+1)
	data, err := ioutil.ReadAll(limitedReader)
	if err != nil {
		return nil, err
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%w, size: %v, limit: %v", errP2PMessageTooLarge, len(data), limit)
	}
	return data, nil
}

type readDeadlineReader interface {
	io.Reader
	SetReadDeadline(time.Time) error
}

type progressDeadlineReader struct {
	reader            readDeadlineReader
	maxMessageSize    int
	inactivityTimeout time.Duration
	startedAt         time.Time
	lastProgressAt    time.Time
}

func (reader *progressDeadlineReader) Read(buffer []byte) (int, error) {
	now := time.Now()
	if reader.startedAt.IsZero() {
		reader.startedAt = now
		reader.lastProgressAt = now
	}
	deadline := common.P2PReassemblyDeadline(reader.startedAt, reader.lastProgressAt,
		reader.maxMessageSize, reader.inactivityTimeout)
	if err := reader.reader.SetReadDeadline(deadline); err != nil {
		return 0, fmt.Errorf("failed to set stream progress deadline: %w", err)
	}
	n, err := reader.reader.Read(buffer)
	if n > 0 {
		reader.lastProgressAt = time.Now()
	}
	return n, err
}

func readAllWithProgressDeadline(reader readDeadlineReader, limit int,
	inactivityTimeout time.Duration) ([]byte, error) {
	return readAllWithLimit(&progressDeadlineReader{
		reader:            reader,
		maxMessageSize:    limit,
		inactivityTimeout: inactivityTimeout,
	}, limit)
}

func isProtocolReadError(err error) bool {
	if errors.Is(err, errP2PMessageTooLarge) {
		return true
	}
	netErr, ok := err.(net.Error)
	return ok && netErr.Timeout()
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

		errorHandler := func(reason interface{}) {
			msgr.penalizeMalformedPeer(peer.ID(), reason)
		}
		stream := transport.NewBufferedStreamWithMaxMessageSize(
			strm, errorHandler, common.MaxP2PMessageSize(channelID))
		stream.Start(msgr.ctx)
		go msgr.readPeerMessageRoutine(stream, peer.ID(), channelID)
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
