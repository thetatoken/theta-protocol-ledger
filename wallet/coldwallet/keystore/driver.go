package keystore

import (
	"io"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/wallet/types"
)

// Driver abstracts the functionality of the hardware wallet
type Driver interface {
	Status() (string, error)
	Open(device io.ReadWriter, password string) error
	Close() error
	Heartbeat() error
	Derive(path types.DerivationPath) (common.Address, error)
	SignTx(path types.DerivationPath, txrlp common.Bytes) (common.Address, *crypto.Signature, error)
}
