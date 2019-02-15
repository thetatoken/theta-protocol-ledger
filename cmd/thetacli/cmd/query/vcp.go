package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// vcpCmd represents the vcp command.
// Example:
//		thetacli query vcp --height=10
var vcpCmd = &cobra.Command{
	Use:     "vcp",
	Short:   "Get validator candidate pool",
	Example: `thetacli query vcp --height=10`,
	Run:     doVcpCmd,
}

func doVcpCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	height := heightFlag
	res, err := client.Call("theta.GetVcpByHeight", rpc.GetVcpByHeightArgs{Height: common.JSONUint64(height)})
	if err != nil {
		utils.Error("Failed to get validator candidate pool: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get validator candidate pool: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	vcpCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block")
	vcpCmd.MarkFlagRequired("height")
}
