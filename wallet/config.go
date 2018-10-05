package wallet

import "github.com/spf13/viper"

const (
	CfgRemoteRPCEndpoint = "remoteRPCEndpoint"
)

func init() {
	viper.SetDefault(CfgRemoteRPCEndpoint, "http://localhost:16888/rpc")
}
