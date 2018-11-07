package tx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"

	rpcc "github.com/ybbus/jsonrpc"
)

// smartContractCmd represents the smart_contract command. It will submit a smart contract transaction
// to the blockchain, which will modify the global consensus state when it is included in the blockchain
// Examples:
//   * Deploy a smart contract
//		banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=1680 --gas_price=3 --gas_limit=50000 --data=600a600c600039600a6000f3600360135360016013f3 --seq=1
//   * Call an API of a smart contract
//		banjo tx smart_contract --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x7ad6cea2bc3162e30a3c98d84f821b3233c22647 --gas_price=3 --gas_limit=50000 --seq=2

var smartContractCmd = &cobra.Command{
	Use:   "smart_contract",
	Short: "Call or deploy a smart contract",
	Run:   doSmartContractCmd,
}

func doSmartContractCmd(cmd *cobra.Command, args []string) {
	cfgPath := cmd.Flag("config").Value.String()
	wallet, fromAddress, fromPubKey, err := walletUnlockAddress(cfgPath, fromFlag)
	if err != nil {
		return
	}
	defer wallet.Lock(fromAddress)

	from := types.TxInput{
		Address: common.HexToAddress(fromFlag),
		Coins: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			GammaWei: new(big.Int).SetUint64(valueFlag),
		},
		Sequence: seqFlag,
	}
	if seqFlag == 1 {
		from.PubKey = fromPubKey
	}

	to := types.TxOutput{
		Address: common.HexToAddress(toFlag),
	}
	data, err := hex.DecodeString(dataFlag)
	if err != nil {
		return
	}

	smartContractTx := &types.SmartContractTx{
		From:     from,
		To:       to,
		GasLimit: gasLimitFlag,
		GasPrice: new(big.Int).SetUint64(gasPriceFlag),
		Data:     data,
	}

	sig, err := wallet.Sign(fromAddress, smartContractTx.SignBytes(chainIDFlag))
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	smartContractTx.SetSignature(fromAddress, sig)

	raw, err := types.TxToBytes(smartContractTx)
	if err != nil {
		fmt.Printf("Failed to encode transaction: %v\n", err)
		return
	}
	signedTx := hex.EncodeToString(raw)

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.BroadcastRawTransaction", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	if err != nil {
		fmt.Printf("Failed to broadcast transaction: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Server returned error: %v\n", res.Error)
		return
	}
	result := &rpc.BroadcastRawTransactionResult{}
	err = res.GetObject(result)
	if err != nil {
		fmt.Printf("Failed to parse server response: %v\n", err)
		return
	}
	formatted, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		fmt.Printf("Failed to parse server response: %v\n", err)
		return
	}
	fmt.Printf("Successfully broadcasted transaction:\n%s\n", formatted)
}

func init() {
	smartContractCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	smartContractCmd.Flags().StringVar(&fromFlag, "from", "", "The caller address")
	smartContractCmd.Flags().StringVar(&toFlag, "to", "", "The smart contract address")
	smartContractCmd.Flags().Uint64Var(&valueFlag, "value", 0, "Value to be transferred")
	smartContractCmd.Flags().Uint64Var(&gasPriceFlag, "gas_price", types.MinimumGasPrice, "The gas price")
	smartContractCmd.Flags().Uint64Var(&gasLimitFlag, "gas_limit", 0, "The gas limit")
	smartContractCmd.Flags().StringVar(&dataFlag, "data", "", "The data for the smart contract")
	smartContractCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")

	smartContractCmd.MarkFlagRequired("chain")
	smartContractCmd.MarkFlagRequired("from")
	smartContractCmd.MarkFlagRequired("gas_price")
	smartContractCmd.MarkFlagRequired("gas_limit")
	smartContractCmd.MarkFlagRequired("seq")
}
