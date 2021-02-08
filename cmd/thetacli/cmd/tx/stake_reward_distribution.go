package tx

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rpc"

	rpcc "github.com/ybbus/jsonrpc"
)

// stakeRewardDistributionCmd represents the stake reward distribution command
// Example:
//		thetacli tx distribute_staking_reward --chain="privatenet" --holder=2E833968E5bB786Ae419c4d13189fB081Cc43bab --beneficiary=0x88884a84d980bbfb7588888126fb903486bb8888 --split_basis_point=100 --purpose=1 --seq=8
var stakeRewardDistributionCmd = &cobra.Command{
	Use:     "distribute_staking_reward",
	Short:   "Configure the distribution of the guardian/elite edge node staking reward",
	Example: `thetacli tx distribute_staking_reward --chain="privatenet" --holder=2E833968E5bB786Ae419c4d13189fB081Cc43bab --beneficiary=0x88884a84d980bbfb7588888126fb903486bb8888 --split_basis_point=100 --purpose=1 --seq=8`,
	Run:     doStakeRewardDistributionCmd,
}

func doStakeRewardDistributionCmd(cmd *cobra.Command, args []string) {
	wallet, holderAddress, err := walletUnlockWithPath(cmd, holderFlag, pathFlag)
	if err != nil {
		return
	}
	defer wallet.Lock(holderAddress)

	fee, ok := types.ParseCoinAmount(feeFlag)
	if !ok {
		utils.Error("Failed to parse fee")
	}

	holder := types.TxInput{
		Address:  holderAddress,
		Sequence: uint64(seqFlag),
	}
	beneficiary := types.TxOutput{
		Address: common.HexToAddress(beneficiaryFlag),
	}

	stakeRewardDistributionTx := &types.StakeRewardDistributionTx{
		Fee: types.Coins{
			ThetaWei: new(big.Int).SetUint64(0),
			TFuelWei: fee,
		},
		Holder:          holder,
		Beneficiary:     beneficiary,
		SplitBasisPoint: uint(splitBasisPointFlag),
		Purpose:         purposeFlag,
	}

	sig, err := wallet.Sign(holderAddress, stakeRewardDistributionTx.SignBytes(chainIDFlag))
	if err != nil {
		utils.Error("Failed to sign transaction: %v\n", err)
	}
	stakeRewardDistributionTx.SetSignature(holderAddress, sig)

	raw, err := types.TxToBytes(stakeRewardDistributionTx)
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
	withdrawStakeCmd.Flags().StringVar(&holderFlag, "holder", "", "Holder of the stake")
	withdrawStakeCmd.Flags().StringVar(&pathFlag, "path", "", "Wallet derivation path")
	withdrawStakeCmd.Flags().StringVar(&feeFlag, "fee", fmt.Sprintf("%dwei", types.MinimumTransactionFeeTFuelWei), "Fee")
	withdrawStakeCmd.Flags().Uint64Var(&seqFlag, "seq", 0, "Sequence number of the transaction")
	withdrawStakeCmd.Flags().StringVar(&beneficiaryFlag, "purpose", "", "Address of the beneficiary")
	withdrawStakeCmd.Flags().Uint64Var(&splitBasisPointFlag, "split_basis_point", 0, "fraction of the reward split in terms of basis point (1/10000). 100 basis point = 100/10000 = 1.00%")

	withdrawStakeCmd.Flags().Uint8Var(&purposeFlag, "purpose", 0, "Purpose of staking")
	withdrawStakeCmd.Flags().StringVar(&walletFlag, "wallet", "soft", "Wallet type (soft|nano)")

	withdrawStakeCmd.MarkFlagRequired("chain")
	withdrawStakeCmd.MarkFlagRequired("holder")
	withdrawStakeCmd.MarkFlagRequired("seq")
}
