package query

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// splitRuleCmd represents the split_rule command.
// Example:
//		thetacli query split_rule --resource_id=vid2dz369du0mkwcrb9
var splitRuleCmd = &cobra.Command{
	Use:     "split_rule",
	Short:   "Get split rule status",
	Example: `thetacli query split_rule --resource_id=vid2dz369du0mkwcrb9`,
	Run:     doSplitRuleCmd,
}

func doSplitRuleCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	resourceID := resourceIDFlag
	res, err := client.Call("theta.GetSplitRule", rpc.GetSplitRuleArgs{ResourceID: resourceID})
	if err != nil {
		utils.Error("Failed to get split rule details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get split rule details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%s\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	splitRuleCmd.Flags().StringVar(&resourceIDFlag, "resource_id", "", "Resource ID of the contract")
	splitRuleCmd.MarkFlagRequired("resource_id")
}
