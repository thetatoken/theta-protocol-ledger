package tx

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rpc"
	wtypes "github.com/thetatoken/theta/wallet/types"

	"github.com/ybbus/jsonrpc"
	rpcc "github.com/ybbus/jsonrpc"
)

// sendCmd represents the send command
// Example:
//		thetacli tx send --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --tfuel=9 --seq=1
//		thetacli tx send --chain="privatenet" --path "m/44'/60'/0'/0/0" --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --tfuel=9 --seq=1 --wallet=trezor
//		thetacli tx send --chain="privatenet" --path "m/44'/60'/0'/0" --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --tfuel=9 --seq=1 --wallet=nano
var sendCmd = &cobra.Command{
	Use:     "send",
	Short:   "Send tokens",
	Example: `thetacli tx send --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --theta=10 --tfuel=9 --seq=1`,
	Run:     doSendCmd,
}

func doSendCmd(cmd *cobra.Command, args []string) {
	walletType := getWalletType(cmd)
	if walletType == wtypes.WalletTypeSoft && len(fromFlag) == 0 {
		utils.Error("The from address cannot be empty") // we don't need to specify the "from address" for hardware wallets
		return
	}

	if len(toFlag) == 0 {
		utils.Error("The to address cannot be empty")
		return
	}
	if fromFlag == toFlag {
		utils.Error("The from and to address cannot be identical")
		return
	}

	wallet, fromAddress, err := walletUnlockWithPath(cmd, fromFlag, pathFlag)
	if err != nil || wallet == nil {
		return
	}
	defer wallet.Lock(fromAddress)

	theta, ok := types.ParseCoinAmount(thetaAmountFlag)
	if !ok {
		utils.Error("Failed to parse theta amount")
	}
	tfuel, ok := types.ParseCoinAmount(tfuelAmountFlag)
	if !ok {
		utils.Error("Failed to parse tfuel amount")
	}
	fee, ok := types.ParseCoinAmount(feeFlag)
	if !ok {
		utils.Error("Failed to parse fee")
	}
	inputs := []types.TxInput{{
		Address: fromAddress,
		Coins: types.Coins{
			TFuelWei: new(big.Int).Add(tfuel, fee),
			ThetaWei: theta,
		},
		Sequence: uint64(seqFlag),
	}}
	outputs := []types.TxOutput{{
		Address: common.HexToAddress(toFlag),
		Coins: types.Coins{
			TFuelWei: tfuel,
			ThetaWei: theta,
		},
	}}
	sendTx := &types.SendTx{
		Fee: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: fee,
		},
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig, err := wallet.Sign(fromAddress, sendTx.SignBytes(chainIDFlag))
	if err != nil {
		utils.Error("Failed to sign transaction: %v\n", err)
	}
	sendTx.SetSignature(fromAddress, sig)

	raw, err := types.TxToBytes(sendTx)
	if err != nil {
		utils.Error("Failed to encode transaction: %v\n", err)
	}
	signedTx := hex.EncodeToString(raw)

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	var res *jsonrpc.RPCResponse
	if asyncFlag {
		res, err = client.Call("theta.BroadcastRawTransactionAsync", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	} else {
		res, err = client.Call("theta.BroadcastRawTransaction", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	}

	if err != nil {
		utils.Error("Failed to broadcast transaction: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Server returned error: %v\n", res.Error)
	}
	result := &rpc.BroadcastRawTransactionResult{}
	err = res.GetObject(result)
	if err != nil {
		utils.Error("Failed to parse server response: %v\n", err)
	}
	formatted, err := json.MarshalIndent(result, "", "    ")
	if err != nil {
		utils.Error("Failed to parse server response: %v\n", err)
	}
	fmt.Printf("Successfully broadcasted transaction:\n%s\n", formatted)
}

func init() {
	sendCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	sendCmd.Flags().StringVar(&fromFlag, "from", "", "Address to send from")
	sendCmd.Flags().StringVar(&toFlag, "to", "", "Address to send to")
	sendCmd.Flags().StringVar(&pathFlag, "path", "", "Wallet derivation path")
	sendCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	sendCmd.Flags().StringVar(&thetaAmountFlag, "theta", "0", "Theta amount")
	sendCmd.Flags().StringVar(&tfuelAmountFlag, "tfuel", "0", "TFuel amount")
	sendCmd.Flags().StringVar(&feeFlag, "fee", fmt.Sprintf("%dwei", types.MinimumTransactionFeeTFuelWei), "Fee")
	sendCmd.Flags().StringVar(&walletFlag, "wallet", "soft", "Wallet type (soft|nano|trezor)")
	sendCmd.Flags().BoolVar(&asyncFlag, "async", false, "block until tx has been included in the blockchain")

	sendCmd.MarkFlagRequired("chain")
	//sendCmd.MarkFlagRequired("from")
	sendCmd.MarkFlagRequired("to")
	sendCmd.MarkFlagRequired("seq")
}
