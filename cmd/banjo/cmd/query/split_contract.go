package query

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/rpc"
	"github.com/thetatoken/ukulele/wallet"
	rpcc "github.com/ybbus/jsonrpc"
)

var (
	resourceIDFlag string
)

// splitContractCmd represents the split_contract command.
// Example:
//		banjo query account --address=0x2E833968E5bB786Ae419c4d13189fB081Cc43bab
var splitContractCmd = &cobra.Command{
	Use:   "split_contract",
	Short: "Get split contract status",
	Run:   doSplitContractCmd,
}

func doSplitContractCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(wallet.CfgRemoteRPCEndpoint))

	resourceID := hex.EncodeToString(common.Bytes(resourceIDFlag))
	res, err := client.Call("theta.GetSplitContract", rpc.GetSplitContractArgs{ResourceID: resourceID})
	if err != nil {
		fmt.Printf("Failed to get split contract details: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Failed to get split contract details: %v\n", res.Error)
		return
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		fmt.Printf("Failed to parse server response: %v\n", err)
	}
	fmt.Println(string(json))
}

func init() {
	splitContractCmd.Flags().StringVar(&resourceIDFlag, "resource_id", "", "Resource ID of the contract")
	splitContractCmd.MarkFlagRequired("resource_id")
}
