package connection

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/timer"
	"github.com/thetatoken/theta/p2p/connection/flowrate"
	"github.com/thetatoken/theta/p2p/types"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/rlp"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "p2p"})

//
// Connection models the connection between the current node and a peer node.
// A connection has a ChannelGroup which can contain multiple Channels
//
type Connection struct {
	netconn net.Conn

	bufWriter   *bufio.Writer
	sendMonitor *flowrate.Monitor

	bufReader   *bufio.Reader
	recvMonitor *flowrate.Monitor

	bufConn io.ReadWriter

	channelGroup ChannelGroup
	onParse      MessageParser
	onEncode     MessageEncoder
	onReceive    ReceiveHandler
	onError      ErrorHandler
	errored      uint32

	sendPulse chan bool
	pongPulse chan bool
	quitPulse chan bool

	flushTimer *timer.ThrottleTimer // flush writes as necessary but throttled
	pingTimer  *timer.RepeatTimer   // send pings periodically

	pendingPings uint32

	config ConnectionConfig

	rmu, wmu sync.Mutex
	rw       *rlpxFrameRW

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

//
// ConnectionConfig specifies the configurations of the Connection
//
type ConnectionConfig struct {
	SendRate        int64
	RecvRate        int64
	PacketBatchSize int64
	FlushThrottle   time.Duration
	PingTimeout     time.Duration
	MaxPendingPings uint32
}

// MessageParser parses the raw message bytes to type p2ptypes.Message
type MessageParser func(channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error)

// MessageEncoder encodes type p2ptypes.Message to raw message bytes
type MessageEncoder func(channelID common.ChannelIDEnum, message interface{}) (common.Bytes, error)

var defaultMessageEncoder MessageEncoder = func(channelID common.ChannelIDEnum, message interface{}) (common.Bytes, error) {
	return rlp.EncodeToBytes(message)
}

// ReceiveHandler is the callback function to handle received bytes from the given channel
type ReceiveHandler func(message p2ptypes.Message) error

// ErrorHandler is the callback function to handle channel read errors
type ErrorHandler func(interface{})

// CreateConnection creates a Connection instance
func CreateConnection(netconn net.Conn, config ConnectionConfig) *Connection {
	if netconn != nil {
		logger.Debugf("Create connection, local: %v, remote: %v", netconn.LocalAddr(), netconn.RemoteAddr())
	}

	channelCheckpoint := createDefaultChannel(common.ChannelIDCheckpoint)
	channelHeader := createDefaultChannel(common.ChannelIDHeader)
	channelBlock := createDefaultChannel(common.ChannelIDBlock)
	channelProposal := createDefaultChannel(common.ChannelIDProposal)
	channelVote := createDefaultChannel(common.ChannelIDVote)
	channelTransaction := createDefaultChannel(common.ChannelIDTransaction)
	channelPeerDiscover := createDefaultChannel(common.ChannelIDPeerDiscovery)
	channelPing := createDefaultChannel(common.ChannelIDPing)
	channelGuardian := createDefaultChannel(common.ChannelIDGuardian)
	channelNATMapping := createDefaultChannel(common.ChannelIDNATMapping)
	channelEliteEdgeNodeVote := createDefaultChannel(common.ChannelIDEliteEdgeNodeVote)
	channelEliteAggregatedEdgeNodeVotes := createDefaultChannel(common.ChannelIDAggregatedEliteEdgeNodeVotes)
	channels := []*Channel{
		&channelCheckpoint,
		&channelHeader,
		&channelBlock,
		&channelProposal,
		&channelVote,
		&channelTransaction,
		&channelPeerDiscover,
		&channelPing,
		&channelGuardian,
		&channelNATMapping,
		&channelEliteEdgeNodeVote,
		&channelEliteAggregatedEdgeNodeVotes,
	}

	success, channelGroup := createChannelGroup(getDefaultChannelGroupConfig(), channels)
	if !success {
		return nil
	}

	conn := &Connection{
		netconn:      netconn,
		bufWriter:    bufio.NewWriter(netconn),
		sendMonitor:  flowrate.New(0, 0),
		bufReader:    bufio.NewReader(netconn),
		recvMonitor:  flowrate.New(0, 0),
		channelGroup: channelGroup,
		sendPulse:    make(chan bool, 1),
		pongPulse:    make(chan bool, 1),
		quitPulse:    make(chan bool, 1),
		flushTimer:   timer.NewThrottleTimer("flush", config.FlushThrottle),
		pingTimer:    timer.NewRepeatTimer("ping", config.PingTimeout),
		config:       config,
		wg:           &sync.WaitGroup{},

		onEncode: defaultMessageEncoder,
	}
	conn.bufConn = struct {
		io.Reader
		io.Writer
	}{Reader: conn.bufReader, Writer: conn.netconn}
	return conn
}

// GetDefaultConnectionConfig returns the default ConnectionConfig
func GetDefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		SendRate:        int64(512000), // 500KB/s
		RecvRate:        int64(512000), // 500KB/s
		PacketBatchSize: int64(10),
		FlushThrottle:   100 * time.Millisecond,
		PingTimeout:     40 * time.Second,
		MaxPendingPings: 3,
	}
}

