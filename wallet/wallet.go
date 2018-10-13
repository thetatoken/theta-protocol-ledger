package wallet

import (
	sw "github.com/thetatoken/ukulele/wallet/softwallet"
	"github.com/thetatoken/ukulele/wallet/types"
)

func NewWallet(keysDirPath string, walletType types.WalletType) (types.Wallet, error) {
	var wallet types.Wallet
	var err error

	if walletType == types.WalletTypeSoft {
		wallet, err = sw.NewSoftWallet(keysDirPath, sw.KeystoreTypeEncrypted)
	} else {
		panic("Cold wallet not supported yet!") // TODO: support for cold wallet
	}
	if err != nil {
		return nil, err
	}

	return wallet, nil
}
