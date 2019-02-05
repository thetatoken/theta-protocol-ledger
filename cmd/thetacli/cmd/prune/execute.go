package backup

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

var (
	startFlag uint64
	endFlag   uint64
)

// executeCmd represents the prune execute command.
// Example:
//		thetacli prune execute
var executeCmd = &cobra.Command{
	Use:     "prune",
	Short:   "execute prune",
	Long:    `Execute prune.`,
	Example: `thetacli prune execute`,
	Run:     doExecuteCmd,
}

func doExecuteCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.ExecutePrune", rpc.PruneArgs{Start: startFlag, End: endFlag})
	if err != nil {
		utils.Error("Failed to get execute prune call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get execute prune res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	executeCmd.Flags().Uint64Var(&startFlag, "start", 0, "Starting block height")
	executeCmd.Flags().Uint64Var(&endFlag, "end", 0, "Ending block height")
	executeCmd.MarkFlagRequired("start")
	executeCmd.MarkFlagRequired("end")
}
