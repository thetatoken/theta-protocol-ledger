package peer

import (
	"context"
	"net"
	"sync/atomic"
	"testing"
	"time"

	pr "github.com/libp2p/go-libp2p-core/peer"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/p2pl/transport"
)

func TestPeerChannelsIncludeCommitCertificate(t *testing.T) {
	require.Contains(t, Channels, common.ChannelIDCC)
}

func TestPeerStopCancelsDelayedStreamOpening(t *testing.T) {
	originalReuse := viper.GetBool(common.CfgP2PReuseStream)
	originalDelay := openStreamsDelay
	viper.Set(common.CfgP2PReuseStream, true)
	openStreamsDelay = 20 * time.Millisecond
	t.Cleanup(func() {
		viper.Set(common.CfgP2PReuseStream, originalReuse)
		openStreamsDelay = originalDelay
	})

	p := CreatePeer(pr.AddrInfo{ID: pr.ID("outbound-peer")}, true)
	var calls int32
	p.SetStreamCreator(func(common.ChannelIDEnum) (*transport.BufferedStream, error) {
		atomic.AddInt32(&calls, 1)
		return nil, nil
	})
	p.Start(context.Background())
	require.NoError(t, p.OpenStreams())
	p.Stop()

	time.Sleep(2 * openStreamsDelay)
	require.Zero(t, atomic.LoadInt32(&calls))
}

func TestPeerStopStreamRemovesStream(t *testing.T) {
	left, right := net.Pipe()
	defer right.Close()

	p := CreatePeer(pr.AddrInfo{ID: pr.ID("peer")}, false)
	stream := transport.NewBufferedStreamWithMaxMessageSize(left, nil, 64)
	p.streamMap[common.ChannelIDBlock] = stream

	p.StopStream(common.ChannelIDBlock)
	require.NotContains(t, p.streamMap, common.ChannelIDBlock)
}