// SetPingTimer for testing purpose
func (conn *Connection) SetPingTimer(seconds time.Duration) {
	conn.pingTimer = timer.NewRepeatTimer("ping", seconds*time.Second)
	conn.pingTimer.Reset()
}

// Start is called when the connection starts
func (conn *Connection) Start(ctx context.Context) bool {
	c, cancel := context.WithCancel(ctx)
	conn.ctx = c
	conn.cancel = cancel

	// NOTE: the life cycle of recvRoutine() is not managed by
	//       the wg since rlp.Decode() is a blocking call
	conn.wg.Add(1)

	go conn.sendRoutine()
	go conn.recvRoutine()
	return true
}

// Wait suspends the caller goroutine
func (conn *Connection) Wait() {
	conn.wg.Wait()
	conn.netconn.Close()
}

// CancelConnection for testing purpose only
func (conn *Connection) CancelConnection() {
	conn.cancel()
}

// Stop is called when the connection stops
func (conn *Connection) Stop() {
	if conn.GetNetconn() == nil {
		return
	}

	logger.Warnf("Stopping connection, local: %v, remote: %v", conn.GetNetconn().LocalAddr(), conn.GetNetconn().RemoteAddr())
	err := conn.netconn.Close()
	if err != nil {
		logger.Warnf("Failed to close connection: %v", err)
	}

	if conn.cancel != nil {
		conn.cancel()
	}
}

// SetMessageParser sets the message parser for the connection
func (conn *Connection) SetMessageParser(messageParser MessageParser) {
	conn.onParse = messageParser
}

// SetMessageEncoder sets the message encoder for the connection
func (conn *Connection) SetMessageEncoder(messageEncoder MessageEncoder) {
	conn.onEncode = messageEncoder
}

// SetReceiveHandler sets the receive handler for the connection
func (conn *Connection) SetReceiveHandler(receiveHandler ReceiveHandler) {
	conn.onReceive = receiveHandler
}

// SetErrorHandler sets the error handler for the connection
func (conn *Connection) SetErrorHandler(errorHandler ErrorHandler) {
	conn.onError = errorHandler
}

// EnqueueMessage enqueues the given message to the target channel.
// The message will be sent out later
func (conn *Connection) EnqueueMessage(channelID common.ChannelIDEnum, message interface{}) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		logger.Errorf("Failed to get channel for ID: %v", channelID)
		return false
	}

	msgBytes, err := conn.onEncode(channelID, message)
	if err != nil {
		logger.Errorf("Failed to encode message to bytes: %v, err: %v", message, err)
		return false
	}
	success := channel.enqueueMessage(msgBytes)
	if success {
		conn.scheduleSendPulse()
	}

	return success
}

// AttemptToEnqueueMessage attempts to enqueue the given message to the
// target channel. The message will be sent out later (non-blocking)
func (conn *Connection) AttemptToEnqueueMessage(channelID common.ChannelIDEnum, message interface{}) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		logger.Errorf("Failed to get channel for ID: %v", channelID)
		return false
	}

	msgBytes, err := conn.onEncode(channelID, message)
	if err != nil {
		logger.Errorf("Failed to encode message to bytes: %v, error: %v", message, err)
		return false
	}
	success := channel.attemptToEnqueueMessage(msgBytes)
	if success {
		conn.scheduleSendPulse()
	}

	return success
}

