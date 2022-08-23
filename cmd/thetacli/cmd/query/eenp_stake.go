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

// eenpStakeCmd represents the eenp stake command.
// Example:
//		thetacli query eenp_stake --height=10
var eenpStakeCmd = &cobra.Command{
	Use:     "eenp_stake",
	Short:   "Get eenp stake",
	Example: `thetacli query eenp_stake --height=10`,
	Run:     doEenpStakeCmd,
}

func doEenpStakeCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.GetEenpStakeByHeight", rpc.GetEenpStakeByHeightArgs{
		Height:        common.JSONUint64(heightFlag),
		Source:        common.HexToAddress(sourceFlag),
		Holder:        common.HexToAddress(holderFlag),
		WithdrawnOnly: withdrawnOnlyFlag,
	})
	if err != nil {
		utils.Error("Failed to get eenp stake: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get eenp stake: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	eenpStakeCmd.Flags().StringVar(&sourceFlag, "source", "", "Source of the stake")
	eenpStakeCmd.Flags().StringVar(&holderFlag, "holder", "", "Holder of the stake")
	eenpStakeCmd.Flags().BoolVar(&withdrawnOnlyFlag, "withdrawn_only", true, "Only want withdrawn stake")
	eenpStakeCmd.Flags().Uint64Var(&heightFlag, "height", uint64(0), "height of the block")
	eenpStakeCmd.MarkFlagRequired("source")
	eenpStakeCmd.MarkFlagRequired("holder")
	eenpStakeCmd.MarkFlagRequired("height")
}
