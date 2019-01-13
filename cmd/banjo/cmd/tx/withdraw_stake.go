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

// withdrawStakeCmd represents the withdraw stake command
// Example:
//		banjo tx withdraw --chain="" --source=2E833968E5bB786Ae419c4d13189fB081Cc43bab --holder=2E833968E5bB786Ae419c4d13189fB081Cc43bab --purpose=0 --seq=9
var withdrawStakeCmd = &cobra.Command{
	Use:     "withdraw",
	Short:   "withdraw stake to a validator or guardian",
	Example: `banjo tx withdraw --chain="" --source=2E833968E5bB786Ae419c4d13189fB081Cc43bab --holder=2E833968E5bB786Ae419c4d13189fB081Cc43bab --purpose=0 --seq=9`,
	Run:     doWithdrawStakeCmd,
}

func doWithdrawStakeCmd(cmd *cobra.Command, args []string) {
	wallet, sourceAddress, err := walletUnlock(cmd, sourceFlag)
	if err != nil {
		return
	}
	defer wallet.Lock(sourceAddress)

	fee, ok := types.ParseCoinAmount(feeFlag)
	if !ok {
		utils.Error("Failed to parse fee")
	}

	source := types.TxInput{
		Address:  sourceAddress,
		Sequence: uint64(seqFlag),
	}
	holder := types.TxOutput{
		Address: common.HexToAddress(holderFlag),
	}

	withdrawStakeTx := &types.WithdrawStakeTx{
		Fee: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			GammaWei: fee,
		},
		Source:  source,
		Holder:  holder,
		Purpose: purposeFlag,
	}

	sig, err := wallet.Sign(sourceAddress, withdrawStakeTx.SignBytes(chainIDFlag))
	if err != nil {
		utils.Error("Failed to sign transaction: %v\n", err)
	}
	withdrawStakeTx.SetSignature(sourceAddress, sig)

	raw, err := types.TxToBytes(withdrawStakeTx)
	if err != nil {
		utils.Error("Failed to encode transaction: %v\n", err)
	}
	signedTx := hex.EncodeToString(raw)

	client := rpcc.NewRPCClient(viper.GetString(utils.CfgRemoteRPCEndpoint))

	res, err := client.Call("theta.BroadcastRawTransaction", rpc.BroadcastRawTransactionArgs{TxBytes: signedTx})
	if err != nil {
		utils.Error("Failed to broadcast transaction: %v\n", err)
	}
	if res.Error != nil {
		utils.Error("Server returned error: %v\n", res.Error)
	}
	fmt.Printf("Successfully broadcasted transaction.\n")
}

func init() {
	withdrawStakeCmd.Flags().StringVar(&chainIDFlag, "chain", "", "Chain ID")
	withdrawStakeCmd.Flags().StringVar(&sourceFlag, "source", "", "Source of the stake")
	withdrawStakeCmd.Flags().StringVar(&holderFlag, "holder", "", "Holder of the stake")
	withdrawStakeCmd.Flags().StringVar(&feeFlag, "fee", fmt.Sprintf("%dwei", types.MinimumTransactionFeeGammaWei), "Fee")
	withdrawStakeCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	withdrawStakeCmd.Flags().Uint8Var(&purposeFlag, "purpose", 0, "Purpose of staking")
	withdrawStakeCmd.Flags().StringVar(&walletFlag, "wallet", "soft", "Wallet type (soft|nano)")

	withdrawStakeCmd.MarkFlagRequired("chain")
	withdrawStakeCmd.MarkFlagRequired("source")
	withdrawStakeCmd.MarkFlagRequired("holder")
	withdrawStakeCmd.MarkFlagRequired("seq")
}
