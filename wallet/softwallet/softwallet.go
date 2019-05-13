package softwallet

import (
	"fmt"
	"sync"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	ks "github.com/thetatoken/theta/wallet/softwallet/keystore"
	"github.com/thetatoken/theta/wallet/types"
)

var _ types.Wallet = (*SoftWallet)(nil)

type KeystoreType int

const (
	KeystoreTypeEncrypted KeystoreType = iota
	KeystoreTypePlain
)

//
// SoftWallet implements the Wallet interface
//

type SoftWallet struct {
	mu             *sync.RWMutex
	keystore       ks.Keystore
	unlockedKeyMap map[common.Address]*UnlockedKey // Currently unlocked keys (decrypted private keys)
}

type UnlockedKey struct {
	*ks.Key
	abort chan struct{}
}

func NewSoftWallet(keysDirPath string, kstype KeystoreType) (*SoftWallet, error) {
	var keystore ks.Keystore
	var err error
	if kstype == KeystoreTypeEncrypted {
		keystore, err = ks.NewKeystoreEncrypted(keysDirPath, ks.StandardScryptN, ks.StandardScryptP)
	} else {
		keystore, err = ks.NewKeystorePlain(keysDirPath)
	}
	if err != nil {
		return nil, err
	}

	wallet := &SoftWallet{
		mu:             &sync.RWMutex{},
		keystore:       keystore,
		unlockedKeyMap: make(map[common.Address]*UnlockedKey),
	}

	return wallet, nil
}

// ID returns the ID of the wallet
func (w *SoftWallet) ID() string {
	return "softwallet"
}

// Status returns the status of the wallet
func (w *SoftWallet) Status() (string, error) {
	return "", nil
}

// List returns the addresses of all the keys
func (w *SoftWallet) List() ([]common.Address, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	addresses, err := w.keystore.ListKeyAddresses()
	return addresses, err
}

// NewKey creates a new key
func (w *SoftWallet) NewKey(password string) (common.Address, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	privKey, _, err := crypto.GenerateKeyPair()
	if err != nil {
		return common.Address{}, err
	}

	key := ks.NewKey(privKey)
	address := key.Address

	w.keystore.StoreKey(key, password)

	// newly created key is considerred unlocked
	unlockedKey := &UnlockedKey{
		Key: key,
	}
	w.unlockedKeyMap[address] = unlockedKey

	return address, nil
}

// Unlock unlocks a key if the password is correct
func (w *SoftWallet) Unlock(address common.Address, password string, derivationPath types.DerivationPath) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	key, err := w.keystore.GetKey(address, password)
	if err != nil {
		return err
	}

	unlockedKey := &UnlockedKey{
		Key: key,
	}
	w.unlockedKeyMap[address] = unlockedKey

	return nil
}

// Lock locks an unlocked key
func (w *SoftWallet) Lock(address common.Address) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	unlockedKey, exists := w.unlockedKeyMap[address]
	if !exists {
		return fmt.Errorf("Cannot close address %v, not unlocked yet", address)
	}

	delete(w.unlockedKeyMap, address)

	if unlockedKey != nil {
		w.zeroKey(unlockedKey)
	}

	return nil
}

// IsUnlocked indicates whether a key is unlocked
func (w *SoftWallet) IsUnlocked(address common.Address) bool {
	if _, exists := w.unlockedKeyMap[address]; exists {
		return true
	}
	return false
}

// Delete deletes a key from disk permanently
func (w *SoftWallet) Delete(address common.Address, password string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	unlockedKey, exists := w.unlockedKeyMap[address]
	if exists {
		delete(w.unlockedKeyMap, address)
		if unlockedKey != nil {
			w.zeroKey(unlockedKey)
		}
	}

	err := w.keystore.DeleteKey(address, password)
	return err
}

// UpdatePassword updates the password for a key
func (w *SoftWallet) UpdatePassword(address common.Address, oldPassword, newPassword string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	key, err := w.keystore.GetKey(address, oldPassword)
	if err != nil {
		return err
	}

	err = w.keystore.StoreKey(key, newPassword)
	return err
}

// Derive is not supported for SoftWallet
func (w *SoftWallet) Derive(path types.DerivationPath, pin bool) (common.Address, error) {
	return common.Address{}, fmt.Errorf("Not supported for software wallet")
}

// GetPublicKey returns the public key of the address if the address has been unlocked
func (w *SoftWallet) GetPublicKey(address common.Address) (*crypto.PublicKey, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	unlockedKey, found := w.unlockedKeyMap[address]
	if !found {
		return nil, fmt.Errorf("Key not unlocked yet for address: %v", address)
	}

	pubKey := unlockedKey.PrivateKey.PublicKey()
	return pubKey, nil
}

// Sign signs the transaction bytes for an address if the address has been unlocked
func (w *SoftWallet) Sign(address common.Address, txrlp common.Bytes) (*crypto.Signature, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	unlockedKey, found := w.unlockedKeyMap[address]
	if !found {
		return nil, fmt.Errorf("Key not unlocked yet for address: %v", address)
	}

	signature, err := unlockedKey.Sign(txrlp)
	return signature, err
}

// zeroKey zeroes a private key in memory
func (w *SoftWallet) zeroKey(unlockedKey *UnlockedKey) {
	if unlockedKey == nil {
		return
	}

	privKey := unlockedKey.PrivateKey
	if privKey == nil || privKey.D() == nil {
		return
	}

	bits := privKey.D().Bits()
	for i := range bits {
		bits[i] = 0
	}
}
