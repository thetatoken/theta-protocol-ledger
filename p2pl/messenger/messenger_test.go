package messenger

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p-core/network"
	pr "github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	p2ptypes "github.com/thetatoken/theta/p2p/types"
	"github.com/thetatoken/theta/p2pl/transport"
	"github.com/thetatoken/theta/rlp"
)

type MockMsgHandler struct {
	C         chan interface{}
	channelID common.ChannelIDEnum
}

type rawEncodingHandler struct {
	*MockMsgHandler
}

func (handler *rawEncodingHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	return message.([]byte), nil
}

func (m *MockMsgHandler) GetChannelIDs() []common.ChannelIDEnum {
	return []common.ChannelIDEnum{m.channelID}
}

func decodeMessage(raw common.Bytes) (interface{}, error) {
	data := []byte{}
	if err := rlp.DecodeBytes(raw, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (m *MockMsgHandler) ParseMessage(peerID string, channelID common.ChannelIDEnum, rawMessageBytes common.Bytes) (p2ptypes.Message, error) {
	message := p2ptypes.Message{PeerID: peerID, ChannelID: channelID}
	data, err := decodeMessage(rawMessageBytes)
	message.Content = data
	return message, err
}

func (m *MockMsgHandler) EncodeMessage(message interface{}) (common.Bytes, error) {
	var buf bytes.Buffer
	if err := rlp.Encode(&buf, message); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (m *MockMsgHandler) HandleMessage(message p2ptypes.Message) error {
	m.C <- message.Content
	return nil
}

func generatePrivateKey(t *testing.T) *crypto.PrivateKey {
	t.Helper()
	privateKey, _, err := crypto.GenerateKeyPair()
	require.NoError(t, err)
	return privateKey
}

func newMessenger(t *testing.T, privKey *crypto.PrivateKey, seedPeerNetAddresses []string, port int) *Messenger {
	t.Helper()
	config := GetDefaultMessengerConfig()
	config.disablePeerPersistence = true
	messenger, err := CreateMessenger(privKey.PublicKey(), seedPeerNetAddresses, port, true,
		config, false, context.Background())
	require.NoError(t, err)
	require.NotNil(t, messenger)
	t.Cleanup(func() {
		messenger.Stop()
		require.NoError(t, messenger.host.Close())
	})
	return messenger
}

func addSeedPeer(node, seed *Messenger) {
	seedID := seed.host.ID()
	node.seedPeers[seedID] = &pr.AddrInfo{ID: seedID, Addrs: seed.host.Addrs()}
}

func seedAddress(node *Messenger, port int) string {
	return fmt.Sprintf("/ip4/127.0.0.1/tcp/%v/ipfs/%v", port, node.host.ID())
}

func registerBlockHandler(node *Messenger, capacity int) *MockMsgHandler {
	return registerHandler(node, common.ChannelIDBlock, capacity)
}

func registerHandler(node *Messenger, channelID common.ChannelIDEnum, capacity int) *MockMsgHandler {
	handler := &MockMsgHandler{C: make(chan interface{}, capacity), channelID: channelID}
	node.RegisterMessageHandler(handler)
	return handler
}

func startMessengers(t *testing.T, nodes ...*Messenger) {
	t.Helper()
	for _, node := range nodes {
		require.NoError(t, node.Start(context.Background()))
	}
}

func receiveContent(t *testing.T, messages <-chan interface{}, timeout time.Duration) []byte {
	t.Helper()
	select {
	case data := <-messages:
		content, ok := data.([]byte)
		require.True(t, ok)
		return content
	case <-time.After(timeout):
		t.Fatal("timed out waiting for peer message")
		return nil
	}
}

func requireNoContent(t *testing.T, messages <-chan interface{}, timeout time.Duration) {
	t.Helper()
	select {
	case data := <-messages:
		t.Fatalf("received unexpected peer message of type %T", data)
	case <-time.After(timeout):
	}
}

func waitForCondition(t *testing.T, timeout time.Duration, condition func() bool, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal(message)
}

func contentWithEncodedSize(t *testing.T, handler *MockMsgHandler, target int) []byte {
	t.Helper()
	content := make([]byte, target)
	for i := 0; i < 8; i++ {
		encoded, err := handler.EncodeMessage(content)
		require.NoError(t, err)
		delta := target - len(encoded)
		if delta == 0 {
			return content
		}
		require.GreaterOrEqual(t, len(content)+delta, 0)
		content = make([]byte, len(content)+delta)
	}
	t.Fatalf("could not construct content with encoded size %v", target)
	return nil
}

func sendAndReceive(t *testing.T, sender *Messenger, receiverID string, receiver <-chan interface{}, message p2ptypes.Message) []byte {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if sender.Send(receiverID, message) {
			return receiveContent(t, receiver, 8*time.Second)
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("timed out waiting for the peer stream to become writable")
	return nil
}

func blockMessage(content string) p2ptypes.Message {
	return p2ptypes.Message{ChannelID: common.ChannelIDBlock, Content: []byte(content)}
}

func TestReadAllWithLimit(t *testing.T) {
	msg, err := readAllWithLimit(bytes.NewReader(bytes.Repeat([]byte{0x42}, 10)), 10)
	require.NoError(t, err)
	require.Len(t, msg, 10)

	msg, err = readAllWithLimit(bytes.NewReader(bytes.Repeat([]byte{0x42}, 11)), 10)
	require.Error(t, err)
	require.Nil(t, msg)
}

func TestReadAllWithProgressDeadlineAllowsActiveSlowTransfer(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()

	expected := []byte("slow-but-active")
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		defer right.Close()
		for _, b := range expected {
			if _, err := right.Write([]byte{b}); err != nil {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}()

	actual, err := readAllWithProgressDeadline(left, 64*1024, 100*time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
	<-writerDone
}

func TestReadAllWithProgressDeadlineCapsContinuousTrickle(t *testing.T) {
	left, right := net.Pipe()
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		defer right.Close()
		for i := 0; i < 100; i++ {
			if _, err := right.Write([]byte{byte(i)}); err != nil {
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
	}()

	_, err := readAllWithProgressDeadline(left, 64, 20*time.Millisecond)
	require.Error(t, err)
	require.NoError(t, left.Close())
	<-writerDone
}

func TestMaintainConnectivityRoutineStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	messenger := &Messenger{seedPeerOnly: true}

	go func() {
		messenger.maintainConnectivityRoutine(ctx)
		close(done)
	}()
	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connectivity routine did not stop after context cancellation")
	}
}

func TestInboundStreamBudget(t *testing.T) {
	pid := pr.ID("test-peer")
	msgr := &Messenger{
		inboundStreamUsage: make(map[pr.ID]*inboundStreamUsage),
		penalizedPeers:     make(map[pr.ID]time.Time),
	}

	require.True(t, msgr.acquireInboundStream(pid, common.ChannelIDBlock, true))
	require.False(t, msgr.acquireInboundStream(pid, common.ChannelIDBlock, true),
		"reusable streams must be unique per peer and channel")
	msgr.releaseInboundStream(pid, common.ChannelIDBlock)

	for i := 0; i < maxInboundReservedBytesPerPeer/common.MaxBlockMessageSize; i++ {
		require.True(t, msgr.acquireInboundStream(pid, common.ChannelIDBlock, false))
	}
	require.False(t, msgr.acquireInboundStream(pid, common.ChannelIDBlock, false),
		"per-peer reserved-byte budget must cap concurrent block streams")
	for i := 0; i < maxInboundReservedBytesPerPeer/common.MaxBlockMessageSize; i++ {
		msgr.releaseInboundStream(pid, common.ChannelIDBlock)
	}
	require.NotContains(t, msgr.inboundStreamUsage, pid)
}

func TestPeerPenaltyExpires(t *testing.T) {
	pid := pr.ID("test-peer")
	now := time.Now()
	msgr := &Messenger{
		inboundStreamUsage: make(map[pr.ID]*inboundStreamUsage),
		penalizedPeers: map[pr.ID]time.Time{
			pid: now.Add(time.Second),
		},
	}

	require.True(t, msgr.isPeerPenalized(pid, now))
	require.False(t, msgr.isPeerPenalized(pid, now.Add(time.Second)))
	require.NotContains(t, msgr.penalizedPeers, pid)
}

func TestPeerPenaltyTableIsBounded(t *testing.T) {
	msgr := &Messenger{penalizedPeers: make(map[pr.ID]time.Time)}
	now := time.Now()

	for i := 0; i < maxPenalizedPeers+100; i++ {
		msgr.recordMalformedPeerPenalty(pr.ID(fmt.Sprintf("peer-%d", i)), now)
	}

	require.Len(t, msgr.penalizedPeers, maxPenalizedPeers)
}

func TestSendMessage(t *testing.T) {
	const port1, port2 = 11001, 12001

	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 4)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 4)
	addSeedPeer(node1, node2)
	startMessengers(t, node1, node2)

	warmup := blockMessage("stream-ready")
	require.Equal(t, warmup.Content, sendAndReceive(t, node1, node2.ID(), handler2.C, warmup))
	require.Equal(t, warmup.Content, sendAndReceive(t, node2, node1.ID(), handler1.C, warmup))

	atLimitContent := contentWithEncodedSize(t, handler1, common.MaxBlockMessageSize)
	atLimit := p2ptypes.Message{ChannelID: common.ChannelIDBlock, Content: atLimitContent}
	require.Equal(t, atLimitContent, sendAndReceive(t, node1, node2.ID(), handler2.C, atLimit))

	overLimitContent := contentWithEncodedSize(t, handler1, common.MaxBlockMessageSize+1)
	overLimit := p2ptypes.Message{ChannelID: common.ChannelIDBlock, Content: overLimitContent}
	require.False(t, node1.Send(node2.ID(), overLimit))
	require.Error(t, node1.Publish(overLimit))
	requireNoContent(t, handler2.C, 250*time.Millisecond)

	afterRejection := blockMessage("still-connected")
	require.Equal(t, afterRejection.Content,
		sendAndReceive(t, node1, node2.ID(), handler2.C, afterRejection))
}

func TestSendMessageWithoutStreamReuse(t *testing.T) {
	original := viper.GetBool(common.CfgP2PReuseStream)
	viper.Set(common.CfgP2PReuseStream, false)
	t.Cleanup(func() { viper.Set(common.CfgP2PReuseStream, original) })

	const port1, port2 = 11001, 12001
	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 2)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 2)
	addSeedPeer(node1, node2)
	startMessengers(t, node1, node2)

	message := blockMessage("one-message-stream")
	require.Equal(t, message.Content, sendAndReceive(t, node1, node2.ID(), handler2.C, message))
	require.Equal(t, message.Content, sendAndReceive(t, node2, node1.ID(), handler1.C, message))
}

func TestOversizedNonReusableStreamIsolatedToSendingPeer(t *testing.T) {
	original := viper.GetBool(common.CfgP2PReuseStream)
	viper.Set(common.CfgP2PReuseStream, false)
	t.Cleanup(func() { viper.Set(common.CfgP2PReuseStream, original) })

	const victimPort, attackerPort, honestPort = 11001, 12001, 13001
	victim := newMessenger(t, generatePrivateKey(t), nil, victimPort)
	victimHandler := registerHandler(victim, common.ChannelIDTransaction, 2)
	attacker := newMessenger(t, generatePrivateKey(t), nil, attackerPort)
	honest := newMessenger(t, generatePrivateKey(t), []string{seedAddress(victim, victimPort)}, honestPort)
	honestHandler := registerHandler(honest, common.ChannelIDTransaction, 2)
	addSeedPeer(victim, attacker)
	addSeedPeer(victim, honest)
	startMessengers(t, victim, honest)

	attacker.host.Peerstore().AddAddrs(victim.host.ID(), victim.host.Addrs(), time.Minute)
	require.NoError(t, attacker.host.Connect(context.Background(), pr.AddrInfo{
		ID: victim.host.ID(), Addrs: victim.host.Addrs(),
	}))

	streamID := protocol.ID(victim.protocolPrefix + strconv.Itoa(int(common.ChannelIDTransaction)))
	rawStream, err := attacker.host.NewStream(context.Background(), victim.host.ID(), streamID)
	require.NoError(t, err)
	require.NoError(t, rawStream.SetWriteDeadline(time.Now().Add(5*time.Second)))
	written, writeErr := rawStream.Write(bytes.Repeat([]byte{0x42}, common.MaxNormalMessageSize+1))
	_ = rawStream.Close()
	require.Greater(t, written, common.MaxNormalMessageSize,
		"the receiver must observe the byte that crosses the configured limit")
	require.NoError(t, writeErr)

	waitForCondition(t, 3*time.Second, func() bool {
		return victim.isPeerPenalized(attacker.host.ID(), time.Now())
	}, "victim did not penalize the oversized-stream sender")
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.host.Network().Connectedness(attacker.host.ID()) != network.Connected
	}, "victim did not disconnect the oversized-stream sender")
	requireNoContent(t, victimHandler.C, 250*time.Millisecond)

	honestMessage := p2ptypes.Message{
		ChannelID: common.ChannelIDTransaction,
		Content:   []byte("honest-peer-still-connected"),
	}
	require.Equal(t, honestMessage.Content,
		sendAndReceive(t, honest, victim.ID(), victimHandler.C, honestMessage))
	requireNoContent(t, honestHandler.C, 250*time.Millisecond)
}

