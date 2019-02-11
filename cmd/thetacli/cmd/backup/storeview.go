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
	heightFlag uint64
)

// storeviewCmd represents the storeview dump command.
// Example:
//		thetacli backup snapshot
var storeviewCmd = &cobra.Command{
	Use:     "storeview",
	Short:   "backup storeview",
	Long:    `Backup storeview.`,
	Example: `thetacli backup storeview`,
	Run:     doStoreviewCmd,
}

func doStoreviewCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.DumpStoreview", rpc.DumpStoreviewArgs{Config: configFlag, Height: heightFlag})
	if err != nil {
		utils.Error("Failed to get dump storeview call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get dump storeview res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	storeviewCmd.Flags().StringVar(&configFlag, "config", "", "Config dir")
	storeviewCmd.Flags().Uint64Var(&heightFlag, "height", 0, "Block height")
	storeviewCmd.MarkFlagRequired("config")
	storeviewCmd.MarkFlagRequired("height")
}
