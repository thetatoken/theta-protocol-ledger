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

// blockCmd represents the block command.
// Example:
//		thetacli query block --height=300
//		thetacli query block --hash=0xc88485a473527c55c5ddb067b018324b7e390b188e76702bc1db74dfc2dc6d13
//
var blockCmd = &cobra.Command{
	Use:     "block",
	Short:   "Get block details",
	Long:    `Get block details.`,
	Example: `thetacli query block --height=300`,
	Run: func(cmd *cobra.Command, args []string) {
		client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

		var res *jsonrpc.RPCResponse
		var err error
		if len(hashFlag) != 0 {
			res, err = client.Call("theta.GetBlock", rpc.GetBlockArgs{
				Hash: common.HexToHash(hashFlag),
			})
		} else if endFlag != 0 {
			res, err = client.Call("theta.GetBlocksByRange", rpc.GetBlocksByRangeArgs{
				Start: common.JSONUint64(startFlag),
				End: common.JSONUint64(endFlag),
			})
		} else {
			res, err = client.Call("theta.GetBlockByHeight", rpc.GetBlockByHeightArgs{
				Height: common.JSONUint64(heightFlag),
			})
		}

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
	blockCmd.Flags().StringVar(&hashFlag, "hash", "", "Block hash")
	blockCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block")
	blockCmd.Flags().Uint64Var(&startFlag, "start", uint64(0), "starting height of the blocks")
	blockCmd.Flags().Uint64Var(&endFlag, "end", uint64(0), "ending height of the blocks")
}