func TestMalformedNonReusablePayloadIsolatedToSendingPeer(t *testing.T) {
	original := viper.GetBool(common.CfgP2PReuseStream)
	viper.Set(common.CfgP2PReuseStream, false)
	t.Cleanup(func() { viper.Set(common.CfgP2PReuseStream, original) })

	const victimPort, attackerPort, honestPort = 11001, 12001, 13001
	victim := newMessenger(t, generatePrivateKey(t), nil, victimPort)
	victimHandler := registerBlockHandler(victim, 2)
	attacker := newMessenger(t, generatePrivateKey(t), []string{seedAddress(victim, victimPort)}, attackerPort)
	attackerHandler := registerBlockHandler(attacker, 2)
	honest := newMessenger(t, generatePrivateKey(t), []string{seedAddress(victim, victimPort)}, honestPort)
	honestHandler := registerBlockHandler(honest, 2)
	addSeedPeer(victim, attacker)
	addSeedPeer(victim, honest)
	startMessengers(t, victim, attacker, honest)

	warmup := blockMessage("streams-ready")
	require.Equal(t, warmup.Content, sendAndReceive(t, attacker, victim.ID(), victimHandler.C, warmup))
	require.Equal(t, warmup.Content, sendAndReceive(t, honest, victim.ID(), victimHandler.C, warmup))

	attacker.msgHandlerMap[common.ChannelIDBlock] = &rawEncodingHandler{attackerHandler}
	malformed := p2ptypes.Message{ChannelID: common.ChannelIDBlock, Content: []byte{0xff}}
	require.True(t, attacker.Send(victim.ID(), malformed))

	waitForCondition(t, 3*time.Second, func() bool {
		return victim.isPeerPenalized(attacker.host.ID(), time.Now())
	}, "victim did not penalize the malformed one-shot payload sender")
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.host.Network().Connectedness(attacker.host.ID()) != network.Connected
	}, "victim did not disconnect the malformed one-shot payload sender")
	requireNoContent(t, victimHandler.C, 250*time.Millisecond)

	honestMessage := blockMessage("honest-after-malformed-one-shot-payload")
	require.Equal(t, honestMessage.Content,
		sendAndReceive(t, honest, victim.ID(), victimHandler.C, honestMessage))
	requireNoContent(t, honestHandler.C, 250*time.Millisecond)
}

