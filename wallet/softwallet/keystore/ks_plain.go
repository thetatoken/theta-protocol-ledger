package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/thetatoken/ukulele/common"
)

var _ Keystore = (*KeystorePlain)(nil)

type KeystorePlain struct {
	keysDirPath string
}

func NewKeystorePlain(keysDirPath string) (KeystorePlain, error) {
	err := os.MkdirAll(keysDirPath, 0700)
	if err != nil {
		return KeystorePlain{}, err
	}

	fi, err := os.Lstat(keysDirPath)
	if err != nil {
		return KeystorePlain{}, err
	}
	if fi.Mode().Perm() != 0700 {
		return KeystorePlain{}, fmt.Errorf("%s must have permission set to 0700", keysDirPath)
	}

	ks := KeystorePlain{
		keysDirPath: keysDirPath,
	}

	return ks, nil
}

func (ks KeystorePlain) GetKey(address common.Address, auth string) (*Key, error) {
	filePath := ks.getFilePath(address)
	fd, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	key := new(Key)
	if err := json.NewDecoder(fd).Decode(key); err != nil {
		return nil, err
	}
	if key.Address() != address {
		return nil, fmt.Errorf("key content mismatch: have address %x, want %x", key.Address, address)
	}
	return key, nil
}

func (ks KeystorePlain) StoreKey(key *Key, auth string) error {
	address := key.Address()
	filePath := ks.getFilePath(address)
	content, err := json.Marshal(key)
	if err != nil {
		return err
	}
	return writeKeyFile(filePath, content)
}

func (ks KeystorePlain) getFilePath(address common.Address) string {
	filePath := path.Join(ks.keysDirPath, address.Hex()[2:])
	return filePath
}
