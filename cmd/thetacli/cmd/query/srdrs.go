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

// srdrsCmd represents the eenp command.
// Example:
//		thetacli query eenp --height=10
var srdrsCmd = &cobra.Command{
	Use:     "srdrs",
	Short:   "Get stake reward distribution rule set",
	Example: `thetacli query srdrs --height=10`,
	Run:     doSrdrsCmd,
}

func doSrdrsCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))
	height := heightFlag
	res, err := client.Call("theta.GetStakeRewardDistributionByHeight", rpc.GetStakeRewardDistributionRuleSetByHeightArgs{
		Height:  common.JSONUint64(height),
		Address: addressFlag,
	})
	if err != nil {
		utils.Error("Failed to get stake reward distribution rule set: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get stake reward distribution rule set: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	srdrsCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block")
	srdrsCmd.Flags().StringVar(&addressFlag, "address", "", "Address of the account")
	srdrsCmd.MarkFlagRequired("height")
}