func TestBroadcastMessage(t *testing.T) {
	const port1, port2, port3 = 11001, 12001, 13001

	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 4)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 4)
	node3 := newMessenger(t, generatePrivateKey(t), []string{
		seedAddress(node1, port1), seedAddress(node2, port2),
	}, port3)
	handler3 := registerBlockHandler(node3, 4)

	addSeedPeer(node1, node2)
	addSeedPeer(node1, node3)
	addSeedPeer(node2, node3)
	startMessengers(t, node1, node2, node3)

	warmup := blockMessage("mesh-ready")
	require.Equal(t, warmup.Content, sendAndReceive(t, node1, node2.ID(), handler2.C, warmup))
	require.Equal(t, warmup.Content, sendAndReceive(t, node1, node3.ID(), handler3.C, warmup))
	time.Sleep(1500 * time.Millisecond)

	message := blockMessage("broadcast")
	node1.Broadcast(message, false)
	require.Equal(t, message.Content, receiveContent(t, handler2.C, 8*time.Second))
	require.Equal(t, message.Content, receiveContent(t, handler3.C, 8*time.Second))
	requireNoContent(t, handler1.C, 250*time.Millisecond)
}

func TestFullyConnected(t *testing.T) {
	const port1, port2, port3 = 11001, 12001, 13001

	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 4)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 4)
	node3 := newMessenger(t, generatePrivateKey(t), []string{
		seedAddress(node1, port1), seedAddress(node2, port2),
	}, port3)
	handler3 := registerBlockHandler(node3, 4)

	addSeedPeer(node1, node2)
	addSeedPeer(node1, node3)
	addSeedPeer(node2, node3)
	startMessengers(t, node1, node2, node3)

	message12 := blockMessage("one-to-two")
	require.Equal(t, message12.Content, sendAndReceive(t, node1, node2.ID(), handler2.C, message12))
	message23 := blockMessage("two-to-three")
	require.Equal(t, message23.Content, sendAndReceive(t, node2, node3.ID(), handler3.C, message23))
	message31 := blockMessage("three-to-one")
	require.Equal(t, message31.Content, sendAndReceive(t, node3, node1.ID(), handler1.C, message31))
}

