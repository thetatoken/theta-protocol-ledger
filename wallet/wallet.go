package wallet

import (
	"path"

	sw "github.com/thetatoken/ukulele/wallet/softwallet"
	"github.com/thetatoken/ukulele/wallet/types"
)

func OpenDefaultWallet(cfgPath string) (types.Wallet, error) {
	wallet, err := openWallet(cfgPath, types.WalletTypeSoft, true)
	return wallet, err
}

func openWallet(cfgPath string, walletType types.WalletType, encrypted bool) (types.Wallet, error) {
	var wallet types.Wallet
	var err error

	keysDirPath := path.Join(cfgPath, "keys")
	if walletType == types.WalletTypeSoft {
		if encrypted {
			wallet, err = sw.NewSoftWallet(keysDirPath, sw.KeystoreTypeEncrypted)
		} else {
			wallet, err = sw.NewSoftWallet(keysDirPath, sw.KeystoreTypePlain)
		}
	} else {
		panic("Cold wallet not supported yet!") // TODO: support for cold wallet
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}
