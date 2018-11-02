package call

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	rpcc "github.com/ybbus/jsonrpc"

	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"
)

// smartContractCmd represents the smart_contract command, which can be used to calls the specified smart contract.
// However, calling a smart contract does NOT modify the globally consensus state. It can be used for dry run, or
// for retrieving info from smart contracts without actually spending gas.
// Examples:
//   * Deploy a smart contract (local only)
//		banjo call smart_contract --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=1680 --gas_price=3 --gas_limit=50000 --data=600a600c600039600a6000f3600360135360016013f3
//   * Call an API of a smart contract (local only)
//		banjo call smart_contract --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x7ad6cea2bc3162e30a3c98d84f821b3233c22647 --gas_price=3 --gas_limit=50000 --data=1234

var smartContractCmd = &cobra.Command{
	Use:   "smart_contract",
	Short: "Call or deploy a smart contract",
	Run:   doSmartContractCmd,
}

func doSmartContractCmd(cmd *cobra.Command, args []string) {
	from := types.TxInput{
		Address: common.HexToAddress(fromFlag),
		Coins: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			GammaWei: new(big.Int).SetUint64(valueFlag),
		},
		Sequence: seqFlag,
	}
	to := types.TxOutput{
		Address: common.HexToAddress(toFlag),
	}
	data, err := hex.DecodeString(dataFlag)
	if err != nil {
		fmt.Printf("Failed to decode data: %v\n", dataFlag)
		return
	}

	sctx := &types.SmartContractTx{
		From:     from,
		To:       to,
		GasLimit: gasLimitFlag,
		GasPrice: new(big.Int).SetUint64(gasPriceFlag),
		Data:     data,
	}

	sctxBytes, err := types.TxToBytes(sctx)
	if err != nil {
		fmt.Printf("Failed to encode smart contract transaction: %v\n", sctx)
		return
	}

	rpcCallArgs := rpc.CallSmartContractArgs{
		SctxBytes: hex.EncodeToString(sctxBytes),
	}

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.CallSmartContract", rpcCallArgs)
	if err != nil {
		fmt.Printf("Failed to call smart contract: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Failed to execute smart contract: %v\n", res.Error)
		return
	}
	json, err := json.MarshalIndent(res.Result, "", "    ")
	if err != nil {
		fmt.Printf("Failed to parse server response: %v\n", err)
	}
	fmt.Println(string(json))
}

func init() {
	smartContractCmd.Flags().StringVar(&fromFlag, "from", "", "The caller address")
	smartContractCmd.Flags().StringVar(&toFlag, "to", "", "The smart contract address")
	smartContractCmd.Flags().Uint64Var(&valueFlag, "value", 0, "Value to be transferred")
	smartContractCmd.Flags().Uint64Var(&gasPriceFlag, "gas_price", 0, "The gas price")
	smartContractCmd.Flags().Uint64Var(&gasLimitFlag, "gas_limit", 0, "The gas limit")
	smartContractCmd.Flags().StringVar(&dataFlag, "data", "", "The data for the smart contract")
	smartContractCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")

	smartContractCmd.MarkFlagRequired("from")
	smartContractCmd.MarkFlagRequired("gas_price")
	smartContractCmd.MarkFlagRequired("gas_limit")
}
