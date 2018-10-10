package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

type Key struct {
	privateKey *crypto.PrivateKey
}

func NewKey(privKey *crypto.PrivateKey) *Key {
	return &Key{
		privateKey: privKey,
	}
}

func (key *Key) Sign(data common.Bytes) (*crypto.Signature, error) {
	sig, err := key.privateKey.Sign(data)
	return sig, err
}

func (key *Key) Address() common.Address {
	return key.privateKey.PublicKey().Address()
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
