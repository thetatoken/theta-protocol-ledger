package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/ukulele/ledger/types"
	"github.com/thetatoken/ukulele/rpc"
	"github.com/thetatoken/ukulele/wallet"
	rpcc "github.com/ybbus/jsonrpc"
)

// releaseFundCmd represents the release fund command
// Example:
//		banjo tx release --chain="" --from=2E833968E5bB786Ae419c4d13189fB081Cc43bab  --reserve_seq=8 --seq=8
var releaseFundCmd = &cobra.Command{
	Use:   "release",
	Short: "Release fund",
	Run:   doReleaseFundCmd,
}

func doReleaseFundCmd(cmd *cobra.Command, args []string) {
	cfgPath := cmd.Flag("config").Value.String()
	privKey, err := loadPrivateKey(cfgPath, fromFlag)
	if err != nil {
		fmt.Printf("Failed to load key for address %v: %v\n", fromFlag, err)
		return
	}

	fromAddress := privKey.PublicKey().Address()
	input := types.TxInput{
		Address:  fromAddress,
		Sequence: uint64(seqFlag),
	}
	if seqFlag == 1 {
		input.PubKey = privKey.PublicKey()
	}

	releaseFundTx := &types.ReleaseFundTx{
		Fee: types.Coins{
			ThetaWei: big.NewInt(0),
			GammaWei: big.NewInt(feeInGammaFlag),
		},
		Gas:             gasAmountFlag,
		Source:          input,
		ReserveSequence: reserveSeqFlag,
	}

	sig, err := privKey.Sign(releaseFundTx.SignBytes(chainIDFlag))
	if err != nil {
		fmt.Printf("Failed to sign transaction: %v\n", err)
		return
	}
	releaseFundTx.SetSignature(fromAddress, sig)

	raw, err := types.TxToBytes(releaseFundTx)
	if err != nil {
		fmt.Printf("Failed to encode transaction: %v\n", err)
		return
	}
	signedTx := hex.EncodeToString(raw)

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
	fmt.Printf("Successfully broadcasted transaction.\n")
}

func init() {
	releaseFundCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	releaseFundCmd.Flags().StringVar(&fromFlag, "from", "", "Reserve owner's address")
	releaseFundCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	releaseFundCmd.Flags().Uint64Var(&gasAmountFlag, "gas", 1, "Gas limit")
	releaseFundCmd.Flags().Int64Var(&feeInGammaFlag, "fee", 1, "Fee limit")
	releaseFundCmd.Flags().Uint64Var(&reserveSeqFlag, "reserve_seq", 1000, "Reserve sequence")

	releaseFundCmd.MarkFlagRequired("chain")
	releaseFundCmd.MarkFlagRequired("from")
	releaseFundCmd.MarkFlagRequired("seq")
	releaseFundCmd.MarkFlagRequired("reserve_seq")
	releaseFundCmd.MarkFlagRequired("resource_id")

}
