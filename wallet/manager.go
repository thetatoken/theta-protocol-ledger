package wallet

import (
	sw "github.com/thetatoken/ukulele/wallet/softwallet"
	"github.com/thetatoken/ukulele/wallet/types"
)

type WalletManager struct {
	Wallet types.Wallet
}

func NewWalletManager(keysDirPath string, walletType types.WalletType) (*WalletManager, error) {
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

	walletManager := &WalletManager{
		Wallet: wallet,
	}

	return walletManager, nil
}
