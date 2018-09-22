package crypto

import (
	"crypto/ecdsa"

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

//
// PrivateKey represents the private key
//
type PrivateKey struct {
	privKey *ecdsa.PrivateKey
}

// ToBytes returns the bytes representation of the private key
func (sk *PrivateKey) ToBytes() common.Bytes {
	skbytes := fromECDSA(sk.privKey)
	return skbytes
}

// PublicKey returns the public key corresponding to the private key
func (sk *PrivateKey) PublicKey() *PublicKey {
	pke := &sk.privKey.PublicKey
	return &PublicKey{
		pubKey: pke,
	}
}

// SaveToFile saves the private key to the designated file
func (sk *PrivateKey) SaveToFile(filepath string) error {
	err := saveECDSA(filepath, sk.privKey)
	return err
}

// Sign signs the given message with the private key
func (sk *PrivateKey) Sign(msg common.Bytes) (*Signature, error) {
	msgHash := keccak256Hash(msg)
	sigBytes, err := sign(msgHash[:], sk.privKey)
	sig := &Signature{data: sigBytes}
	return sig, err
}

//
// PublicKey represents the public key
//
type PublicKey struct {
	pubKey *ecdsa.PublicKey
}

// ToBytes returns the bytes representation of the public key
func (pk *PublicKey) ToBytes() common.Bytes {
	pkbytes := fromECDSAPub(pk.pubKey)
	return pkbytes
}

// Address returns the address corresponding to the public key
func (pk *PublicKey) Address() common.Address {
	pubBytes := fromECDSAPub(pk.pubKey)
	address := common.BytesToAddress(keccak256(pubBytes[1:])[12:])
	return address
}

// IsEmpty indicates whether the public key is empty
func (pk *PublicKey) IsEmpty() bool {
	isEmpty := (pk.pubKey == nil || pk.pubKey.X == nil || pk.pubKey.Y == nil)
	return isEmpty
}

// VerifySignature verifies the signature with the public key
func (pk *PublicKey) VerifySignature(msg common.Bytes, sig *Signature) bool {
	msgHash := keccak256Hash(msg)
	isValid := verifySignature(pk.ToBytes(), msgHash[:], sig.ToBytes())
	return isValid
}

//
// Signature represents the digital signature
//
type Signature struct {
	data common.Bytes
}

// ToBytes returns the bytes representation of the signature
func (sig *Signature) ToBytes() common.Bytes {
	return sig.data
}

// IsEmpty indicates whether the signature is empty
func (sig *Signature) IsEmpty() bool {
	return len(sig.data) == 0
}

// GenerateKeyPair generates a random private/public key pair
func GenerateKeyPair() (*PrivateKey, *PublicKey, error) {
	ske, err := generateKey()
	pke := &ske.PublicKey
	return &PrivateKey{privKey: ske}, &PublicKey{pubKey: pke}, err
}

// PrivateKeyFromFile loads the private key from the given file
func PrivateKeyFromFile(filepath string) (*PrivateKey, error) {
	key, err := loadECDSA(filepath)
	sk := &PrivateKey{privKey: key}
	return sk, err
}

// PrivateKeyFromBytes converts the given bytes to a private key
func PrivateKeyFromBytes(skBytes common.Bytes) (*PrivateKey, error) {
	key, err := toECDSA(skBytes)
	sk := &PrivateKey{privKey: key}
	return sk, err

}

// PublicKeyFromBytes converts the given bytes to a public key
func PublicKeyFromBytes(pkBytes common.Bytes) (*PublicKey, error) {
	key, err := unmarshalPubkey(pkBytes)
	pk := &PublicKey{pubKey: key}
	return pk, err
}

// SignatureFromBytes converts the given bytes to a signature
func SignatureFromBytes(sigBytes common.Bytes) (*Signature, error) {
	sig := &Signature{data: sigBytes}
	return sig, nil
}