func TestMalformedFrameIsolatedToSendingPeer(t *testing.T) {
	const port1, port2, port3 = 11001, 12001, 13001

	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 4)
	node2 := newMessenger(t, generatePrivateKey(t), nil, port2)
	handler2 := registerBlockHandler(node2, 4)
	honest := newMessenger(t, generatePrivateKey(t), nil, port3)
	honestHandler := registerBlockHandler(honest, 4)

	victim, victimHandler := node1, handler1
	attacker, attackerHandler := node2, handler2
	if victim.ID() > attacker.ID() {
		victim, attacker = attacker, victim
		victimHandler, attackerHandler = attackerHandler, victimHandler
	}
	addSeedPeer(victim, attacker)
	addSeedPeer(victim, honest)
	addSeedPeer(honest, victim)
	startMessengers(t, victim, honest)

	attacker.host.Peerstore().AddAddrs(victim.host.ID(), victim.host.Addrs(), time.Minute)
	require.NoError(t, attacker.host.Connect(context.Background(), pr.AddrInfo{
		ID: victim.host.ID(), Addrs: victim.host.Addrs(),
	}))

	streamID := protocol.ID(victim.protocolPrefix + strconv.Itoa(int(common.ChannelIDBlock)))
	rawStream, err := attacker.host.NewStream(context.Background(), victim.host.ID(), streamID)
	require.NoError(t, err)
	header := make([]byte, 8)
	binary.BigEndian.PutUint32(header[0:4], 0)
	binary.BigEndian.PutUint32(header[4:8], 1<<30)
	written, err := rawStream.Write(header)
	require.NoError(t, err)
	require.Equal(t, len(header), written)
	_ = rawStream.Close()
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.isPeerPenalized(attacker.host.ID(), time.Now())
	}, "victim did not penalize the malformed-frame sender")
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.host.Network().Connectedness(attacker.host.ID()) != network.Connected
	}, "victim did not disconnect the malformed-frame sender")

	_ = attacker.host.Connect(context.Background(), pr.AddrInfo{
		ID: victim.host.ID(), Addrs: victim.host.Addrs(),
	})
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.host.Network().Connectedness(attacker.host.ID()) != network.Connected
	}, "penalized peer reconnected before its penalty expired")
	requireNoContent(t, victimHandler.C, 250*time.Millisecond)

	honestMessage := blockMessage("victim-still-available")
	require.Equal(t, honestMessage.Content,
		sendAndReceive(t, honest, victim.ID(), victimHandler.C, honestMessage))
	requireNoContent(t, attackerHandler.C, 250*time.Millisecond)
	requireNoContent(t, honestHandler.C, 250*time.Millisecond)
}

