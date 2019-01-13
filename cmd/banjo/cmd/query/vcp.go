package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

var (
	heightFlag uint64
)

// vcpCmd represents the vcp command.
// Example:
//		banjo query vcp --height=10
var vcpCmd = &cobra.Command{
	Use:     "vcp",
	Short:   "Get validator candidate pool",
	Example: `banjo query vcp --height=10`,
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
