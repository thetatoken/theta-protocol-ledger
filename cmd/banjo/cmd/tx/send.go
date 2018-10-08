package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"path"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"
	"github.com/thetatoken/ukulele/wallet"
	rpcc "github.com/ybbus/jsonrpc"
)

var (
	chainIDFlag     string
	fromFlag        string
	toFlag          string
	seqFlag         uint64
	thetaAmountFlag int64
	gammaAmountFlag int64
	gasAmountFlag   uint64
	feeInGammaFlag  int64
)

// sendCmd represents the new command
var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Send tokens",
	Long:  `Send tokens.`,
	Run:   doSendCmd,
}

func doSendCmd(cmd *cobra.Command, args []string) {
	cfgPath := cmd.Flag("config").Value.String()
	privKey, err := loadPrivateKey(cfgPath, fromFlag)
	if err != nil {
		fmt.Printf("Failed to load key for address %v: %v\n", fromFlag, err)
		return
	}

	fromAddress := privKey.PublicKey().Address()
	inputs := []types.TxInput{{
		Address: fromAddress,
		Coins: types.Coins{
			GammaWei: big.NewInt(gammaAmountFlag + feeInGammaFlag),
			ThetaWei: big.NewInt(thetaAmountFlag),
		},
		Sequence: uint64(seqFlag),
	}}
	if seqFlag == 1 {
		inputs[0].PubKey = privKey.PublicKey()
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
			GammaWei: big.NewInt(feeInGammaFlag),
		},
		Gas:     gasAmountFlag,
		Inputs:  inputs,
		Outputs: outputs,
	}

	sig, err := privKey.Sign(sendTx.SignBytes(chainIDFlag))
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	sendTx.SetSignature(fromAddress, sig)

	signedTx := hex.EncodeToString(types.TxToBytes(sendTx))

	client := rpcc.NewRPCClient(viper.GetString(wallet.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.BroadcastRawTransaction", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	if err != nil {
		fmt.Printf("Failed to broadcast transaction: %v\n", err)
		return
	}
	if res.Error != nil {
		fmt.Printf("Server returned error: %v\n", res.Error)
		return
	}
	fmt.Printf("Successfully broadcasted transaction:\n%v\n", res.Result)
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

func loadPrivateKey(cfgPath string, address string) (*crypto.PrivateKey, error) {
	if strings.HasPrefix(address, "0x") {
		address = address[2:]
	}
	filePath := path.Join(cfgPath, "keys", address)
	return crypto.PrivateKeyFromFile(filePath)
}
