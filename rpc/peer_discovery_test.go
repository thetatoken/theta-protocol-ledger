package rpc

import (
	"net/rpc"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// The peer-discovery listener must expose exactly one method: theta.GetPeerURLs.
// If someone later embeds *ThetaRPCService into ThetaRPCPeerDiscoveryService,
// net/rpc would promote and expose the entire RPC surface on the peer-discovery
// port; this test guards against that regression.
func TestPeerDiscoveryServiceExposesOnlyGetPeerURLs(t *testing.T) {
	typ := reflect.TypeOf(&ThetaRPCPeerDiscoveryService{})

	methods := []string{}
	for i := 0; i < typ.NumMethod(); i++ {
		methods = append(methods, typ.Method(i).Name)
	}

	require.Equal(t, []string{"GetPeerURLs"}, methods)
}

// net/rpc RegisterName returns an error when the receiver has no methods with a
// suitable RPC signature, so this confirms GetPeerURLs is registrable and that
// theta.GetPeerURLs is actually callable on the peer-discovery listener.
func TestPeerDiscoveryServiceRegisters(t *testing.T) {
	s := rpc.NewServer()
	require.NoError(t, s.RegisterName("theta", &ThetaRPCPeerDiscoveryService{}))
}
