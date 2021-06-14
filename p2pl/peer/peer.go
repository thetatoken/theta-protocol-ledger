package peer

import (
	"context"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	cmn "github.com/thetatoken/theta/common"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/p2pl/transport"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/thetatoken/theta/rlp"

	pr "github.com/libp2p/go-libp2p-core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2pl"})

var Channels = []cmn.ChannelIDEnum{
	cmn.ChannelIDCheckpoint,
	cmn.ChannelIDHeader,
	cmn.ChannelIDBlock,
	cmn.ChannelIDProposal,
	cmn.ChannelIDVote,
	cmn.ChannelIDTransaction,
	cmn.ChannelIDPeerDiscovery,
	cmn.ChannelIDPing,
	cmn.ChannelIDGuardian,
	cmn.ChannelIDEliteEdgeNodeVote,
	cmn.ChannelIDAggregatedEliteEdgeNodeVotes,
}

//
// Peer models a peer node in a network
//
type Peer struct {
	addrInfo   pr.AddrInfo
	isOutbound bool
	streamMap  map[cmn.ChannelIDEnum](*transport.BufferedStream) // channelID -> stream
	mutex      *sync.Mutex

	openStreamsTimer *time.Timer

	onStream    StreamCreator
	onRawStream RawStreamCreator
	onParse     MessageParser
	onEncode    MessageEncoder
	onReceive   ReceiveHandler
	onError     ErrorHandler

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

func CreatePeer(addrInfo pr.AddrInfo, isOutbound bool) *Peer {
	peer := &Peer{
		addrInfo:   addrInfo,
		isOutbound: isOutbound,
		streamMap:  make(map[cmn.ChannelIDEnum](*transport.BufferedStream)),
		mutex:      &sync.Mutex{},
		onEncode:   defaultMessageEncoder,
		wg:         &sync.WaitGroup{},
	}

	return peer
}

func (peer *Peer) OpenStreams() error {
	if peer.isOutbound {
		peer.openStreamsTimer = time.NewTimer(3 * time.Second)
		go func() {
			<-peer.openStreamsTimer.C
			peer.openStreams()
		}()
	}
	return nil
}

func (peer *Peer) openStreams() {
	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	for _, channel := range Channels {
		stream, err := peer.onStream(channel)
		if err != nil {
			logger.Debugf("Failed to create stream with peer %v %v for channel %v, %v", peer.addrInfo.ID, peer.addrInfo.Addrs, channel, err)
			continue
		}

		if s, ok := peer.streamMap[channel]; ok {
			s.Stop()
		}

		peer.streamMap[channel] = stream
	}
}

func (peer *Peer) AcceptStream(channel cmn.ChannelIDEnum, stream *transport.BufferedStream) {
	if !peer.isOutbound {
		peer.mutex.Lock()
		defer peer.mutex.Unlock()
		if s, ok := peer.streamMap[channel]; ok {
			s.Stop()
		}
		peer.streamMap[channel] = stream
	}
}

func (peer *Peer) StopStream(channel cmn.ChannelIDEnum) {
	peer.mutex.Lock()
	defer peer.mutex.Unlock()
	if stream, ok := peer.streamMap[channel]; ok {
		stream.Stop()
	}
}

// Start is called when the peer starts
func (peer *Peer) Start(ctx context.Context) bool {
	c, cancel := context.WithCancel(ctx)
	peer.ctx = c
	peer.cancel = cancel
	return true
}

// Wait suspends the caller goroutine
func (peer *Peer) Wait() {
	peer.wg.Wait()
}

// Stop is called when the peer stops
func (peer *Peer) Stop() {
	if peer.openStreamsTimer != nil {
		peer.openStreamsTimer.Stop()
	}

	peer.mutex.Lock()
	defer peer.mutex.Unlock()

	for _, stream := range peer.streamMap {
		stream.Stop()
	}
	// peer.streamMap = nil
}

// Send sends the given message through the specified channel to the target peer
func (peer *Peer) Send(channelID cmn.ChannelIDEnum, message interface{}) bool {
	msgBytes, err := peer.onEncode(channelID, message)
	if err != nil {
		logger.Errorf("Failed to encode message to bytes: %v, err: %v", message, err)
		return false
	}

	var n int
	reuseStream := viper.GetBool(cmn.CfgP2PReuseStream)
	if reuseStream {
		peer.mutex.Lock()
		stream := peer.streamMap[channelID]
		peer.mutex.Unlock()

		if stream == nil {
			logger.Debugf("Can't find stream for channel %v", channelID)
			return false
		}

		n, err = stream.Write(msgBytes)
		if err != nil {
			logger.Errorf("Error writing stream to peer %v for channel %v, %v", peer.addrInfo.ID, channelID, err)
			return false
		}
	} else {
		rawStream, err := peer.onRawStream(channelID)
		if rawStream != nil {
			defer rawStream.Close()
		}
		if err != nil {
			logger.Errorf("Stream open failed: %v. peer: %v, addrs: %v, channel: %v", err, peer.ID(), peer.Addrs(), channelID)
			return false
		}

		n, err = rawStream.Write(msgBytes)
		if err != nil {
			logger.Errorf("Error writing stream to peer %v for channel %v, %v", peer.addrInfo.ID, channelID, err)
			return false
		}
	}

	if n != len(msgBytes) {
		logger.Errorf("Didn't write expected bytes length")
		return false
	}

	return true
}

// ID returns the unique idenitifier of the peer in the P2P network
func (peer *Peer) ID() pr.ID {
	return peer.addrInfo.ID
}

// Addrs returns the Multiaddresses of the peer in the P2P network
func (peer *Peer) Addrs() []ma.Multiaddr {
	return peer.addrInfo.Addrs
}

// AddrInfo returns the addrInfo of the peer in the P2P network
func (peer *Peer) AddrInfo() pr.AddrInfo {
	return peer.addrInfo
}

// StreamCreator creates a buffered stream with this peer
type StreamCreator func(channelID cmn.ChannelIDEnum) (*transport.BufferedStream, error)

// RawStreamCreator creates a raw libp2p stream with this peer
type RawStreamCreator func(channelID cmn.ChannelIDEnum) (network.Stream, error)

// MessageParser parses the raw message bytes to type p2ptypes.Message
type MessageParser func(channelID cmn.ChannelIDEnum, rawMessageBytes cmn.Bytes) (p2ptypes.Message, error)

// MessageEncoder encodes type p2ptypes.Message to raw message bytes
type MessageEncoder func(channelID cmn.ChannelIDEnum, message interface{}) (cmn.Bytes, error)

var defaultMessageEncoder MessageEncoder = func(channelID cmn.ChannelIDEnum, message interface{}) (cmn.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

// ReceiveHandler is the callback function to handle received bytes from the given channel
type ReceiveHandler func(message p2ptypes.Message) error

// ErrorHandler is the callback function to handle channel read errors
type ErrorHandler func(interface{})

func (peer *Peer) SetStreamCreator(streamCreator StreamCreator) {
	peer.onStream = streamCreator
}

func (peer *Peer) SetRawStreamCreator(rawStreamCreator RawStreamCreator) {
	peer.onRawStream = rawStreamCreator
}

// SetMessageParser sets the message parser for the connection
func (peer *Peer) SetMessageParser(messageParser MessageParser) {
	peer.onParse = messageParser
}

// SetMessageEncoder sets the message encoder for the connection
func (peer *Peer) SetMessageEncoder(messageEncoder MessageEncoder) {
	peer.onEncode = messageEncoder
}

// SetReceiveHandler sets the receive handler for the connection
func (peer *Peer) SetReceiveHandler(receiveHandler ReceiveHandler) {
	peer.onReceive = receiveHandler
}

// SetErrorHandler sets the error handler for the connection
func (peer *Peer) SetErrorHandler(errorHandler ErrorHandler) {
	peer.onError = errorHandler
}
