package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thetatoken/theta/common"
)

type AddressFormat int

const (
	allLowerCase AddressFormat = iota
	mixedCase
	allUpperCase
)

type Keystore interface {

	// List the addresses of all the stored keys
	ListKeyAddresses() ([]common.Address, error)

	// Loads and decrypts the key from disk.
	GetKey(address common.Address, auth string) (*Key, error)

	// Writes and encrypts the key.
	StoreKey(k *Key, auth string) error

	// Deletes the key from the disk.
	DeleteKey(address common.Address, auth string) error
}

func writeKeyFile(file string, content common.Bytes) error {
	// Create the keystore directory with appropriate permissions
	// in case it is not present yet.
	const dirPerm = 0700
	if err := os.MkdirAll(filepath.Dir(file), dirPerm); err != nil {
		return err
	}
	// Atomic write: create a temporary hidden file first
	// then move it into place. TempFile assigns mode 0600.
	f, err := ioutil.TempFile(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return err
	}
	if _, err := f.Write(content); err != nil {
		f.Close()
		os.Remove(f.Name())
		return err
	}
	f.Close()
	return os.Rename(f.Name(), file)
}

func deleteKeyFile(file string) error {
	err := os.Remove(file)
	return err
}
