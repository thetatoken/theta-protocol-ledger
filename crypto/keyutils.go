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

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/big"
	"os"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/math"
	"github.com/thetatoken/theta/crypto/sha3"
	"github.com/thetatoken/theta/rlp"
)

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

var errInvalidPubkey = errors.New("invalid secp256k1 public key")

// keccak256 calculates and returns the Keccak256 hash of the input data.
func keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// keccak256Hash calculates and returns the Keccak256 hash of the input data,
// converting it to an internal Hash data structure.
func keccak256Hash(data ...[]byte) (h common.Hash) {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	d.Sum(h[:0])
	return h
}

// keccak512 calculates and returns the Keccak512 hash of the input data.
func keccak512(data ...[]byte) []byte {
	d := sha3.NewKeccak512()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

// createAddress creates an ethereum address given the bytes and the nonce
func createAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(keccak256(data)[12:])
}

// createAddress2 creates an ethereum address given the address bytes, initial
// contract code and a salt.
func createAddress2(b common.Address, salt common.Hash, code []byte) common.Address {
	return common.BytesToAddress(keccak256([]byte{0xff}, b.Bytes(), salt.Bytes(), code)[12:])
}

// toECDSA creates a private key with the given D value.
func toECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	return toECDSAInternal(d, true)
}

// ToECDSAUnsafe blindly converts a binary blob to a private key. It should almost
// never be used unless you are sure the input is valid and want to avoid hitting
// errors due to bad origin encoding (0 prefixes cut off).
func toECDSAUnsafe(d []byte) *ecdsa.PrivateKey {
	priv, _ := toECDSAInternal(d, false)
	return priv
}

// toECDSAInternal creates a private key with the given D value. The strict parameter
// controls whether the key's length should be enforced at the curve size or
// it can also accept legacy encodings (0 prefixes).
func toECDSAInternal(d []byte, strict bool) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = s256()
	if strict && 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}
	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}
	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	return priv, nil
}

// fromECDSA exports a private key into a binary dump.
func fromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return math.PaddedBigBytes(priv.D, priv.Params().BitSize/8)
}

// unmarshalPubkey converts bytes to a secp256k1 public key.
func unmarshalPubkey(pub []byte) (*ecdsa.PublicKey, error) {
	x, y := elliptic.Unmarshal(s256(), pub)
	if x == nil {
		return nil, errInvalidPubkey
	}
	return &ecdsa.PublicKey{Curve: s256(), X: x, Y: y}, nil
}

func fromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(s256(), pub.X, pub.Y)
}

// hexToECDSA parses a secp256k1 private key.
func hexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	b, err := hex.DecodeString(hexkey)
	if err != nil {
		return nil, errors.New("invalid hex string")
	}
	return toECDSA(b)
}

// loadECDSA loads a secp256k1 private key from the given file.
func loadECDSA(file string) (*ecdsa.PrivateKey, error) {
	buf := make([]byte, 64)
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	if _, err := io.ReadFull(fd, buf); err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}
	return toECDSA(key)
}

// saveECDSA saves a secp256k1 private key to the given file with
// restrictive permissions. The key data is saved hex-encoded.
func saveECDSA(file string, key *ecdsa.PrivateKey) error {
	k := hex.EncodeToString(fromECDSA(key))
	return ioutil.WriteFile(file, []byte(k), 0600)
}

func generateKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(s256(), rand.Reader)
}

// validateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func validateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256k1halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1N) < 0 && s.Cmp(secp256k1N) < 0 && (v == 0 || v == 1)
}

func pubkeyToAddress(p ecdsa.PublicKey) common.Address {
	pubBytes := fromECDSAPub(&p)
	return common.BytesToAddress(keccak256(pubBytes[1:])[12:])
}

func zeroBytes(bytes []byte) {
	for i := range bytes {
		bytes[i] = 0
	}
}

// ----------------------- Crypto Utils for Other Modules ----------------------- //

// PrivKeyToECDSA convert private key to ecdsa.
func PrivKeyToECDSA(key *PrivateKey) *ecdsa.PrivateKey {
	return key.privKey
}

// PubKeyToECDSA convert public key to ecdsa.
func PubKeyToECDSA(key *PublicKey) *ecdsa.PublicKey {
	return key.pubKey
}

// ECDSAToPubKey converts given ecdsa public key to pubkey.
func ECDSAToPubKey(p *ecdsa.PublicKey) *PublicKey {
	return &PublicKey{p}
}

// ECDSAToPrivKey converts given ecdsa public key to pubkey.
func ECDSAToPrivKey(p *ecdsa.PrivateKey) *PrivateKey {
	return &PrivateKey{p}
}

// HexToECDSA parses a secp256k1 private key.
func HexToECDSA(hexkey string) (*ecdsa.PrivateKey, error) {
	return hexToECDSA(hexkey)
}

var (
	UnmarshalPubkey         = unmarshalPubkey
	FromECDSAPub            = fromECDSAPub
	FromECDSA               = fromECDSA
	ValidateSignatureValues = validateSignatureValues
	CreateAddress           = createAddress
	CreateAddress2          = createAddress2
)