// CanEnqueueMessage returns whether more messages can still be enqueued
// into the connection at the moment
func (conn *Connection) CanEnqueueMessage(channelID common.ChannelIDEnum) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		return false
	}

	return channel.canEnqueueMessage()
}

// --------------------- Send goroutine --------------------- //

func (conn *Connection) sendRoutine() {
	defer conn.wg.Done()
	defer conn.recover()
	defer conn.flushTimer.Stop()
	defer conn.pingTimer.Stop()
	defer func() {
		_ = conn.netconn.Close()
	}()

	for {
		var err error
		select {
		case <-conn.ctx.Done():
			conn.wmu.Lock()
			conn.stopped = true
			conn.wmu.Unlock()
			return
		case <-conn.flushTimer.Ch:
			conn.flush()
		case <-conn.pingTimer.Ch:
			err = conn.sendPingSignal()
		case <-conn.pongPulse:
			err = conn.sendPongSignal()
		case <-conn.sendPulse:
			conn.sendPacketBatchAndScheduleSendPulse()
		case <-conn.quitPulse:
			return
		}
		if err != nil {
			logger.Warnf("sendRoutine error: %v", err)
			conn.stopForError(err)
			return
		}
	}
}

func (conn *Connection) sendPingSignal() error {
	if atomic.LoadUint32(&conn.pendingPings) >= conn.config.MaxPendingPings {
		//conn.onError(nil)
		conn.stopForError(nil)
		logger.Warnf("Peer not responding to ping %v", conn.netconn.RemoteAddr())
		return fmt.Errorf("Peer not responding to ping %v", conn.netconn.RemoteAddr())
	}
	pingPacket := &Packet{
		ChannelID: common.ChannelIDPing,
		Bytes:     []byte{p2ptypes.PingSignal},
		IsEOF:     byte(0x01),
	}
	err := conn.writePacket(pingPacket)
	if err != nil {
		return err
	}
	conn.sendMonitor.Update(int(1))
	conn.flush()
	atomic.AddUint32(&conn.pendingPings, 1)
	return nil
}

func (conn *Connection) sendPongSignal() error {
	pongPacket := &Packet{
		ChannelID: common.ChannelIDPing,
		Bytes:     []byte{p2ptypes.PongSignal},
		IsEOF:     byte(0x01),
	}
	err := conn.writePacket(pongPacket)
	if err != nil {
		return err
	}
	conn.sendMonitor.Update(int(1))
	conn.flush()
	return nil
}

func (conn *Connection) sendPacketBatchAndScheduleSendPulse() {
	success, dataExhausted := conn.sendPacketBatch()
	if !success || !dataExhausted {
		conn.scheduleSendPulse()
	}
}

// --------------------- Recv goroutine --------------------- //

func (conn *Connection) readPacket() (*Packet, error) {
	// Plaintext transport.
	if conn.rw == nil {
		packet := &Packet{}
		s := rlp.NewStream(conn.bufReader, maxPayloadSize*1024)
		err := s.Decode(packet)
		return packet, err
	}
	// Encrypted transport.
	conn.rmu.Lock()
	defer conn.rmu.Unlock()
	return conn.rw.ReadPacket()
}

func (conn *Connection) writePacket(packet *Packet) error {
	// Plaintext transport.
	if conn.rw == nil {
		return rlp.Encode(conn.bufWriter, packet)
	}
	// Encrypted transport.
	conn.wmu.Lock()
	defer conn.wmu.Unlock()
	return conn.rw.WritePacket(packet)
}

func (conn *Connection) recvRoutine() {
	//defer conn.wg.Done() // NOTE: rlp.Decode() is a blocking call
	defer conn.recover()

	for {
		select {
		case <-conn.ctx.Done():
			conn.wmu.Lock()
			conn.stopped = true
			conn.wmu.Unlock()
			return
		default:
		}

		// Block until recvMonitor allows reading
		conn.recvMonitor.Limit(maxPacketTotalSize, atomic.LoadInt64(&conn.config.RecvRate), true)

		packet, err := conn.readPacket()
		if err != nil {
			logger.Warnf("recvRoutine: failed to decode packet: %v, error: %v", packet, err)
			return
		}
		conn.recvMonitor.Update(int(1))
		switch packet.ChannelID {
		case common.ChannelIDPing:
			conn.handlePingPong(packet)
		default:
			conn.handleReceivedPacket(packet)
		}

		//conn.pingTimer.Reset() // TODO: replace with lightweight Reset()
		atomic.StoreUint32(&conn.pendingPings, 0)
	}
}

