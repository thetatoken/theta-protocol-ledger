// Package bls implements a go-wrapper around a library implementing the
// the BLS12-381 curve and signature scheme. This package exposes a public API for
// verifying and aggregating BLS signatures used by Theta.
//
// Some of the code are adapted from:
// 	https://github.com/prysmaticlabs/prysm/
//  https://github.com/phoreproject/bls/
//

package bls

import (
	"encoding/binary"
	"fmt"
	"io"

	phorebls "github.com/phoreproject/bls"
	"github.com/pkg/errors"
)

const (
	DomainGuardian uint64 = iota
)

// Signature is a message signature.
type Signature struct {
	s *phorebls.G2Projective
}

// Marshal serializes a signature in compressed form.
func (s *Signature) Marshal() []byte {
	ret := phorebls.CompressG2(s.s.ToAffine())
	return ret[:]
}

// SignatureFromBytes creates a BLS signature from a byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	b := toBytes96(sig)
	a, err := phorebls.DecompressG2(b)
	if err != nil {
		return nil, err
	}

	return &Signature{s: a.ToProjective()}, nil
}

func (s *Signature) String() string {
	return s.s.String()
}

// Aggregate adds one signature to another
func (s Signature) Aggregate(other *Signature) {
	newS := s.s.Add(other.s)
	s.s = newS
}

// Copy returns a copy of the signature.
func (s *Signature) Copy() *Signature {
	return &Signature{s.s.Copy()}
}

// Verify verifies a signature against a message and a public key.
func (sig *Signature) Verify(m []byte, p PublicKey) bool {
	h := phorebls.HashG2(m)
	lhs := phorebls.Pairing(phorebls.G1ProjectiveOne, sig.s)
	rhs := phorebls.Pairing(p.p, h.ToProjective())
	return lhs.Equals(rhs)
}

// VerifyWithDomain verifies a signature against a message and a public key and a domain
func (sig *Signature) VerifyWithDomain(m []byte, p *PublicKey, domain uint64) bool {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, domain)
	h := phorebls.HashG2WithDomain(toBytes32(m), toBytes8(b))
	lhs := phorebls.Pairing(phorebls.G1ProjectiveOne, sig.s)
	rhs := phorebls.Pairing(p.p, h.ToAffine().ToProjective())
	return lhs.Equals(rhs)
}

// PublicKey is a public key.
type PublicKey struct {
	p *phorebls.G1Projective
}

func (p PublicKey) String() string {
	return p.p.String()
}

// Marshal serializes a public key to bytes.
func (p PublicKey) Marshal() []byte {
	ret := phorebls.CompressG1(p.p.ToAffine())
	return ret[:]
}

// PublicKeyFromBytes creates a BLS public key from a byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	b := toBytes48(pub)
	a, err := phorebls.DecompressG1(b)
	if err != nil {
		return nil, err
	}

	return &PublicKey{p: a.ToProjective()}, nil
}

// Equals checks if two public keys are equal
func (p PublicKey) Equals(other PublicKey) bool {
	return p.p.Equal(other.p)
}

// Aggregate adds two public keys together.
func (p *PublicKey) Aggregate(other *PublicKey) {
	newP := p.p.Add(other.p)
	p.p = newP
}

// Copy copies the public key and returns it.
func (p *PublicKey) Copy() *PublicKey {
	return &PublicKey{p: p.p.Copy()}
}

// SecretKey represents a BLS private key.
type SecretKey struct {
	f *phorebls.FR
}

// GetFRElement gets the underlying FR element.
func (s SecretKey) GetFRElement() *phorebls.FR {
	return s.f
}

func (s SecretKey) String() string {
	return s.f.String()
}

// Marshal serializes a secret key to bytes.
func (s SecretKey) Marshal() []byte {
	ret := s.f.Bytes()
	return ret[:]
}

// SecretKeyFromBytes creates a BLS private key from a byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	if len(priv) != 32 {
		return nil, fmt.Errorf("expected byte slice of length 32, received: %d", len(priv))
	}
	k := toBytes32(priv)
	val := &SecretKey{phorebls.FRReprToFR(phorebls.FRReprFromBytes(k))}
	if val.GetFRElement() == nil {
		return nil, errors.New("invalid private key")
	}
	return val, nil
}

// Sign signs a message with a secret key.
func (s SecretKey) Sign(message []byte) *Signature {
	h := phorebls.HashG2(message).MulFR(s.f.ToRepr())
	return &Signature{s: h}
}

// SignWithDomain signs a message with a secret key and its domain.
func (s SecretKey) SignWithDomain(message []byte, domain uint64) *Signature {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, domain)
	h := phorebls.HashG2WithDomain(toBytes32(message), toBytes8(b)).MulFR(s.f.ToRepr())
	return &Signature{s: h}
}

// PublicKey converts the private key into a public key.
func (s SecretKey) PublicKey() *PublicKey {
	return &PublicKey{p: phorebls.G1AffineOne.MulFR(s.f.ToRepr())}
}

// RandKey generates a random secret key.
func RandKey(r io.Reader) (*SecretKey, error) {
	k, err := phorebls.RandFR(r)
	if err != nil {
		return nil, err
	}
	s := &SecretKey{f: k}
	return s, nil
}

// AggregateSignatures adds up all of the signatures.
func AggregateSignatures(s []*Signature) *Signature {
	newSig := &Signature{s: phorebls.G2ProjectiveZero.Copy()}
	for _, sig := range s {
		newSig.Aggregate(sig)
	}
	return newSig
}

// AggregatePublicKeys adds public keys together.
func AggregatePublicKeys(p []*PublicKey) *PublicKey {
	newPub := &PublicKey{p: phorebls.G1ProjectiveZero.Copy()}
	for _, pub := range p {
		newPub.Aggregate(pub)
	}
	return newPub
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() *Signature {
	return &Signature{s: phorebls.G2ProjectiveZero.Copy()}
}

// NewAggregatePubkey creates a blank public key.
func NewAggregatePubkey() *PublicKey {
	return &PublicKey{p: phorebls.G1ProjectiveZero.Copy()}
}

//
// -------------- utils -----------------
//

// toBytes8 is a convenience method for converting a byte slice to a fix
// sized 8 byte array. This method will truncate the input if it is larger
// than 8 bytes.
func toBytes8(x []byte) [8]byte {
	var y [8]byte
	copy(y[:], x)
	return y
}

// toBytes32 is a convenience method for converting a byte slice to a fix
// sized 32 byte array. This method will truncate the input if it is larger
// than 32 bytes.
func toBytes32(x []byte) [32]byte {
	var y [32]byte
	copy(y[:], x)
	return y
}

// toBytes48 is a convenience method for converting a byte slice to a fix
// sized 48 byte array. This method will truncate the input if it is larger
// than 48 bytes.
func toBytes48(x []byte) [48]byte {
	var y [48]byte
	copy(y[:], x)
	return y
}

// toBytes96 is a convenience method for converting a byte slice to a fix
// sized 96 byte array. This method will truncate the input if it is larger
// than 96 bytes.
func toBytes96(x []byte) [96]byte {
	var y [96]byte
	copy(y[:], x)
	return y
}
