package tx

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/thetatoken/ukulele/cmd/banjo/cmd/utils"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/wallet"
	wtypes "github.com/thetatoken/ukulele/wallet/types"
)

// Common flags used in Tx sub commands.
var (
	chainIDFlag                  string
	fromFlag                     string
	toFlag                       string
	seqFlag                      uint64
	thetaAmountFlag              int64
	gammaAmountFlag              int64
	gasAmountFlag                uint64
	feeInGammaFlag               int64
	resourceIDsFlag              []string
	resourceIDFlag               string
	durationFlag                 uint64
	reserveFundInGammaFlag       int64
	reserveCollateralInGammaFlag int64
	reserveSeqFlag               uint64
	addressesFlag                []string
	percentagesFlag              []string
	valueFlag                    uint64
	gasPriceFlag                 uint64
	gasLimitFlag                 uint64
	dataFlag                     string
)

// TxCmd represents the Tx command
var TxCmd = &cobra.Command{
	Use:   "tx",
	Short: "Manage transactions",
	Long:  `Manage transactions.`,
}

func init() {
	TxCmd.AddCommand(sendCmd)
	TxCmd.AddCommand(reserveFundCmd)
	//TxCmd.AddCommand(releaseFundCmd) // No need for releaseFundCmd since auto-release is already implemented
	TxCmd.AddCommand(splitRuleCmd)
	TxCmd.AddCommand(smartContractCmd)
}

func walletUnlockAddress(cfgPath, addressStr string) (wtypes.Wallet, common.Address, *crypto.PublicKey, error) {
	wallet, err := wallet.OpenDefaultWallet(cfgPath)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	prompt := fmt.Sprintf("Please enter password: ")
	password, err := utils.GetPassword(prompt)
	if err != nil {
		fmt.Printf("Failed to get password: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	address := common.HexToAddress(addressStr)
	err = wallet.Unlock(address, password)
	if err != nil {
		fmt.Printf("Failed to unlock address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, nil, err
	}

	pubKey, err := wallet.GetPublicKey(address)
	if err != nil {
		fmt.Printf("Failed to get the public key for address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, nil, err
	}

	return wallet, address, pubKey, nil
}
