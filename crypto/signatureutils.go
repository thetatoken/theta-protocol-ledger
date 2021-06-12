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
	"errors"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/math"
	"github.com/thetatoken/theta/crypto/secp256k1"
)

const SignatureLength = 64 + 1 // 64 bytes ECDSA signature + 1 byte recovery id

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

// ----------------------- ETH signature utils ----------------------- //

func EncodeSignature(R, S, Vb *big.Int) (*Signature, error) {
	if Vb.BitLen() > 8 {
		return nil, errors.New("invalid v, r, s values")
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, true) {
		return nil, errors.New("invalid v, r, s values")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sigBytes := make([]byte, SignatureLength)
	copy(sigBytes[32-len(r):32], r)
	copy(sigBytes[64-len(s):64], s)
	sigBytes[64] = V

	sig, err := SignatureFromBytes(sigBytes)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func DecodeSignature(sig *Signature) (r, s, v *big.Int) {
	sigBytes := sig.ToBytes()
	if len(sigBytes) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sigBytes), SignatureLength))
	}
	r = new(big.Int).SetBytes(sigBytes[:32])
	s = new(big.Int).SetBytes(sigBytes[32:64])
	v = new(big.Int).SetBytes([]byte{sigBytes[64] + 27})
	return r, s, v
}

func recoverPlain(txhash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := Ecrecover(txhash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], Keccak256(pub[1:])[12:])
	return addr, nil
}

func HomesteadSignerSender(txHash common.Hash, sig *Signature) (common.Address, error) {
	v, r, s := DecodeSignature(sig)
	return recoverPlain(txHash, r, s, v, true)
}

func ValidateEthSignature(sender common.Address, txHash common.Hash, sig *Signature) error {
	recoveredSender, err := HomesteadSignerSender(txHash, sig)
	if err != nil {
		return err
	}

	if recoveredSender != sender {
		return errors.New(fmt.Sprintf("Recovered sender mismatch, recovered sender: %v, sender: %v", recoveredSender.Hex(), sender.Hex()))
	}

	return nil
}
