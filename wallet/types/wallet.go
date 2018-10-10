package types

import (
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

type Wallet interface {
	Open(passphrase string) error
	Close() error
	NewKey(passphrase string) (common.Address, error)
	UpdatePassphrase(address common.Address, oldPassphrase, newPassphrase string) error
	Derive(path DerivationPath, pin bool) (common.Address, error)
	Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error)
}
