// Adapted for Theta
// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

/*

This key store behaves as KeyStorePlain with the difference that
the private key is encrypted and on disk uses another JSON encoding.

The crypto is documented at https://github.com/ethereum/wiki/wiki/Web3-Secret-Storage-Definition

*/

package keystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pborman/uuid"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/math"
	"github.com/thetatoken/theta/crypto"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/scrypt"
)

const (
	version = 3

	keyHeaderKDF = "scrypt"

	// StandardScryptN is the N parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptN = 1 << 18

	// StandardScryptP is the P parameter of Scrypt encryption algorithm, using 256MB
	// memory and taking approximately 1s CPU time on a modern processor.
	StandardScryptP = 1

	// LightScryptN is the N parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptN = 1 << 12

	// LightScryptP is the P parameter of Scrypt encryption algorithm, using 4MB
	// memory and taking approximately 100ms CPU time on a modern processor.
	LightScryptP = 6

	scryptR     = 8
	scryptDKLen = 32
)

var (
	ErrDecrypt = fmt.Errorf("could not decrypt key with given password")
)

type KeystoreEncrypted struct {
	keysDirPath string
	scryptN     int
	scryptP     int
}

func NewKeystoreEncrypted(keysDirRoot string, scryptN, scryptP int) (KeystoreEncrypted, error) {
	keysDirPath := path.Join(keysDirRoot, "encrypted")
	err := os.MkdirAll(keysDirPath, 0700)
	if err != nil {
		return KeystoreEncrypted{}, err
	}
	os.Chmod(keysDirPath, 0700)

	fi, err := os.Lstat(keysDirPath)
	if err != nil {
		return KeystoreEncrypted{}, err
	}
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0700 {
		return KeystoreEncrypted{}, fmt.Errorf("%s must have permission set to 0700", keysDirPath)
	}

	ks := KeystoreEncrypted{
		keysDirPath: keysDirPath,
		scryptN:     scryptN,
		scryptP:     scryptP,
	}

	return ks, nil
}

func (ks KeystoreEncrypted) ListKeyAddresses() ([]common.Address, error) {
	filenames, err := filepath.Glob(path.Join(ks.keysDirPath, "*"))
	if err != nil {
		return []common.Address{}, err
	}

	addresses := []common.Address{}
	for _, filename := range filenames {
		addrStr := filepath.Base(filename)
		address := common.HexToAddress(addrStr)
		addresses = append(addresses, address)
	}

	return addresses, nil
}

func (ks KeystoreEncrypted) GetKey(address common.Address, auth string) (*Key, error) {
	var keyjson []byte
	var err error
	for af := allLowerCase; af <= allUpperCase; af++ { // try all formats
		filePath := ks.getFilePath(address, af)
		keyjson, err = ioutil.ReadFile(filePath)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, err
	}

	key, err := decryptKey(keyjson, auth)
	if err != nil {
		return nil, err
	}
	// Make sure we're really operating on the requested key (no swap attacks)
	if key.Address != address {
		return nil, fmt.Errorf("key content mismatch: have account %x, want %x", key.Address, address)
	}
	return key, nil
}

func (ks KeystoreEncrypted) StoreKey(key *Key, auth string) error {
	address := key.Address
	filePath := ks.getFilePath(address, mixedCase)
	keyjson, err := encryptKey(key, auth, ks.scryptN, ks.scryptP)
	if err != nil {
		return err
	}
	return writeKeyFile(filePath, keyjson)
}

func (ks KeystoreEncrypted) DeleteKey(address common.Address, auth string) error {
	_, err := ks.GetKey(address, auth)
	if err != nil {
		return err
	}

	for af := allLowerCase; af <= allUpperCase; af++ { // try all formats
		filePath := ks.getFilePath(address, af)
		deleteKeyFile(filePath)
	}

	return nil
}

func (ks KeystoreEncrypted) getFilePath(address common.Address, addrFormat AddressFormat) string {
	var filePath string
	addrStr := address.Hex()[2:]
	if addrFormat == allLowerCase {
		filePath = path.Join(ks.keysDirPath, strings.ToLower(addrStr))
	} else if addrFormat == allUpperCase {
		filePath = path.Join(ks.keysDirPath, strings.ToUpper(addrStr))
	} else {
		filePath = path.Join(ks.keysDirPath, addrStr)
	}
	return filePath
}

// encryptKey encrypts a key using the specified scrypt parameters into a json
// blob that can be decrypted later on.
func encryptKey(key *Key, auth string, scryptN, scryptP int) ([]byte, error) {
	authArray := []byte(auth)

	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	derivedKey, err := scrypt.Key(authArray, salt, scryptN, scryptR, scryptP, scryptDKLen)
	if err != nil {
		return nil, err
	}
	encryptKey := derivedKey[:16]
	keyBytes := math.PaddedBigBytes(key.PrivateKey.D(), 32)

	iv := make([]byte, aes.BlockSize) // 16
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic("reading from crypto/rand failed: " + err.Error())
	}
	cipherText, err := aesCTRXOR(encryptKey, keyBytes, iv)
	if err != nil {
		return nil, err
	}
	mac := crypto.Keccak256(derivedKey[16:32], cipherText)

	scryptParamsJSON := make(map[string]interface{}, 5)
	scryptParamsJSON["n"] = scryptN
	scryptParamsJSON["r"] = scryptR
	scryptParamsJSON["p"] = scryptP
	scryptParamsJSON["dklen"] = scryptDKLen
	scryptParamsJSON["salt"] = hex.EncodeToString(salt)

	cipherParamsJSON := cipherparamsJSON{
		IV: hex.EncodeToString(iv),
	}

	cryptoStruct := cryptoJSON{
		Cipher:       "aes-128-ctr",
		CipherText:   hex.EncodeToString(cipherText),
		CipherParams: cipherParamsJSON,
		KDF:          keyHeaderKDF,
		KDFParams:    scryptParamsJSON,
		MAC:          hex.EncodeToString(mac),
	}

	encryptedKeyJSON := encryptedKeyJSON{
		hex.EncodeToString(key.Address[:]),
		cryptoStruct,
		key.Id.String(),
		version,
	}
	return json.Marshal(encryptedKeyJSON)
}

