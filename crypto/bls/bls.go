package bls

import (
	"io"
	"sync"

	bh "github.com/herumi/bls-eth-go-binary/bls"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/rlp"
)

func init() {
	bh.Init(bh.BLS12_381)
}

// ------------- Signature --------------

// Signature is a message signature.
type Signature struct {
	s *bh.Sign
}

// ToBytes serializes a signature in compressed form.
func (s *Signature) ToBytes() common.Bytes {
	return s.s.Serialize()
}

// SignatureFromBytes creates a BLS signature from a byte slice.
func SignatureFromBytes(sig []byte) (*Signature, error) {
	s := &bh.Sign{}
	err := s.Deserialize(sig)
	if err != nil {
		return nil, err
	}
	return &Signature{s: s}, nil
}

// IsEmpty checks if signature is empty.
func (s *Signature) IsEmpty() bool {
	return s == nil || s.s == nil
}

func (s *Signature) String() string {
	return s.s.GetHexString()
}

// Aggregate adds one signature to another
func (s *Signature) Aggregate(other *Signature) {
	s.s.Add(other.s)
}

// Copy returns a copy of the signature.
func (s *Signature) Copy() *Signature {
	newS := &bh.Sign{}
	newS.Deserialize(s.s.Serialize())
	return &Signature{s: newS}
}

// Equals checks if two signatures are equal
func (s *Signature) Equals(other *Signature) bool {
	return s.s.IsEqual(other.s)
}

var _ rlp.Encoder = (*Signature)(nil)

// EncodeRLP implements RLP Encoder interface.
func (s *Signature) EncodeRLP(w io.Writer) error {
	if s == nil {
		return rlp.Encode(w, []byte{})
	}
	b := s.ToBytes()
	return rlp.Encode(w, b)
}

var _ rlp.Decoder = (*Signature)(nil)

// DecodeRLP implements RLP Decoder interface.
func (s *Signature) DecodeRLP(stream *rlp.Stream) error {
	raw, err := stream.Bytes()
	if err != nil {
		return err
	}
	if raw == nil || len(raw) == 0 {
		s.s = nil
		return nil
	}
	tmp, err := SignatureFromBytes(raw)
	if err != nil {
		return err
	}
	s.s = tmp.s
	return nil
}

// Verify verifies a signature against a message and a public key.
func (s *Signature) Verify(m []byte, p *PublicKey) bool {
	return s.s.Verify(p.p, string(m))
}

// PopVerify verifies a proof of possesion of a public key.
func (s *Signature) PopVerify(p *PublicKey) bool {
	return s.s.VerifyPop(p.p)
}

// ------------- Public key --------------

// PublicKey is a public key.
type PublicKey struct {
	p *bh.PublicKey
}

func (p *PublicKey) String() string {
	return p.p.GetHexString()
}

var _ rlp.Encoder = (*PublicKey)(nil)

// EncodeRLP implements RLP Encoder interface.
func (p *PublicKey) EncodeRLP(w io.Writer) error {
	if p == nil {
		return rlp.Encode(w, []byte{})
	}
	b := p.ToBytes()
	return rlp.Encode(w, b)
}

var _ rlp.Decoder = (*PublicKey)(nil)

// DecodeRLP implements RLP Decoder interface.
func (p *PublicKey) DecodeRLP(stream *rlp.Stream) error {
	raw, err := stream.Bytes()
	if err != nil {
		return err
	}
	if raw == nil || len(raw) == 0 {
		p.p = nil
		return nil
	}
	tmp, err := PublicKeyFromBytes(raw)
	if err != nil {
		return err
	}
	p.p = tmp.p
	return nil
}

// ToBytes serializes a public key to bytes.
func (p *PublicKey) ToBytes() common.Bytes {
	return p.p.Serialize()
}

// PublicKeyFromBytes creates a BLS public key from a byte slice.
func PublicKeyFromBytes(pub []byte) (*PublicKey, error) {
	p := &bh.PublicKey{}
	err := p.Deserialize(pub)
	if err != nil {
		return nil, err
	}

	return &PublicKey{p: p}, nil
}

// IsEmpty checks if pubkey is empty.
func (p *PublicKey) IsEmpty() bool {
	return p == nil || p.p == nil
}

// Equals checks if two public keys are equal
func (p *PublicKey) Equals(other *PublicKey) bool {
	return p.p.IsEqual(other.p)
}

// Aggregate adds two public keys together.
func (p *PublicKey) Aggregate(other *PublicKey) {
	p.p.Add(other.p)
}

// Copy copies the public key and returns it.
func (p *PublicKey) Copy() *PublicKey {
	newP := &bh.PublicKey{}
	newP.Deserialize(p.p.Serialize())
	return &PublicKey{p: newP}
}

