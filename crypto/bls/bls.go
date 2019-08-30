// Adapted for Theta from https://github.com/prysmaticlabs/prysm/.
//
// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Ethereum 2.0.
package bls

import (
	"encoding/binary"
	"fmt"
	"io"

	phorebls "github.com/phoreproject/bls"
	g1 "github.com/phoreproject/bls/g1pubs"
	"github.com/pkg/errors"
)

// PubkeyZero represents the pubkey from the point at infinity.
func PubkeyZero() *PublicKey {
	return &PublicKey{val: g1.NewPublicKeyFromG1(phorebls.G1AffineZero.Copy())}
}

// SignatureZero represents the signature from the point at infinity.
func SignatureZero() *Signature {
	return &Signature{val: g1.NewSignatureFromG2(phorebls.G2AffineZero.Copy())}
}

// Signature used in the BLS signature scheme.
type Signature struct {
	val *g1.Signature
}

// SecretKey used in the BLS signature scheme.
type SecretKey struct {
	val *g1.SecretKey
}

// PublicKey used in the BLS signature scheme.
type PublicKey struct {
	val *g1.PublicKey
}

// RandKey creates a new private key using a random method provided as an io.Reader.
func RandKey(r io.Reader) (*SecretKey, error) {
	k, err := g1.RandKey(r)
	if err != nil {
		return nil, errors.Wrap(err, "could not initialize secret key")
	}
	return &SecretKey{val: k}, nil
}

// SecretKeyFromBytes creates a BLS private key from a byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	if len(priv) != 32 {
		return nil, fmt.Errorf("expected byte slice of length 32, received: %d", len(priv))
	}
	k := ToBytes32(priv)
	val := g1.DeserializeSecretKey(k)
	if val.GetFRElement() == nil {
		return nil, errors.New("invalid private key")
	}
	return &SecretKey{val}, nil
}

// PublicKeyFromBytes creates a BLS public key from a byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	b := ToBytes48(pub)
	k, err := g1.DeserializePublicKey(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into public key")
	}
	return &PublicKey{val: k}, nil
}

// SignatureFromBytes creates a BLS signature from a byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	b := ToBytes96(sig)
	s, err := g1.DeserializeSignature(b)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal bytes into signature")
	}
	return &Signature{val: s}, nil
}

// PublicKey obtains the public key corresponding to the BLS secret key.
func (s *SecretKey) PublicKey() *PublicKey {
	return &PublicKey{val: g1.PrivToPub(s.val)}
}

// Sign a message using a secret key - in a beacon/validator client,
func (s *SecretKey) Sign(msg []byte, domain uint64) *Signature {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, domain)
	sig := g1.SignWithDomain(ToBytes32(msg), s.val, ToBytes8(b))
	return &Signature{val: sig}
}

// Marshal a secret key into a byte slice.
func (s *SecretKey) Marshal() []byte {
	k := s.val.Serialize()
	return k[:]
}

// Marshal a public key into a byte slice.
func (p *PublicKey) Marshal() []byte {
	k := p.val.Serialize()
	return k[:]
}

// Aggregate two public keys.
func (p *PublicKey) Aggregate(p2 *PublicKey) {
	p1 := p.val
	p1.Aggregate(p2.val)
}

// Aggregate two signatures.
func (s *Signature) Aggregate(s2 *Signature) {
	s1 := s.val
	s1.Aggregate(s2.val)
}

// Verify a bls signature given a public key, a message, and a domain.
func (s *Signature) Verify(msg []byte, pub *PublicKey, domain uint64) bool {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, domain)
	return g1.VerifyWithDomain(ToBytes32(msg), pub.val, s.val, ToBytes8(b))
}

// VerifyAggregate verifies each public key against a message.
// This is vulnerable to rogue public-key attack. Each user must
// provide a proof-of-knowledge of the public key.
func (s *Signature) VerifyAggregate(pubKeys []*PublicKey, msg []byte, domain uint64) bool {
	if len(pubKeys) == 0 {
		return false // Otherwise panic in VerifyAggregateCommonWithDomain.
	}
	var keys []*g1.PublicKey
	for _, v := range pubKeys {
		keys = append(keys, v.val)
	}
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, domain)
	return s.val.VerifyAggregateCommonWithDomain(keys, ToBytes32(msg), ToBytes8(b))
}

// Marshal a signature into a byte slice.
func (s *Signature) Marshal() []byte {
	k := s.val.Serialize()
	return k[:]
}

// AggregateSignatures converts a list of signatures into a single, aggregated sig.
func AggregateSignatures(sigs []*Signature) *Signature {
	var ss []*g1.Signature
	for _, v := range sigs {
		if v == nil {
			continue
		}
		ss = append(ss, v.val)
	}
	return &Signature{val: g1.AggregateSignatures(ss)}
}

//
// -------------- utils -----------------
//

// ToBytes8 is a convenience method for converting a byte slice to a fix
// sized 8 byte array. This method will truncate the input if it is larger
// than 8 bytes.
func ToBytes8(x []byte) [8]byte {
	var y [8]byte
	copy(y[:], x)
	return y
}

// ToBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func ToBytes32(x []byte) [32]byte {
	var y [32]byte
	copy(y[:], x)
	return y
}

// ToBytes48 is a convenience method for converting a byte slice to a fix
// sized 48 byte array. This method will truncate the input if it is larger
// than 48 bytes.
func ToBytes48(x []byte) [48]byte {
	var y [48]byte
	copy(y[:], x)
	return y
}

// ToBytes96 is a convenience method for converting a byte slice to a fix
// sized 96 byte array. This method will truncate the input if it is larger
// than 96 bytes.
func ToBytes96(x []byte) [96]byte {
	var y [96]byte
	copy(y[:], x)
	return y
}
