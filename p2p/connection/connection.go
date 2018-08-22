package connection

import (
	"bufio"
	"net"
	"runtime/debug"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/timer"
	"github.com/thetatoken/ukulele/p2p/connection/flowrate"
	"github.com/thetatoken/ukulele/serialization/rlp"
)

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

	channelGroup ChannelGroup
	onReceive    ReceiveHandler
	onError      ErrorHandler
	errored      uint32

	sendPulse chan bool
	pongPulse chan bool
	quitPulse chan bool

	flushTimer *timer.ThrottleTimer // flush writes as necessary but throttled
	pingTimer  *timer.RepeatTimer   // send pings periodically

	config ConnectionConfig
}

type ConnectionConfig struct {
	MinWriteBufferSize int
	MinReadBufferSize  int
	SendRate           int64
	RecvRate           int64
	PacketBatchSize    int64
	FlushThrottle      time.Duration
	PingTimeout        time.Duration
}

type ReceiveHandler func(channelID common.ChannelIDEnum, msgBytes common.Bytes)
type ErrorHandler func(interface{})

// CreateConnection creates a Connection instance
func CreateConnection(netconn net.Conn, config ConnectionConfig) *Connection {
	return &Connection{
		netconn:     netconn,
		bufWriter:   bufio.NewWriterSize(netconn, config.MinWriteBufferSize),
		sendMonitor: flowrate.New(0, 0),
		bufReader:   bufio.NewReaderSize(netconn, config.MinReadBufferSize),
		recvMonitor: flowrate.New(0, 0),
		sendPulse:   make(chan bool, 1),
		pongPulse:   make(chan bool, 1),
		quitPulse:   make(chan bool, 1),
		flushTimer:  timer.NewThrottleTimer("flush", config.FlushThrottle),
		pingTimer:   timer.NewRepeatTimer("ping", config.PingTimeout),
		config:      config,
	}
}

func CreateDefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		SendRate:        int64(512000), // 500KB/s
		RecvRate:        int64(512000), // 500KB/s
		PacketBatchSize: int64(10),
		FlushThrottle:   100 * time.Millisecond,
		PingTimeout:     40 * time.Second,
	}
}

// OnStart is called when the connection starts
func (conn *Connection) OnStart() bool {
	go conn.sendRoutine()
	go conn.recvRoutine()
	return true
}

// OnStop is called whten the connection stops
func (conn *Connection) OnStop() {
	if conn.sendPulse != nil {
		close(conn.sendPulse)
	}
	if conn.pongPulse != nil {
		close(conn.pongPulse)
	}
	if conn.quitPulse != nil {
		close(conn.quitPulse)
	}
	conn.netconn.Close()
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
// The message will be send out later
func (conn *Connection) EnqueueMessage(channelID byte, message interface{}) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		log.Errorf("[p2p] Failed to get channel for ID: %v", channelID)
		return false
	}

	msgBytes, err := rlp.EncodeToBytes(message)
	if err != nil {
		log.Errorf("[p2p] Failed to encode message to bytes: %v", message)
		return false
	}
	success := channel.enqueueMessage(msgBytes)
	if success {
		conn.scheduleSendPulse()
	}

	return success
}

// AttemptToEnqueueMessage attempts to enqueue the given message to the
// target channel. The message will be send out later (non-blocking)
func (conn *Connection) AttemptToEnqueueMessage(channelID byte, message interface{}) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		log.Errorf("[p2p] Failed to get channel for ID: %v", channelID)
		return false
	}

	msgBytes, err := rlp.EncodeToBytes(message)
	if err != nil {
		log.Errorf("[p2p] Failed to encode message to bytes: %v", message)
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
func (conn *Connection) CanEnqueueMessage(channelID byte) bool {
	channel := conn.channelGroup.getChannel(channelID)
	if channel == nil {
		return false
	}

	return channel.canEnqueueMessage()
}

// --------------------- Send goroutine --------------------- //

func (conn *Connection) sendRoutine() {
	defer conn.recover()

	for {
		var err error
		select {
		case <-conn.flushTimer.Ch:
			conn.flush()
		case <-conn.pingTimer.Ch:
			err = conn.sendPingSignal()
		case <-conn.pongPulse:
			err = conn.sendPongSignal()
		case <-conn.sendPulse:
			conn.sendPacketBatchAndScheduleSendPulse()
		case <-conn.quitPulse:
			break
		}
		if err != nil {
			log.Errorf("[p2p] sendRoutine error: %v", err)
			conn.stopForError(err)
			break
		}
	}
}

func (conn *Connection) sendPingSignal() error {
	err := rlp.Encode(conn.bufWriter, packetTypePing)
	conn.sendMonitor.Update(int(1))
	conn.flush()
	return err
}

func (conn *Connection) sendPongSignal() error {
	err := rlp.Encode(conn.bufWriter, packetTypePong)
	conn.sendMonitor.Update(int(1))
	conn.flush()
	return err
}

func (conn *Connection) sendPacketBatchAndScheduleSendPulse() {
	success, dataExhausted := conn.sendPacketBatch()
	if !success || !dataExhausted {
		conn.scheduleSendPulse()
	}
}

// --------------------- Recv goroutine --------------------- //

func (conn *Connection) recvRoutine() {
	defer conn.recover()

	for {
		// Block until recvMonitor allows reading
		conn.recvMonitor.Limit(maxPacketTotalSize, atomic.LoadInt64(&conn.config.RecvRate), true)

		// Read packet type
		var packetType byte
		err := rlp.Decode(conn.bufReader, packetType)
		conn.recvMonitor.Update(int(1))
		if err != nil {
			log.Errorf("[p2p] recvRoutine: failed to decode packetType")
			break
		}

		// Read more data based on the packet type
		switch packetType {
		case packetTypePing:
			conn.schedulePongPulse()
		case packetTypePong:
			// Do nothing for now
		case packetTypeMsg:
			conn.handleReceivedPacket()
		default:
			log.Errorf("[p2p] recvRoutine: unknown packetType %v", packetType)
		}

		conn.pingTimer.Reset()
	}

	close(conn.pongPulse)
}

func (conn *Connection) handleReceivedPacket() (success bool) {
	//pkt, n, err := Packet{}, int(0), error(nil)
	return false
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
			log.Errorf("[p2p] sendPacketBatch: failed to send out packet")
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

	nonemptyPacket, numBytes, err := channel.sendPacketTo(conn.bufWriter)
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

func (conn *Connection) stopForError(r interface{}) {
	if atomic.CompareAndSwapUint32(&conn.errored, 0, 1) {
		if conn.onError != nil {
			conn.onError(r)
		}
	}
}

func (conn *Connection) recover() {
	if r := recover(); r != nil {
		stack := debug.Stack()
		err := struct {
			Srr   interface{}
			Stack []byte
		}{
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
