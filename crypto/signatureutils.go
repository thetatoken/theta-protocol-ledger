// Adapted for Theta
// Copyright 2017 The go-ethereum Authors
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

// +build !nacl,!js,!nocgo

package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	"github.com/thetatoken/theta/common/math"
	"github.com/thetatoken/theta/crypto/secp256k1"
)

// ecrecover returns the uncompressed public key that created the given signature.
func ecrecover(hash, sig []byte) ([]byte, error) {
	return secp256k1.RecoverPubkey(hash, sig)
}

// sigToPub returns the public key that created the given signature.
func sigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	s, err := ecrecover(hash, sig)
	if err != nil {
		return nil, err
	}

	x, y := elliptic.Unmarshal(s256(), s)
	return &ecdsa.PublicKey{Curve: s256(), X: x, Y: y}, nil
}

// sign calculates an ECDSA signature.
//
// This function is susceptible to chosen plaintext attacks that can leak
// information about the private key that is used for signing. Callers must
// be aware that the given hash cannot be chosen by an adversery. Common
// solution is to hash any input before calculating the signature.
//
// The produced signature is in the [R || S || V] format where V is 0 or 1.
func sign(hash []byte, prv *ecdsa.PrivateKey) (sig []byte, err error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash is required to be exactly 32 bytes (%d)", len(hash))
	}
	seckey := math.PaddedBigBytes(prv.D, prv.Params().BitSize/8)
	defer zeroBytes(seckey)
	return secp256k1.Sign(hash, seckey)
}

// verifySignature checks that the given public key created signature over hash.
// The public key should be in compressed (33 bytes) or uncompressed (65 bytes) format.
// The signature should have the 64 byte [R || S] format.
func verifySignature(pubkey, hash, signature []byte) bool {
	return secp256k1.VerifySignature(pubkey, hash, signature)
}

// decompressPubkey parses a public key in the 33-byte compressed format.
func decompressPubkey(pubkey []byte) (*ecdsa.PublicKey, error) {
	x, y := secp256k1.DecompressPubkey(pubkey)
	if x == nil {
		return nil, fmt.Errorf("invalid public key")
	}
	return &ecdsa.PublicKey{X: x, Y: y, Curve: s256()}, nil
}

// compressPubkey encodes a public key to the 33-byte compressed format.
func compressPubkey(pubkey *ecdsa.PublicKey) []byte {
	return secp256k1.CompressPubkey(pubkey.X, pubkey.Y)
}

// s256 returns an instance of the secp256k1 curve.
func s256() elliptic.Curve {
	return secp256k1.S256()
}

// ----------------------- Crypto Utils for Other Modules ----------------------- //

// S256 returns an instance of the secp256k1 curve.
func S256() elliptic.Curve {
	return s256()
}

var (
	Ecrecover = ecrecover
	Sign      = sign
)