func (conn *Connection) handlePingPong(packet *Packet) (success bool) {
	if packet.ChannelID != common.ChannelIDPing {
		logger.Errorf("Invalid channel for Ping/Pong signal")
		return false
	}
	if len(packet.Bytes) != 1 {
		logger.Errorf("Invalid Ping/Pong packet")
		return false
	}

	pingpong := packet.Bytes[0]
	switch pingpong {
	case p2ptypes.PingSignal:
		conn.schedulePongPulse()
	case p2ptypes.PongSignal:
		// do nothing for now
	default:
		logger.Errorf("Invalid Ping/Pong signal")
		return false
	}

	return true
}

func (conn *Connection) handleReceivedPacket(packet *Packet) (success bool) {
	channelID := packet.ChannelID
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		return false
	}

	aggregatedBytes, success := channel.receivePacket(packet)
	if !success {
		return false
	}

	if aggregatedBytes == nil {
		return true
	}

	message, err := conn.onParse(packet.ChannelID, aggregatedBytes)
	if err != nil {
		logger.Errorf("Error parsing packet: %v, err: %v", packet, err)
		return false
	}

	err = conn.onReceive(message)
	if err != nil {
		logger.Debugf("Error handling message: %v, err: %v", message, err)
		return false
	}

	return true
}

// --------------------- IO Handling --------------------- //

func (conn *Connection) flush() error {
	err := conn.bufWriter.Flush()
	return err
}

func (conn *Connection) sendPacketBatch() (success bool, exhausted bool) {
	// Block until sendMonitor allows sending
	conn.sendMonitor.Limit(maxPacketTotalSize, atomic.LoadInt64(&conn.config.SendRate), true)

	// Now send out the packet batch
	packetBatchSize := conn.config.PacketBatchSize
	for i := int64(0); i < packetBatchSize; i++ {
		success, exhausted := conn.sendPacket()
		if !success {
			logger.Errorf("sendPacketBatch: failed to send out packet")
			return false, exhausted
		}
		if exhausted {
			return true, true
		}
	}

	return true, false
}

// Boolean exhausted indicates whether the data in the selected channel has exhausted
func (conn *Connection) sendPacket() (success bool, exhausted bool) {
	success, channel := conn.channelGroup.nextChannelToSendPacket()
	if !success {
		return false, false // TODO: error handling
	}
	if channel == nil {
		return true, true // Nothing to be sent
	}

	nonemptyPacket, numBytes, err := channel.sendPacketTo(conn)
	if err != nil {
		return false, !nonemptyPacket
	}
	if !nonemptyPacket {
		return true, true // Nothing to be sent
	}

	conn.sendMonitor.Update(numBytes)
	conn.flushTimer.Set()

	return true, false
}

// --------------------- Utils --------------------- //

// GetNetconn returns the attached network connection
func (conn *Connection) GetNetconn() net.Conn {
	return conn.netconn
}

// GetBufNetconn returns buffered network connection
func (conn *Connection) GetBufNetconn() io.ReadWriter {
	return conn.bufConn
}

// GetBufReader returns buffered reader for network connection
func (conn *Connection) GetBufReader() *bufio.Reader {
	return conn.bufReader
}

func (conn *Connection) stopForError(r interface{}) {
	logger.Warnf("Connection error: %v", r)
	if atomic.CompareAndSwapUint32(&conn.errored, 0, 1) {
		conn.Stop()
		if conn.onError != nil {
			conn.onError(r)
		}
	}
}

func (conn *Connection) recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := types.StackError{
			r, stack,
		}
		conn.stopForError(err)
	}
}

func (conn *Connection) scheduleSendPulse() {
	select {
	case conn.sendPulse <- true:
	default:
	}
}

func (conn *Connection) schedulePongPulse() {
	select {
	case conn.pongPulse <- true:
	default:
	}
}

func (conn *Connection) scheduleQuitPulse() {
	select {
	case conn.quitPulse <- true:
	default:
	}
}