// decryptKey decrypts a key from a json blob, returning the private key itself.
func decryptKey(keyjson []byte, auth string) (*Key, error) {
	encryptedKeyJs := new(encryptedKeyJSON)
	if err := json.Unmarshal(keyjson, encryptedKeyJs); err != nil {
		return nil, err
	}

	if encryptedKeyJs.Version != version {
		return nil, fmt.Errorf("Version %v not supported", encryptedKeyJs.Version)
	}

	if encryptedKeyJs.Crypto.Cipher != "aes-128-ctr" {
		return nil, fmt.Errorf("Cipher not supported: %v", encryptedKeyJs.Crypto.Cipher)
	}

	keyId := uuid.Parse(encryptedKeyJs.Id)

	mac, err := hex.DecodeString(encryptedKeyJs.Crypto.MAC)
	if err != nil {
		return nil, err
	}

	iv, err := hex.DecodeString(encryptedKeyJs.Crypto.CipherParams.IV)
	if err != nil {
		return nil, err
	}

	cipherText, err := hex.DecodeString(encryptedKeyJs.Crypto.CipherText)
	if err != nil {
		return nil, err
	}

	derivedKey, err := getKDFKey(encryptedKeyJs.Crypto, auth)
	if err != nil {
		return nil, err
	}

	calculatedMAC := crypto.Keccak256(derivedKey[16:32], cipherText)
	if !bytes.Equal(calculatedMAC, mac) {
		return nil, ErrDecrypt
	}

	keyBytes, err := aesCTRXOR(derivedKey[:16], cipherText, iv)
	if err != nil {
		return nil, err
	}

	// Use the "unsafe" convertor to support legacy private keys
	// whose lengths are less than 32 bytes
	privKey := crypto.PrivateKeyFromBytesUnsafe(keyBytes)

	key := &Key{
		Id:         keyId,
		Address:    privKey.PublicKey().Address(),
		PrivateKey: privKey,
	}

	return key, nil
}

func getKDFKey(cryptoJSON cryptoJSON, auth string) ([]byte, error) {
	authArray := []byte(auth)
	salt, err := hex.DecodeString(cryptoJSON.KDFParams["salt"].(string))
	if err != nil {
		return nil, err
	}
	dkLen := ensureInt(cryptoJSON.KDFParams["dklen"])

	if cryptoJSON.KDF == keyHeaderKDF {
		n := ensureInt(cryptoJSON.KDFParams["n"])
		r := ensureInt(cryptoJSON.KDFParams["r"])
		p := ensureInt(cryptoJSON.KDFParams["p"])
		return scrypt.Key(authArray, salt, n, r, p, dkLen)

	} else if cryptoJSON.KDF == "pbkdf2" {
		c := ensureInt(cryptoJSON.KDFParams["c"])
		prf := cryptoJSON.KDFParams["prf"].(string)
		if prf != "hmac-sha256" {
			return nil, fmt.Errorf("Unsupported PBKDF2 PRF: %s", prf)
		}
		key := pbkdf2.Key(authArray, salt, c, dkLen, sha256.New)
		return key, nil
	}

	return nil, fmt.Errorf("Unsupported KDF: %s", cryptoJSON.KDF)
}

func aesCTRXOR(key, inText, iv []byte) ([]byte, error) {
	// AES-128 is selected due to size of encryptKey.
	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	stream := cipher.NewCTR(aesBlock, iv)
	outText := make([]byte, len(inText))
	stream.XORKeyStream(outText, inText)
	return outText, err
}

// TODO: can we do without this when unmarshalling dynamic JSON?
// why do integers in KDF params end up as float64 and not int after
// unmarshal?
func ensureInt(x interface{}) int {
	res, ok := x.(int)
	if !ok {
		res = int(x.(float64))
	}
	return res
}

type encryptedKeyJSON struct {
	Address string     `json:"address"`
	Crypto  cryptoJSON `json:"crypto"`
	Id      string     `json:"id"`
	Version int        `json:"version"`
}

type cryptoJSON struct {
	Cipher       string                 `json:"cipher"`
	CipherText   string                 `json:"ciphertext"`
	CipherParams cipherparamsJSON       `json:"cipherparams"`
	KDF          string                 `json:"kdf"`
	KDFParams    map[string]interface{} `json:"kdfparams"`
	MAC          string                 `json:"mac"`
}

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

var (
	EncryptKey = encryptKey
	DecryptKey = decryptKey
)
