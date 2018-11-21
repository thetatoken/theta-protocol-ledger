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

func walletUnlock(cmd *cobra.Command, addressStr string) (
	wallet wtypes.Wallet, address common.Address, pubKey *crypto.PublicKey, err error) {
	walletType := getWalletType(cmd)
	if walletType == wtypes.WalletTypeSoft {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, address, pubKey, err = softWalletUnlock(cfgPath, addressStr)
	} else {
		wallet, address, pubKey, err = coldWalletUnlock()
	}
	return wallet, address, pubKey, err
}

func coldWalletUnlock() (wtypes.Wallet, common.Address, *crypto.PublicKey, error) {
	wallet, err := wallet.OpenWallet("", wtypes.WalletTypeCold, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	addresses, err := wallet.List()
	if err != nil {
		fmt.Printf("Failed to list wallet addresses: %v\n", err)
		return nil, common.Address{}, nil, err
	}

	if len(addresses) == 0 {
		errMsg := fmt.Sprintf("No address detected in the wallet\n")
		return nil, common.Address{}, nil, fmt.Errorf(errMsg)
	}
	address := addresses[0]
	pubKey := (*crypto.PublicKey)(nil) // TODO: remove the pubkey requirement for tx verification

	return wallet, address, pubKey, nil
}

func softWalletUnlock(cfgPath, addressStr string) (wtypes.Wallet, common.Address, *crypto.PublicKey, error) {
	wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
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

func getWalletType(cmd *cobra.Command) (walletType wtypes.WalletType) {
	walletTypeStr := cmd.Flag("wallet").Value.String()
	if walletTypeStr == "nano" {
		walletType = wtypes.WalletTypeCold
	} else {
		walletType = wtypes.WalletTypeSoft
	}
	return walletType
}
