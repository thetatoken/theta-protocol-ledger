package snapshot

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
	configFlag string
)

// exportCmd represents the export snapshot command.
// Example:
//		thetacli snapshot export
var exportCmd = &cobra.Command{
	Use:     "export",
	Short:   "export snapshot",
	Long:    `Export snapshot.`,
	Example: `thetacli snapshot export`,
	Run:     doExportCmd,
}

func doExportCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.GenSnapshot", rpc.GenSnapshotArgs{Config: configFlag})
	if err != nil {
		utils.Error("Failed to get export snapshot call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get export snapshot res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	exportCmd.Flags().StringVar(&configFlag, "config", "", "Config dir")
	exportCmd.MarkFlagRequired("config")
}
