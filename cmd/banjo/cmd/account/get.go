package account

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/rpc"
	"github.com/thetatoken/ukulele/wallet"
	rpcc "github.com/ybbus/jsonrpc"
)

var (
	addressFlag string
)

// getCmd represents the get command.
// Example:
//		banjo account get --address=0x2E833968E5bB786Ae419c4d13189fB081Cc43bab
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get account status",
	Long:  `Get account status.`,
	Run:   doGetCmd,
}

func doGetCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(wallet.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.GetAccount", rpc.GetAccountArgs{Address: addressFlag})
	if err != nil {
		fmt.Printf("Failed to get account details: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Failed to get account details: %v\n", res.Error)
		return
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		fmt.Printf("Failed to parse server response: %v\n", err)
	}
	fmt.Println(string(json))
}

func init() {
	getCmd.Flags().StringVar(&addressFlag, "address", "", "Address of the account")
	getCmd.MarkFlagRequired("address")
}
