package tx

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/wallet"
	wtypes "github.com/thetatoken/theta/wallet/types"
)

func walletUnlock(cmd *cobra.Command, addressStr string) (wtypes.Wallet, common.Address, error) {
	var wallet wtypes.Wallet
	var address common.Address
	var err error
	walletType := getWalletType(cmd)
	if walletType == wtypes.WalletTypeSoft {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, address, err = SoftWalletUnlock(cfgPath, addressStr)
	} else {
		wallet, address, err = ColdWalletUnlock(walletType)
	}
	return wallet, address, err
}

func ColdWalletUnlock(walletType wtypes.WalletType) (wtypes.Wallet, common.Address, error) {
	wallet, err := wallet.OpenWallet("", walletType, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	err = wallet.Unlock(common.Address{}, "")
	if err != nil {
		fmt.Printf("Failed to unlock wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	addresses, err := wallet.List()
	if err != nil {
		fmt.Printf("Failed to list wallet addresses: %v\n", err)
		return nil, common.Address{}, err
	}

	if len(addresses) == 0 {
		errMsg := fmt.Sprintf("No address detected in the wallet\n")
		fmt.Printf(errMsg)
		return nil, common.Address{}, fmt.Errorf(errMsg)
	}
	address := addresses[0]

	log.Infof("Wallet address: %v", address)

	return wallet, address, nil
}

func SoftWalletUnlock(cfgPath, addressStr string) (wtypes.Wallet, common.Address, error) {
	wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	prompt := fmt.Sprintf("Please enter password: ")
	password, err := utils.GetPassword(prompt)
	if err != nil {
		fmt.Printf("Failed to get password: %v\n", err)
		return nil, common.Address{}, err
	}

	address := common.HexToAddress(addressStr)
	err = wallet.Unlock(address, password)
	if err != nil {
		fmt.Printf("Failed to unlock address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, err
	}

	return wallet, address, nil
}

func getWalletType(cmd *cobra.Command) (walletType wtypes.WalletType) {
	walletTypeStr := cmd.Flag("wallet").Value.String()
	if walletTypeStr == "nano" {
		walletType = wtypes.WalletTypeColdNano
	} else if walletTypeStr == "trezor" {
		walletType = wtypes.WalletTypeColdTrezor
	} else {
		walletType = wtypes.WalletTypeSoft
	}
	return walletType
}
