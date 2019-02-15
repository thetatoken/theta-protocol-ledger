package query

import (
	"encoding/json"
	"fmt"

	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"
)

// txCmd represents the query tx command.
// Example:
//		thetacli query tx --hash=0x2fe41732b40ca852e9c36f52b278dde78f0fe34f28f9c94083112aa6a0624b8c
//
var txCmd = &cobra.Command{
	Use:     "tx",
	Short:   "Get transaction details",
	Long:    `Get transaction details.`,
	Example: `thetacli query tx --hash=0x2fe41732b40ca852e9c36f52b278dde78f0fe34f28f9c94083112aa6a0624b8c`,
	Run: func(cmd *cobra.Command, args []string) {
		client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))
		res, err := client.Call("theta.GetTransaction", rpc.GetTransactionArgs{
			Hash: hashFlag,
		})

		if err != nil {
			utils.Error("Failed to get transaction details: %v\n", err)
		}
		if res.Error != nil {
			utils.Error("Failed to retrieve transaction details: %v\n", res.Error)
		}
		json, err := json.MarshalIndent(res.Result, "", "    ")
		if err != nil {
			utils.Error("Failed to parse server response: %v\n%v\n", err, string(json))
		}
		fmt.Println(string(json))
	},
}

func init() {
	txCmd.Flags().StringVar(&hashFlag, "hash", "", "Block hash")
}
