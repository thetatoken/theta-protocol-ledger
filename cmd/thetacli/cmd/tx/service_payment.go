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

// servicePaymentCmd represents the send command
// Example:
//		thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --payment_seq=1 --reserve_seq=1 --resource_id=rid1000001
var servicePaymentCmd = &cobra.Command{
	Use:     "service_payment",
	Short:   "Make Service Payment from Reserve fund",
	Example: `thetacli tx service_payment --chain="privatenet" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab --to=9F1233798E905E173560071255140b4A8aBd3Ec6 --payment_seq=1 --reserve_seq=1 --resource_id=rid1000001`,
	Run:     doServicePaymentCmd,
}

func doServicePaymentCmd(cmd *cobra.Command, args []string) {
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

	fee, ok := types.ParseCoinAmount(feeFlag)
	if !ok {
		utils.Error("Failed to parse fee")
	}

	sinput := types.TxInput{
		Address: fromAddress,
		Coins: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: new(big.Int).SetUint64(0),
		},
		Sequence: uint64(paymentSeqFlag),
	}

	tinput := types.TxInput{
		Address: common.HexToAddress(toFlag),
		Coins: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: new(big.Int).SetUint64(0),
		},
		//Sequence: uint64(paymentSeqFlag),
	}

	// See theta-protocol-ledger > ledger > types > tx.go : Line 522
	servicePaymentTx := &types.ServicePaymentTx{
		Fee: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: fee,
		},
		Source:     sinput,
		Target:     tinput,
		PaymentSequence: paymentSeqFlag,
		ReserveSequence: reserveSeqFlag,
		ResourceID: resourceIDFlag,
	}

	ssig, err := wallet.Sign(fromAddress, servicePaymentTx.SourceSignBytes(chainIDFlag))
	if err != nil {
		utils.Error("Failed to sign source transaction: %v\n", err)
	}
	servicePaymentTx.SetSourceSignature(ssig)

	tsig, err := wallet.Sign(tinput.Address, servicePaymentTx.TargetSignBytes(chainIDFlag))
	if err != nil {
		utils.Error("Failed to sign target transaction: %v\n", err)
	}
	servicePaymentTx.SetTargetSignature(tsig)

	raw, err := types.TxToBytes(servicePaymentTx)
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
	servicePaymentCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	servicePaymentCmd.Flags().StringVar(&fromFlag, "from", "", "Address to send from")
	servicePaymentCmd.Flags().StringVar(&toFlag, "to", "", "Address to send to")
	servicePaymentCmd.Flags().StringVar(&pathFlag, "path", "", "Wallet derivation path")
	servicePaymentCmd.Flags().Uint64Var(&paymentSeqFlag, "payment_seq", 0, "Payment sequence number of the transaction")
	servicePaymentCmd.Flags().Uint64Var(&reserveSeqFlag, "reserve_seq", 0, "Reserve sequence number of the transaction")
	servicePaymentCmd.Flags().StringVar(&resourceIDFlag, "resource_id", "", "Corresponding resourceID")
	servicePaymentCmd.Flags().StringVar(&feeFlag, "fee", fmt.Sprintf("%dwei", types.MinimumTransactionFeeTFuelWei), "Fee")
	servicePaymentCmd.Flags().StringVar(&walletFlag, "wallet", "soft", "Wallet type (soft|nano|trezor)")
	servicePaymentCmd.Flags().BoolVar(&asyncFlag, "async", false, "block until tx has been included in the blockchain")

	servicePaymentCmd.MarkFlagRequired("chain")
	servicePaymentCmd.MarkFlagRequired("from")
	servicePaymentCmd.MarkFlagRequired("to")
	servicePaymentCmd.MarkFlagRequired("payment_seq")
	servicePaymentCmd.MarkFlagRequired("reserve_seq")
	servicePaymentCmd.MarkFlagRequired("resource_id")
}
