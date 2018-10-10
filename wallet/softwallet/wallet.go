package softwallet

import (
	"fmt"
	"sync"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	ks "github.com/thetatoken/ukulele/wallet/softwallet/keystore"
	"github.com/thetatoken/ukulele/wallet/types"
)

var _ types.Wallet = (*SoftWallet)(nil)

//
// SoftWallet implements the Wallet interface
//

type SoftWallet struct {
	keyMgr KeyManager
}

func (w *SoftWallet) Open(passphrase string) error {
	return nil
}

func (w *SoftWallet) Close() error {
	return nil
}

func (w *SoftWallet) NewKey(passphrase string) (common.Address, error) {
	_, address, err := w.keyMgr.NewKey(passphrase)
	return address, err
}

func (w *SoftWallet) UpdatePassphrase(address common.Address, oldPassphrase, newPassphrase string) error {
	err := w.keyMgr.UpdatePassphrase(address, oldPassphrase, newPassphrase)
	return err
}

func (w *SoftWallet) Derive(path types.DerivationPath, pin bool) (common.Address, error) {
	return common.Address{}, fmt.Errorf("Not supported for software wallet")
}

func (w *SoftWallet) Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error) {
	key, err := w.keyMgr.GetUnlockedKey(address)
	if err != nil {
		return nil, err
	}
	signature, err := key.Sign(txrlp)
	return signature, err
}

//
// KeyManager manages the keys
//

type KeyManager struct {
	mu             sync.RWMutex
	keystore       ks.Keystore
	unlockedKeyMap map[common.Address]*UnlockedKey // Currently unlocked keys (decrypted private keys)
}

type UnlockedKey struct {
	*ks.Key
	abort chan struct{}
}

func NewKeyManager(keysDirPath string) (KeyManager, error) {
	keystore, err := ks.NewKeystorePlain(keysDirPath)
	if err != nil {
		return KeyManager{}, nil
	}
	km := KeyManager{
		keystore: keystore,
	}
	km.initialize()
	return km, nil
}

func (km KeyManager) NewKey(passphrase string) (*ks.Key, common.Address, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	privKey, _, err := crypto.GenerateKeyPair()
	if err != nil {
		return nil, common.Address{}, err
	}

	key := ks.NewKey(privKey)
	address := key.Address()

	km.keystore.StoreKey(key, passphrase)

	return key, address, nil
}

func (km KeyManager) GetUnlockedKey(address common.Address) (*ks.Key, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	unlockedKey, found := km.unlockedKeyMap[address]
	if !found {
		return nil, fmt.Errorf("Key not unlocked yet for address: %v", address)
	}

	return unlockedKey.Key, nil
}

func (km KeyManager) Unlock(address common.Address, passphrase string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	key, err := km.keystore.GetKey(address, passphrase)
	if err != nil {
		return err
	}

	unlockedKey := &UnlockedKey{
		Key: key,
	}
	km.unlockedKeyMap[address] = unlockedKey

	return nil
}

func (km KeyManager) UpdatePassphrase(address common.Address, oldPassphrase, newPassphrase string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	key, err := km.keystore.GetKey(address, oldPassphrase)
	if err != nil {
		return err
	}

	err = km.keystore.StoreKey(key, newPassphrase)
	return err
}

func (km KeyManager) initialize() {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.unlockedKeyMap = make(map[common.Address]*UnlockedKey)
}