func TestMalformedReusablePayloadIsolatedToSendingPeer(t *testing.T) {
	const victimPort, attackerPort, honestPort = 11001, 12001, 13001
	victim := newMessenger(t, generatePrivateKey(t), nil, victimPort)
	victimHandler := registerBlockHandler(victim, 2)
	attacker := newMessenger(t, generatePrivateKey(t), []string{seedAddress(victim, victimPort)}, attackerPort)
	attackerHandler := registerBlockHandler(attacker, 2)
	honest := newMessenger(t, generatePrivateKey(t), []string{seedAddress(victim, victimPort)}, honestPort)
	honestHandler := registerBlockHandler(honest, 2)
	addSeedPeer(victim, attacker)
	addSeedPeer(victim, honest)
	startMessengers(t, victim, attacker, honest)

	warmup := blockMessage("streams-ready")
	require.Equal(t, warmup.Content, sendAndReceive(t, attacker, victim.ID(), victimHandler.C, warmup))
	require.Equal(t, warmup.Content, sendAndReceive(t, honest, victim.ID(), victimHandler.C, warmup))

	attacker.msgHandlerMap[common.ChannelIDBlock] = &rawEncodingHandler{attackerHandler}
	malformed := p2ptypes.Message{ChannelID: common.ChannelIDBlock, Content: []byte{0xff}}
	require.True(t, attacker.Send(victim.ID(), malformed))

	waitForCondition(t, 3*time.Second, func() bool {
		return victim.isPeerPenalized(attacker.host.ID(), time.Now())
	}, "victim did not penalize the malformed-payload sender")
	waitForCondition(t, 3*time.Second, func() bool {
		return victim.host.Network().Connectedness(attacker.host.ID()) != network.Connected
	}, "victim did not disconnect the malformed-payload sender")
	requireNoContent(t, victimHandler.C, 250*time.Millisecond)

	honestMessage := blockMessage("honest-after-malformed-payload")
	require.Equal(t, honestMessage.Content,
		sendAndReceive(t, honest, victim.ID(), victimHandler.C, honestMessage))
	requireNoContent(t, honestHandler.C, 250*time.Millisecond)
}

