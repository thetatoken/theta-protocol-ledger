package query

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

var (
	addressFlag string
)

// accountCmd represents the account command.
// Example:
//		banjo query account --address=0x2E833968E5bB786Ae419c4d13189fB081Cc43bab
var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Get account status",
	Long:  `Get account status.`,
	Run:   doAccountCmd,
}

func doAccountCmd(cmd *cobra.Command, args []string) {
	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

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
	accountCmd.Flags().StringVar(&addressFlag, "address", "", "Address of the account")
	accountCmd.MarkFlagRequired("address")
}
