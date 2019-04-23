package types

import (
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

type WalletType int

const (
	WalletTypeSoft WalletType = iota
	WalletTypeColdNano
	WalletTypeColdTrezor
)

type Wallet interface {
	ID() string
	Status() (string, error)
	List() ([]common.Address, error)
	NewKey(password string) (common.Address, error)
	Unlock(address common.Address, password string, derivationPath DerivationPath) error
	Lock(address common.Address) error
	IsUnlocked(address common.Address) bool
	Delete(address common.Address, password string) error
	UpdatePassword(address common.Address, oldPassword, newPassword string) error
	Derive(path DerivationPath, pin bool) (common.Address, error)
	GetPublicKey(address common.Address) (*crypto.PublicKey, error)
	Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error)
}
