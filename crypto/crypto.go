package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

// ----------------------- Crypto Interfaces ----------------------- //

// CrytoScheme is a enum for different crypto schemes
type CrytoScheme byte

const (
	// CrytoSchemeECDSA indicates the ECDSA scheme
	CrytoSchemeECDSA CrytoScheme = 1
)

//
// PublicKey defines the interface of the public key
//
type PublicKey interface {
	Address() common.Address
	IsEmpty() bool
	ToBytes() common.Bytes
}

//
// PrivateKey defines the interface of the private key
//
type PrivateKey interface {
	PublicKey() PublicKey
	SaveToFile(filepath string) error
}

//
// Signature defines the interface of the digital signature
//
type Signature interface {
}

// GenerateKeyPair generates a random private/public key pair
func GenerateKeyPair(scheme CrytoScheme) (PrivateKey, PublicKey, error) {
	if scheme == CrytoSchemeECDSA {
		ske, err := generateKey()
		pke := &ske.PublicKey
		return &PrivateKeyECDSA{privKey: ske}, &PublicKeyECDSA{pubKey: pke}, err
	}
	return nil, nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// LoadPrivateKeyFromFile loads the private key from the given file
func LoadPrivateKeyFromFile(filepath string, scheme CrytoScheme) (PrivateKey, error) {
	if scheme == CrytoSchemeECDSA {
		privKey, err := loadECDSA(filepath)
		ske := &PrivateKeyECDSA{privKey: privKey}
		return ske, err
	}
	return nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// ----------------------- ECDSA Implementation ----------------------- //

//
// PublicKeyECDSA implements the PublicKey interface
//
type PublicKeyECDSA struct {
	pubKey *ecdsa.PublicKey
}

// Address returns the address corresponding to the public key
func (pke *PublicKeyECDSA) Address() common.Address {
	pubBytes := fromECDSAPub(pke.pubKey)
	address := common.BytesToAddress(keccak256(pubBytes[1:])[12:])
	return address
}

// IsEmpty indicates whether the public key is empty
func (pke *PublicKeyECDSA) IsEmpty() bool {
	isEmpty := (pke.pubKey == nil || pke.pubKey.X == nil || pke.pubKey.Y == nil)
	return isEmpty
}

// ToBytes returns the bytes representation of the public key
func (pke *PublicKeyECDSA) ToBytes() common.Bytes {
	pkbytes := fromECDSAPub(pke.pubKey)
	return pkbytes
}

//
// PrivateKeyECDSA implements the PrivateKey interface
//
type PrivateKeyECDSA struct {
	privKey *ecdsa.PrivateKey
}

// PublicKey returns the public key corresponding to the private key
func (ske *PrivateKeyECDSA) PublicKey() PublicKey {
	pke := &ske.privKey.PublicKey
	return &PublicKeyECDSA{
		pubKey: pke,
	}
}

// SaveToFile saves the private key to the designated file
func (ske *PrivateKeyECDSA) SaveToFile(filepath string) error {
	err := saveECDSA(filepath, ske.privKey)
	return err
}

//
// SignatureECDSA implements the Signature interface
//
type SignatureECDSA struct {
	data []byte
}

// ----------------------- Crypto Utils for Other Modules ----------------------- //

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	return keccak256(data...)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	return keccak256Hash(data...)
}

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return s256()
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	return hexToECDSA(hexkey)
}
