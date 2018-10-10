package coldwallet

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	dr "github.com/thetatoken/ukulele/wallet/coldwallet/driver"
	"github.com/thetatoken/ukulele/wallet/types"
)

var _ types.Wallet = (*ColdWallet)(nil)

//
// ColdWallet implements the Wallet interface
//

type ColdWallet struct {
	driverMgr *DriverManager
}

// TODO: to be implemented

func (w *ColdWallet) Open(passphrase string) error {
	return nil
}

func (w *ColdWallet) Close() error {
	return nil
}

func (w *ColdWallet) NewKey(passphrase string) (common.Address, error) {
	return common.Address{}, fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) UpdatePassphrase(address common.Address, oldPassphrase, newPassphrase string) error {
	return fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Derive(path types.DerivationPath, pin bool) (common.Address, error) {
	return common.Address{}, nil
}

func (w *ColdWallet) Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error) {
	return nil, nil
}

//
// DriverManager manages the hardware drivers
//

type DriverManager struct {
	driver dr.Driver
}
