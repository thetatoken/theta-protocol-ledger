package keystore

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pborman/uuid"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
)

type Key struct {
	Id         uuid.UUID
	Address    common.Address
	PrivateKey *crypto.PrivateKey
}

func NewKey(privKey *crypto.PrivateKey) *Key {
	Id := uuid.NewRandom()
	return &Key{
		Id:         Id,
		Address:    privKey.PublicKey().Address(),
		PrivateKey: privKey,
	}
}

func (key *Key) Sign(data common.Bytes) (*crypto.Signature, error) {
	sig, err := key.PrivateKey.Sign(data)
	return sig, err
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
