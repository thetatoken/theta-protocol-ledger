package crypto

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/thetatoken/ukulele/common"
)

//
// ----------------------------- Hash APIs ----------------------------- //
//

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	return keccak256(data...)
}

// Keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func Keccak256Hash(data ...[]byte) (h common.Hash) {
	return keccak256Hash(data...)
}

//
// ----------------------- Digital Signature APIs ----------------------- //
//

// CryptoScheme is a enum for different crypto schemes
type CryptoScheme byte

const (
	// CryptoSchemeECDSA indicates the ECDSA scheme
	CryptoSchemeECDSA CryptoScheme = 1
)

//
// PrivateKey defines the interface of the private key
//
type PrivateKey interface {
	ToBytes() common.Bytes
	PublicKey() PublicKey
	SaveToFile(filepath string) error
	Sign(msg common.Bytes) (Signature, error)
}

//
// PublicKey defines the interface of the public key
//
type PublicKey interface {
	ToBytes() common.Bytes
	Address() common.Address
	IsEmpty() bool
	VerifySignature(msg common.Bytes, sig Signature) bool
}

//
// Signature defines the interface of the digital signature
//
type Signature interface {
	ToBytes() common.Bytes
	IsEmpty() bool
}

// GenerateKeyPair generates a random private/public key pair
func GenerateKeyPair(scheme CryptoScheme) (PrivateKey, PublicKey, error) {
	if scheme == CryptoSchemeECDSA {
		ske, err := generateKey()
		pke := &ske.PublicKey
		return &PrivateKeyECDSA{privKey: ske}, &PublicKeyECDSA{pubKey: pke}, err
	}
	return nil, nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// TODO: parse the CryptoScheme from the file instead of passing in as a parameter
// PrivateKeyFromFile loads the private key from the given file
func PrivateKeyFromFile(filepath string, scheme CryptoScheme) (PrivateKey, error) {
	if scheme == CryptoSchemeECDSA {
		key, err := loadECDSA(filepath)
		ske := &PrivateKeyECDSA{privKey: key}
		return ske, err
	}
	return nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// TODO: parse the CryptoScheme from the bytes instead of passing in as a parameter
// PrivateKeyFromBytes converts the given bytes to a private key
func PrivateKeyFromBytes(skBytes common.Bytes, scheme CryptoScheme) (PrivateKey, error) {
	if scheme == CryptoSchemeECDSA {
		key, err := toECDSA(skBytes)
		ske := &PrivateKeyECDSA{privKey: key}
		return ske, err
	}
	return nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// TODO: parse the CryptoScheme from the bytes instead of passing in as a parameter
// PublicKeyFromBytes converts the given bytes to a public key
func PublicKeyFromBytes(pkBytes common.Bytes, scheme CryptoScheme) (PublicKey, error) {
	if scheme == CryptoSchemeECDSA {
		key, err := unmarshalPubkey(pkBytes)
		pke := &PublicKeyECDSA{pubKey: key}
		return pke, err
	}
	return nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// TODO: parse the CryptoScheme from the bytes instead of passing in as a parameter
// SignatureFromBytes converts the given bytes to a signature
func SignatureFromBytes(sigBytes common.Bytes, scheme CryptoScheme) (Signature, error) {
	if scheme == CryptoSchemeECDSA {
		sige := &SignatureECDSA{data: sigBytes}
		return sige, nil
	}
	return nil, fmt.Errorf("Invalid crypto scheme: %v", scheme)
}

// ----------------------- ECDSA Implementation ----------------------- //

//
// PrivateKeyECDSA implements the PrivateKey interface
//
type PrivateKeyECDSA struct {
	privKey *ecdsa.PrivateKey
}

// ToBytes returns the bytes representation of the private key
func (ske *PrivateKeyECDSA) ToBytes() common.Bytes {
	skbytes := fromECDSA(ske.privKey)
	return skbytes
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

// Sign signs the given message with the private key
func (ske *PrivateKeyECDSA) Sign(msg common.Bytes) (Signature, error) {
	msgHash := keccak256Hash(msg)
	sigBytes, err := sign(msgHash[:], ske.privKey)
	sig := &SignatureECDSA{data: sigBytes}
	return sig, err
}

//
// PublicKeyECDSA implements the PublicKey interface
//
type PublicKeyECDSA struct {
	pubKey *ecdsa.PublicKey
}

// ToBytes returns the bytes representation of the public key
func (pke *PublicKeyECDSA) ToBytes() common.Bytes {
	pkbytes := fromECDSAPub(pke.pubKey)
	return pkbytes
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

// VerifySignature verifies the signature with the public key
func (pke *PublicKeyECDSA) VerifySignature(msg common.Bytes, sig Signature) bool {
	msgHash := keccak256Hash(msg)
	isValid := verifySignature(pke.ToBytes(), msgHash[:], sig.ToBytes())
	return isValid
}

//
// SignatureECDSA implements the Signature interface
//
type SignatureECDSA struct {
	data common.Bytes
}

// ToBytes returns the bytes representation of the signature
func (sige *SignatureECDSA) ToBytes() common.Bytes {
	return sige.data
}

// IsEmpty indicates whether the signature is empty
func (sige *SignatureECDSA) IsEmpty() bool {
	return len(sige.data) == 0
}
