package wallet

import (
	"fmt"
	"path"

	cw "github.com/thetatoken/ukulele/wallet/coldwallet"
	sw "github.com/thetatoken/ukulele/wallet/softwallet"
	"github.com/thetatoken/ukulele/wallet/types"
)

func OpenWallet(cfgPath string, walletType types.WalletType, encrypted bool) (types.Wallet, error) {
	var wallet types.Wallet
	var err error

	keysDirPath := path.Join(cfgPath, "keys")
	if walletType == types.WalletTypeSoft {
		if encrypted {
			wallet, err = sw.NewSoftWallet(keysDirPath, sw.KeystoreTypeEncrypted)
		} else {
			wallet, err = sw.NewSoftWallet(keysDirPath, sw.KeystoreTypePlain)
		}
		if err != nil {
			return nil, err
		}
	} else {
		hub, err := cw.NewLedgerHub() // only support Ledger Nano S for now
		if err != nil {
			return nil, err
		}
		wallets := hub.Wallets()
		if len(wallets) == 0 {
			return nil, fmt.Errorf("No cold wallet detected")
		}
		if len(wallets) > 1 {
			return nil, fmt.Errorf("Multiple cold wallets detected, for now we only support one cold wallet at a time")
		}
		wallet = wallets[0]
	}

	return wallet, nil
}
