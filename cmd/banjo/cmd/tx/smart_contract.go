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
//		banjo tx smart_contract --from=0x2E833968E5bB786Ae419c4d13189fB081Cc43bab --value=68000000000 --gas_price=10000000000 --gas_limit=50000 --data=0x600160020160135360016013f3 --seq=1
//   * Call an API of a smart contract
//		banjo tx smart_contract --from=0x2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=0x3883f5e181fccaf8410fa61e12b59bad963fb645 --value=68000000000 --gas_price=10000000000 --gas_limit=50000 --data=0x78FADDC641DEF878 --seq=2

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
		fmt.Printf("Failed to call smart contract: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Failed to call smart contraact: %v\n", res.Error)
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
	smartContractCmd.MarkFlagRequired("seq")
}
