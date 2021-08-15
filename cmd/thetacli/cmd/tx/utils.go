package tx

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/wallet"
	"github.com/thetatoken/theta/wallet/types"
	wtypes "github.com/thetatoken/theta/wallet/types"
)

const HARDENED_FLAG = 1 << 31

func walletUnlock(cmd *cobra.Command, addressStr string, password string) (wtypes.Wallet, common.Address, error) {
	return walletUnlockWithPath(cmd, addressStr, "", password)
}

func walletUnlockWithPath(cmd *cobra.Command, addressStr string, path string, password string) (wtypes.Wallet, common.Address, error) {
	var wallet wtypes.Wallet
	var address common.Address
	var err error
	walletType := getWalletType(cmd)
	if walletType == wtypes.WalletTypeSoft {
		cfgPath := cmd.Flag("config").Value.String()
		wallet, address, err = SoftWalletUnlock(cfgPath, addressStr, password)
	} else {
		derivationPath, err := parseDerivationPath(path, walletType)
		if err != nil {
			return nil, common.Address{}, err
		}
		wallet, address, err = ColdWalletUnlock(walletType, derivationPath)
	}
	return wallet, address, err
}

func ColdWalletUnlock(walletType wtypes.WalletType, derivationPath types.DerivationPath) (wtypes.Wallet, common.Address, error) {
	wallet, err := wallet.OpenWallet("", walletType, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	err = wallet.Unlock(common.Address{}, "", derivationPath)
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

func SoftWalletUnlock(cfgPath, addressStr string, password string) (wtypes.Wallet, common.Address, error) {
	wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	if password == "" || len(password) == 0 {
		prompt := fmt.Sprintf("Please enter password: ")
		password, err = utils.GetPassword(prompt)
		if err != nil {
			fmt.Printf("Failed to get password: %v\n", err)
			return nil, common.Address{}, err
		}
	}

	address := common.HexToAddress(addressStr)
	err = wallet.Unlock(address, password, nil)
	if err != nil {
		fmt.Printf("Failed to unlock address %v: %v\n", address.Hex(), err)
		return nil, common.Address{}, err
	}

	return wallet, address, nil
}

func SoftWalletUnlockPW(cfgPath, addressStr string, password string) (wtypes.Wallet, common.Address, error) {
	wallet, err := wallet.OpenWallet(cfgPath, wtypes.WalletTypeSoft, true)
	if err != nil {
		fmt.Printf("Failed to open wallet: %v\n", err)
		return nil, common.Address{}, err
	}

	//prompt := fmt.Sprintf("Please enter pasword: ")
	//password, err := utils.GetPassword(prompt)
	//if err != nil {
	//	fmt.Printf("Failed to get password: %v\n", err)
	//	return nil, common.Address{}, err
	//}

	// password:= "qwertyuiop"

	address := common.HexToAddress(addressStr)
	err = wallet.Unlock(address, password, nil)
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

func parseDerivationPath(nstr string, walletType wtypes.WalletType) (types.DerivationPath, error) {
	if len(nstr) == 0 {
		if walletType == wtypes.WalletTypeColdNano {
			// nstr = "m/44'/60'/0'/0"
			return types.DefaultRootDerivationPath, nil
		} else if walletType == wtypes.WalletTypeColdTrezor {
			// nstr = "m/44'/60'/0'/0/0"
			return types.DefaultBaseDerivationPath, nil
		} else {
			return nil, fmt.Errorf("can't parse derivation path for soft wallet")
		}
	}

	n := strings.Split(nstr, "/")

	// m/a/b/c => a/b/c
	if n[0] == "m" {
		n = n[1:]
	}

	derivationPath := types.DerivationPath{}
	for _, i := range n {
		p, err := strToHarden(i)
		if err != nil {
			return derivationPath, err
		}
		derivationPath = append(derivationPath, uint32(p))
	}

	return derivationPath, nil
}

func strToHarden(x string) (int, error) {
	if strings.HasPrefix(x, "-") {
		i, err := strconv.Atoi(x)
		if err != nil {
			return 0, err
		}
		return H(int(math.Abs(float64(i)))), nil
	} else if strings.HasSuffix(x, "h") || strings.HasSuffix(x, "'") {
		i, err := strconv.Atoi(x[:len(x)-1])
		if err != nil {
			return 0, err
		}
		return H(i), nil
	}
	return strconv.Atoi(x)
}

// Shortcut function that "hardens" a number in a BIP44 path.
func H(x int) int {
	return x | HARDENED_FLAG
}