func TestReusableStreamAccountingReleasedAfterRemoteClose(t *testing.T) {
	const port1, port2 = 11001, 12001
	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	node2 := newMessenger(t, generatePrivateKey(t), nil, port2)
	victim, remote := node1, node2
	if victim.ID() > remote.ID() {
		victim, remote = remote, victim
	}
	victimHandler := registerBlockHandler(victim, 1)
	addSeedPeer(remote, victim)
	startMessengers(t, victim)
	// Add after Start so the victim accepts the inbound connection without
	// racing the designated outbound peer to establish a duplicate one.
	addSeedPeer(victim, remote)
	startMessengers(t, remote)
	streamID := protocol.ID(victim.protocolPrefix + strconv.Itoa(int(common.ChannelIDBlock)))

	waitForCondition(t, 8*time.Second, func() bool {
		victim.securityLock.Lock()
		defer victim.securityLock.Unlock()
		usage := victim.inboundStreamUsage[remote.host.ID()]
		return usage != nil && usage.channels[common.ChannelIDBlock] == 1
	}, "victim did not account for the inbound reusable stream")
	remotePeer := remote.peerTable.GetPeer(victim.host.ID())
	require.NotNil(t, remotePeer)
	remotePeer.StopStream(common.ChannelIDBlock)
	waitForCondition(t, 3*time.Second, func() bool {
		victim.securityLock.Lock()
		defer victim.securityLock.Unlock()
		return victim.inboundStreamUsage[remote.host.ID()] == nil
	}, "victim did not release accounting after the remote stream closed")
	require.False(t, victim.isPeerPenalized(remote.host.ID(), time.Now()))

	replacement, err := remote.host.NewStream(context.Background(), victim.host.ID(), streamID)
	require.NoError(t, err)
	sender := transport.NewBufferedStreamWithMaxMessageSize(
		replacement, nil, common.MaxBlockMessageSize)
	sender.Start(context.Background())
	defer sender.Stop()
	content := []byte("replacement-stream-works")
	encoded, err := victimHandler.EncodeMessage(content)
	require.NoError(t, err)
	written, err := sender.Write(encoded)
	require.NoError(t, err)
	require.Equal(t, len(encoded), written)
	require.Equal(t, content, receiveContent(t, victimHandler.C, 8*time.Second))
	waitForCondition(t, 3*time.Second, func() bool {
		victim.securityLock.Lock()
		defer victim.securityLock.Unlock()
		usage := victim.inboundStreamUsage[remote.host.ID()]
		return usage != nil && usage.channels[common.ChannelIDBlock] == 1
	}, "victim rejected a replacement stream after ordinary EOF")
}

