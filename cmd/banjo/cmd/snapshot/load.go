package snapshot

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

// loadCmd represents the load snapshot command.
// Example:
//		banjo snapshot load
var loadCmd = &cobra.Command{
	Use:     "load",
	Short:   "load snapshot",
	Long:    `Load snapshot.`,
	Example: `banjo snapshot load`,
	Run:     doLoadCmd,
}

func doLoadCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.LoadSnapshot", rpc.GenSnapshotArgs{})
	if err != nil {
		utils.Error("Failed to get load snapshot call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get load snapshot res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
}
