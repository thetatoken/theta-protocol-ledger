package coldwallet

import (
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	ks "github.com/thetatoken/ukulele/wallet/coldwallet/keystore"
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

func (w *ColdWallet) List() ([]common.Address, error) {
	return nil, fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) NewKey(password string) (common.Address, error) {
	return common.Address{}, fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Unlock(address common.Address, password string) error {
	return fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Lock(address common.Address) error {
	return fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Delete(address common.Address, password string) error {
	return fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) UpdatePassword(address common.Address, oldPassword, newPassword string) error {
	return fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Derive(path types.DerivationPath, pin bool) (common.Address, error) {
	return common.Address{}, fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) GetPublicKey(address common.Address) (*crypto.PublicKey, error) {
	return nil, fmt.Errorf("Not supported for cold wallet")
}

func (w *ColdWallet) Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error) {
	return nil, fmt.Errorf("Not supported for cold wallet")
}

//
// DriverManager manages the hardware drivers
//

type DriverManager struct {
	driver ks.Driver
}
