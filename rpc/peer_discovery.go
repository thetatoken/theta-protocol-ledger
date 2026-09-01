package rpc

// ThetaRPCPeerDiscoveryService is a restricted RPC service that exposes ONLY the
// peer-discovery method (theta.GetPeerURLs). It is registered on a separate,
// optionally-enabled listener (see common.CfgRPCPeerDiscoveryEnabled and
// common.CfgRPCPeerDiscoveryPort) so operators can keep peer discovery reachable
// by edge nodes while firewalling the full RPC surface served on rpc.port.
//
// Only GetPeerURLs is exported on this type. Do NOT embed *ThetaRPCService here:
// net/rpc exposes every exported method of the registered receiver, so embedding
// would promote and expose the entire RPC surface on the peer-discovery port,
// defeating the purpose of the separate listener.
type ThetaRPCPeerDiscoveryService struct {
	svc *ThetaRPCService
}

// GetPeerURLs delegates to the main service so the peer-selection logic — and the
// logRPCArgs call — live in one place (see ThetaRPCService.GetPeerURLs). Do not add
// a logRPCArgs call here: the delegate already logs, so a call would be logged twice.
func (s *ThetaRPCPeerDiscoveryService) GetPeerURLs(args *GetPeersArgs, result *GetPeerURLsResult) error {
	return s.svc.GetPeerURLs(args, result)
}
