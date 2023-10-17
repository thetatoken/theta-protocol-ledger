package query

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/common"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/ybbus/jsonrpc"
	rpcc "github.com/ybbus/jsonrpc"
)

// traceBlocksCmd represents the block command.
// Example:
//		thetacli query chain_of_blocks --upstream=true --hash=0xc88485a473527c55c5ddb067b018324b7e390b188e76702bc1db74dfc2dc6d13
//
var traceBlocksCmd = &cobra.Command{
	Use:     "chain_of_blocks",
	Short:   "Get chain of blocks",
	Long:    `Get chain of blocks.`,
	Example: `thetacli query chain_of_blocks --upstream=true --hash=0xc88485a473527c55c5ddb067b018324b7e390b188e76702bc1db74dfc2dc6d13`,
	Run: func(cmd *cobra.Command, args []string) {
		client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

		var res *jsonrpc.RPCResponse
		var err error
		res, err = client.Call("theta.TraceBlocks", rpc.TraceBlocksArgs{
			Hash:     common.HexToHash(hashFlag),
			Upstream: upstreamFlag,
		})

		if err != nil {
			utils.Error("Failed to get block(s) details: %v\n", err)
		}
		if res.Error != nil {
			utils.Error("Failed to retrieve block(s) details: %v\n", res.Error)
		}
		json, err := json.MarshalIndent(res.Result, "", "    ")
		if err != nil {
			utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
		}
		fmt.Println(string(json))
	},
}

func init() {
	traceBlocksCmd.Flags().StringVar(&hashFlag, "hash", "", "Block hash")
	traceBlocksCmd.Flags().BoolVar(&upstreamFlag, "upstream", true, "trace the chain of block upward")
}
