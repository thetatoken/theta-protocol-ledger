package crypto

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"hash"
	"io"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/hexutil"
	"github.com/thetatoken/theta/rlp"
	"golang.org/x/crypto/sha3"
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

// D returns the D parameter of the ECDSA private key
func (sk *PrivateKey) D() *big.Int {
	return sk.privKey.D
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
	msgHash := keccak256(msg)
	sigBytes, err := sign(msgHash, sk.privKey)
	sig := &Signature{data: sigBytes}
	return sig, err
}

//
// PublicKey represents the public key
//
type PublicKey struct {
	pubKey *ecdsa.PublicKey
}

var _ rlp.Encoder = (*PublicKey)(nil)

// EncodeRLP implements RLP Encoder interface.
func (pk *PublicKey) EncodeRLP(w io.Writer) error {
	if pk == nil {
		return rlp.Encode(w, []byte{})
	}
	b := pk.ToBytes()
	return rlp.Encode(w, b)
}

var _ rlp.Decoder = (*PublicKey)(nil)

// DecodeRLP implements RLP Decoder interface.
func (pk *PublicKey) DecodeRLP(stream *rlp.Stream) error {
	var b []byte
	err := stream.Decode(&b)
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return nil
	}
	pubKey, err := unmarshalPubkey(b)
	if err != nil {
		return err
	}
	pk.pubKey = pubKey
	return nil
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

// VerifySignature verifies the signature with the public key (using ecrecover)
func (pk *PublicKey) VerifySignature(msg common.Bytes, sig *Signature) bool {
	if sig == nil {
		return false
	}

	msgHash := keccak256(msg)
	recoveredUncompressedPubKey, err := ecrecover(msgHash, sig.ToBytes())
	if err != nil {
		return false
	}

	uncompressedPubKey := pk.ToBytes()
	if bytes.Compare(recoveredUncompressedPubKey, uncompressedPubKey) != 0 {
		return false
	}

	return true
}

// // VerifySignature verifies the signature with the public key (using secp256k1.VerifySignature)
// func (pk *PublicKey) VerifySignature(msg common.Bytes, sig *Signature) bool {
// 	if sig == nil {
// 		return false
// 	}

// 	// https://github.com/ethereum/go-ethereum/blob/master/crypto/secp256k1/secp256.go#L52
// 	// signature should be 65 bytes long, where the 64th byte is the recovery id
// 	sigBytes := sig.ToBytes()
// 	if len(sigBytes) != 65 {
// 		return false
// 	}

// 	msgHash := keccak256(msg)
// 	isValid := verifySignature(pk.ToBytes(), msgHash, sigBytes[:64])
// 	return isValid
// }

//
// Signature represents the digital signature
//
type Signature struct {
	data common.Bytes
}

var _ rlp.Encoder = (*Signature)(nil)

// EncodeRLP implements RLP Encoder interface.
func (sig *Signature) EncodeRLP(w io.Writer) error {
	if sig == nil {
		return rlp.Encode(w, []byte{})
	}
	b := sig.ToBytes()
	return rlp.Encode(w, b)
}

var _ rlp.Decoder = (*Signature)(nil)

// DecodeRLP implements RLP Decoder interface.
func (sig *Signature) DecodeRLP(stream *rlp.Stream) error {
	var b []byte
	err := stream.Decode(&b)
	if err != nil {
		return err
	}
	sig.data = b
	return nil
}

// ToBytes returns the bytes representation of the signature
func (sig *Signature) ToBytes() common.Bytes {
	return sig.data
}

// MarshalJSON returns the JSON representation of the signature
func (sig *Signature) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(sig.data))
}

// UnmarshalJSON parses the JSON representation of the signature
func (sig *Signature) UnmarshalJSON(data []byte) error {
	raw := &hexutil.Bytes{}
	err := raw.UnmarshalJSON(data)
	if err != nil {
		return err
	}
	sig.data = ([]byte)(*raw)
	return nil
}

// IsEmpty indicates whether the signature is empty
func (sig *Signature) IsEmpty() bool {
	return len(sig.data) == 0
}

// RecoverSignerAddress recovers the address of the signer for the given message
func (sig *Signature) RecoverSignerAddress(msg common.Bytes) (common.Address, error) {
	msgHash := keccak256(msg)
	recoveredUncompressedPubKey, err := ecrecover(msgHash, sig.ToBytes())
	if err != nil {
		return common.Address{}, err
	}

	pk, err := PublicKeyFromBytes(recoveredUncompressedPubKey)
	if err != nil {
		return common.Address{}, err
	}

	address := pk.Address()
	return address, nil
}

// Verify verifies the signature with given raw message and address.
func (sig *Signature) Verify(msg common.Bytes, addr common.Address) bool {
	if sig == nil || sig.IsEmpty() {
		return false
	}
	recoveredAddress, err := sig.RecoverSignerAddress(msg)
	if err != nil {
		return false
	}
	if recoveredAddress != addr {
		return false
	}
	return true
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

// PrivateKeyFromBytesUnsafe blindly converts a binary blob to a private key. It
// should almost never be used unless you are sure the input is valid and want to
// avoid hitting errors due to bad origin encoding (0 prefixes cut off).
func PrivateKeyFromBytesUnsafe(skBytes common.Bytes) *PrivateKey {
	key := toECDSAUnsafe(skBytes)
	sk := &PrivateKey{privKey: key}
	return sk
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

// KeccakState wraps sha3.state. In addition to the usual hash methods, it also supports
// Read to get a variable amount of data from the hash state. Read is faster than Sum
// because it doesn't copy the internal state, but also modifies the internal state.
type KeccakState interface {
	hash.Hash
	Read([]byte) (int, error)
}

// NewKeccakState creates a new KeccakState
func NewKeccakState() KeccakState {
	return sha3.NewLegacyKeccak256().(KeccakState)
}
