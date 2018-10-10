package keystore

import (
	"github.com/thetatoken/ukulele/common"
)

type Keystore interface {

	// Loads and decrypts the key from disk.
	GetKey(addr common.Address, auth string) (*Key, error)

	// Writes and encrypts the key.
	StoreKey(k *Key, auth string) error
}
