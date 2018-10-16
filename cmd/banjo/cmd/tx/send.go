package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// sendCmd represents the send command
// Example:
//		banjo tx send --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --gamma=900000 --seq=2
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send tokens",
	Run:   doSendCmd,
}

func doSendCmd(cmd *cobra.Command, args []string) {
	cfgPath := cmd.Flag("config").Value.String()
	wallet, fromAddress, fromPubKey, err := utils.WalletUnlockAddress(cfgPath, fromFlag)
	if err != nil {
		return
	}
	defer wallet.Lock(fromAddress)

	inputs := []types.TxInput{{
		Address: fromAddress,
		Coins: types.Coins{
			GammaWei: big.NewInt(gammaAmountFlag + feeInGammaFlag),
			ThetaWei: big.NewInt(thetaAmountFlag),
		},
		Sequence: uint64(seqFlag),
	}}
	if seqFlag == 1 {
		inputs[0].PubKey = fromPubKey
	}
	outputs := []types.TxOutput{{
		Address: common.HexToAddress(toFlag),
		Coins: types.Coins{
			GammaWei: big.NewInt(gammaAmountFlag),
			ThetaWei: big.NewInt(thetaAmountFlag),
		},
	}}
	sendTx := &types.SendTx{
		Fee: types.Coins{
			ThetaWei: big.NewInt(0),
			GammaWei: big.NewInt(feeInGammaFlag),
		},
		Gas:     gasAmountFlag,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig, err := wallet.Sign(fromAddress, sendTx.SignBytes(chainIDFlag))
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	sendTx.SetSignature(fromAddress, sig)

	raw, err := types.TxToBytes(sendTx)
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
	sendCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	sendCmd.Flags().StringVar(&fromFlag, "from", "", "Address to send from")
	sendCmd.Flags().StringVar(&toFlag, "to", "", "Address to send to")
	sendCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	sendCmd.Flags().Int64Var(&thetaAmountFlag, "theta", 0, "Theta amount in Wei")
	sendCmd.Flags().Int64Var(&gammaAmountFlag, "gamma", 0, "Gamma amount in Wei")
	sendCmd.Flags().Uint64Var(&gasAmountFlag, "gas", 1, "Gas limit")
	sendCmd.Flags().Int64Var(&feeInGammaFlag, "fee", 1, "Fee limit")

	sendCmd.MarkFlagRequired("chain")
	sendCmd.MarkFlagRequired("from")
	sendCmd.MarkFlagRequired("to")
	sendCmd.MarkFlagRequired("seq")
}
