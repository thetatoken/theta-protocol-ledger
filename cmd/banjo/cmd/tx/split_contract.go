package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// splitContractCmd represents the release fund command
// Example:
//		banjo tx split_contract --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --seq=8 --resource_id=die_another_day --addresses=2E833968E5bB786Ae419c4d13189fB081Cc43bab,9F1233798E905E173560071255140b4A8aBd3Ec6 --percentages=30,30 --chain="" --duration=1000
var splitContractCmd = &cobra.Command{
	Use:   "split_contract",
	Short: "Initiate or update a split contract",
	Run:   doSplitContractCmd,
}

func doSplitContractCmd(cmd *cobra.Command, args []string) {
	cfgPath := cmd.Flag("config").Value.String()
	wallet, fromAddress, fromPubKey, err := walletUnlockAddress(cfgPath, fromFlag)
	if err != nil {
		return
	}
	defer wallet.Lock(fromAddress)

	input := types.TxInput{
		Address:  fromAddress,
		Sequence: uint64(seqFlag),
	}
	if seqFlag == 1 {
		input.PubKey = fromPubKey
	}

	if len(addressesFlag) != len(percentagesFlag) {
		fmt.Println("Should have the same number of addresses and percentages")
		return
	}
	var splits []types.Split
	for idx, addressStr := range addressesFlag {
		percentageStr := percentagesFlag[idx]

		address, err := hex.DecodeString(addressStr)
		if err != nil {
			fmt.Println("The address must be a hex string")
			return
		}

		percentage, err := strconv.ParseUint(percentageStr, 10, 32)
		if err != nil {
			fmt.Println(err)
			return
		}

		split := types.Split{
			Address:    common.BytesToAddress(address),
			Percentage: uint(percentage),
		}
		splits = append(splits, split)
	}

	splitContractTx := &types.SplitContractTx{
		Fee: types.Coins{
			ThetaWei: big.NewInt(0),
			GammaWei: big.NewInt(feeInGammaFlag),
		},
		Gas:        gasAmountFlag,
		ResourceID: common.Bytes(resourceIDFlag),
		Initiator:  input,
		Duration:   durationFlag,
		Splits:     splits,
	}

	sig, err := wallet.Sign(fromAddress, splitContractTx.SignBytes(chainIDFlag))
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	splitContractTx.SetSignature(fromAddress, sig)

	raw, err := types.TxToBytes(splitContractTx)
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
	fmt.Printf("Successfully broadcasted transaction.\n")
}

func init() {
	splitContractCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	splitContractCmd.Flags().StringVar(&fromFlag, "from", "", "Initiator's address")
	splitContractCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	splitContractCmd.Flags().Uint64Var(&gasAmountFlag, "gas", 1, "Gas limit")
	splitContractCmd.Flags().Int64Var(&feeInGammaFlag, "fee", 1, "Fee limit")
	splitContractCmd.Flags().StringVar(&resourceIDFlag, "resource_id", "", "The resourceID of interest")
	splitContractCmd.Flags().StringSliceVar(&addressesFlag, "addresses", []string{}, "List of addresses participating in the split")
	splitContractCmd.Flags().StringSliceVar(&percentagesFlag, "percentages", []string{}, "List of integers (between 0 and 100) representing of percentage of split")
	splitContractCmd.Flags().Uint64Var(&durationFlag, "duration", 1000, "Reserve duration")

	splitContractCmd.MarkFlagRequired("chain")
	splitContractCmd.MarkFlagRequired("from")
	splitContractCmd.MarkFlagRequired("seq")
	splitContractCmd.MarkFlagRequired("addresses")
	splitContractCmd.MarkFlagRequired("percentages")
	splitContractCmd.MarkFlagRequired("resource_id")
	splitContractCmd.MarkFlagRequired("duration")

}
