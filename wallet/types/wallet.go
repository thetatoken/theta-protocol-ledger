package types

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

type WalletType int

const (
	WalletTypeSoft WalletType = iota
	WalletTypeCold
)

type Wallet interface {
	List() ([]common.Address, error)
	NewKey(password string) (common.Address, error)
	Unlock(address common.Address, password string) error
	Lock(address common.Address) error
	Delete(address common.Address, password string) error
	UpdatePassword(address common.Address, oldPassword, newPassword string) error
	Derive(path DerivationPath, pin bool) (common.Address, error)
	GetPublicKey(address common.Address) (*crypto.PublicKey, error)
	Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error)
}