func TestMalformedPubsubMessageDoesNotStopSubscription(t *testing.T) {
	const port1, port2 = 11001, 12001
	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	registerBlockHandler(node1, 2)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 2)
	addSeedPeer(node1, node2)
	startMessengers(t, node1, node2)

	time.Sleep(1500 * time.Millisecond)
	topic := node1.protocolPrefix + strconv.Itoa(int(common.ChannelIDBlock))
	require.NoError(t, node1.pubsub.Publish(topic, []byte{0xff}))
	time.Sleep(250 * time.Millisecond)

	message := blockMessage("valid-after-malformed-pubsub")
	require.NoError(t, node1.Publish(message))
	require.Equal(t, message.Content, receiveContent(t, handler2.C, 8*time.Second))
}

func TestPartiallyConnected(t *testing.T) {
	const port1, port2, port3 = 11001, 12001, 13001

	node1 := newMessenger(t, generatePrivateKey(t), nil, port1)
	handler1 := registerBlockHandler(node1, 4)
	node2 := newMessenger(t, generatePrivateKey(t), []string{seedAddress(node1, port1)}, port2)
	handler2 := registerBlockHandler(node2, 4)
	node3 := newMessenger(t, generatePrivateKey(t), nil, port3)
	handler3 := registerBlockHandler(node3, 4)
	addSeedPeer(node1, node2)
	startMessengers(t, node1, node2, node3)

	warmup := blockMessage("connected-pair")
	require.Equal(t, warmup.Content, sendAndReceive(t, node1, node2.ID(), handler2.C, warmup))
	require.Equal(t, warmup.Content, sendAndReceive(t, node2, node1.ID(), handler1.C, warmup))
	require.False(t, node1.Send(node3.ID(), warmup))
	require.False(t, node2.Send(node3.ID(), warmup))
	require.False(t, node3.Send(node1.ID(), warmup))
	require.False(t, node3.Send(node2.ID(), warmup))

	time.Sleep(1500 * time.Millisecond)
	broadcast := blockMessage("connected-only")
	node1.Broadcast(broadcast, false)
	require.Equal(t, broadcast.Content, receiveContent(t, handler2.C, 8*time.Second))
	requireNoContent(t, handler3.C, 500*time.Millisecond)

	require.NoError(t, node3.Publish(blockMessage("isolated")))
	requireNoContent(t, handler1.C, 250*time.Millisecond)
	requireNoContent(t, handler2.C, 250*time.Millisecond)
}