// ------------- Secret key --------------

// SecretKey represents a BLS private key.
type SecretKey struct {
	f *bh.SecretKey
}

func (s *SecretKey) String() string {
	return s.f.GetHexString()
}

// Equals checks if two secret keys are equal
func (s *SecretKey) Equals(other *SecretKey) bool {
	return s.f.IsEqual(other.f)
}

// ToBytes serializes a secret key to bytes.
func (s *SecretKey) ToBytes() common.Bytes {
	return s.f.Serialize()
}

// SecretKeyFromBytes creates a BLS private key from a byte slice.
func SecretKeyFromBytes(priv []byte) (*SecretKey, error) {
	sk := &bh.SecretKey{}
	err := sk.Deserialize(priv)
	if err != nil {
		return nil, err
	}
	return &SecretKey{f: sk}, nil
}

// Sign signs a message with a secret key.
func (s *SecretKey) Sign(message []byte) *Signature {
	sig := s.f.Sign(string(message))
	return &Signature{s: sig}
}

// PublicKey converts the private key into a public key.
func (s *SecretKey) PublicKey() *PublicKey {
	return &PublicKey{p: s.f.GetPublicKey()}
}

// PopProve generates a proof of poccession of the secrect key.
func (s *SecretKey) PopProve() *Signature {
	sig := s.f.GetPop()
	return &Signature{s: sig}
}

// ------------- Static functions ----------------

var genkeyLock = sync.Mutex{}

func GenKey(seed io.Reader) (*SecretKey, error) {
	genkeyLock.Lock()
	defer genkeyLock.Unlock()

	bh.SetRandFunc(seed)
	defer bh.SetRandFunc(nil)

	k := &bh.SecretKey{}
	k.SetByCSPRNG()
	return &SecretKey{f: k}, nil
}

// RandKey generates a random secret key.
func RandKey() (*SecretKey, error) {
	// genkeyLock.Lock()
	// defer genkeyLock.Unlock()

	// k := &bh.SecretKey{}
	// k.SetByCSPRNG()
	// return &SecretKey{f: k}, nil
	return GenKey(nil)
}

// AggregateSignatures adds up all of the signatures.
func AggregateSignatures(s []*Signature) *Signature {
	newSig := NewAggregateSignature()
	for _, sig := range s {
		newSig.Aggregate(sig)
	}
	return newSig
}

func sigExp(s *Signature, exp uint32) *Signature {
	ret := NewAggregateSignature()
	c := s.Copy()
	for exp > 0 {
		if exp&1 == 1 {
			ret.Aggregate(c)
		}
		exp = exp >> 1
		c.Aggregate(c)
	}
	return ret
}

func pubkeyExp(p *PublicKey, exp uint32) *PublicKey {
	ret := NewAggregatePubkey()
	c := p.Copy()
	for exp > 0 {
		if exp&1 == 1 {
			ret.Aggregate(c)
		}
		exp = exp >> 1
		c.Aggregate(c)
	}
	return ret
}

// AggregateSignaturesVec aggregates signatures based on given vector.
func AggregateSignaturesVec(s []*Signature, vec []uint32) *Signature {
	if len(s) != len(vec) {
		panic("len(sigs) must be equal to len(vec)")
	}
	newSig := NewAggregateSignature()
	for i, sig := range s {
		if vec[i] == 0 {
			continue // performance improvement
		}
		newSig.Aggregate(sigExp(sig, vec[i]))
	}
	return newSig
}

// AggregatePublicKeys adds public keys together.
func AggregatePublicKeys(p []*PublicKey) *PublicKey {
	newPub := NewAggregatePubkey()
	for _, pub := range p {
		newPub.Aggregate(pub)
	}
	return newPub
}

// AggregatePublicKeysVec aggregates public keys based on given vector.
func AggregatePublicKeysVec(p []*PublicKey, vec []uint32) *PublicKey {
	if len(p) != len(vec) {
		panic("len(pubkeys) must be equal to len(vec)")
	}
	newPub := NewAggregatePubkey()
	for i, pub := range p {
		if vec[i] == 0 {
			continue // performance improvement
		}
		newPub.Aggregate(pubkeyExp(pub, vec[i]))
	}
	return newPub
}

// NewAggregateSignature creates a blank aggregate signature.
func NewAggregateSignature() *Signature {
	return &Signature{s: &bh.Sign{}}
}

// NewAggregatePubkey creates a blank public key.
func NewAggregatePubkey() *PublicKey {
	return &PublicKey{p: &bh.PublicKey{}}
}
