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
	startFlag  uint64
	endFlag    uint64
	configFlag string
)

// generateCmd represents the generate backup command.
// Example:
//		thetacli backup generate
var generateCmd = &cobra.Command{
	Use:     "generate",
	Short:   "generate backup",
	Long:    `Generate backup.`,
	Example: `thetacli backup generate`,
	Run:     doGenerateCmd,
}

func doGenerateCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.GenBackup", rpc.BackupArgs{Start: startFlag, End: endFlag, Config: configFlag})
	if err != nil {
		utils.Error("Failed to get generate backup call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get generate backup res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	generateCmd.Flags().Uint64Var(&startFlag, "start", 0, "Starting block height")
	generateCmd.Flags().Uint64Var(&endFlag, "end", 0, "Ending block height")
	generateCmd.Flags().StringVar(&configFlag, "config", "", "Config dir")
	generateCmd.MarkFlagRequired("start")
	generateCmd.MarkFlagRequired("end")
	generateCmd.MarkFlagRequired("config")
}
