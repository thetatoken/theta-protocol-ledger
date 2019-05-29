package backup

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

// correctionCmd represents the chain correction command.
// Example:
//		thetacli backup correction
var chainCorrectionCmd = &cobra.Command{
	Use:     "correction",
	Short:   "backup correction",
	Long:    `Backup correction.`,
	Example: `thetacli backup correction`,
	Run:     doChainCorrectionCmd,
}

func doChainCorrectionCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.BackupChainCorrection", rpc.BackupChainCorrectionArgs{RollbackHeight: heightFlag, EndBlockHash: common.HexToHash(hashFlag), Config: configFlag})
	if err != nil {
		utils.Error("Failed to get backup chain call details: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Failed to get backup chain res details: %v\n", res.Error)
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
	}
	fmt.Println(string(json))
}

func init() {
	chainCorrectionCmd.Flags().Uint64Var(&heightFlag, "rollback_height", 0, "Rollback block height")
	chainCorrectionCmd.Flags().StringVar(&hashFlag, "end_block_hash", "", "Ending block hash")
	chainCorrectionCmd.Flags().StringVar(&configFlag, "config", "", "Config dir")
	chainCorrectionCmd.MarkFlagRequired("rollback_height")
	chainCorrectionCmd.MarkFlagRequired("end_block_hash")
	chainCorrectionCmd.MarkFlagRequired("config")
}
