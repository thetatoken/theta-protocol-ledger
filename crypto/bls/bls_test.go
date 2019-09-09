package bls

import (
	"bytes"
	"crypto/rand"
	mrand "math/rand"
	"testing"

	"github.com/prysmaticlabs/prysm/shared/bytesutil"
)

func TestMarshalUnmarshal(t *testing.T) {
	b := []byte("hi")
	b32 := bytesutil.ToBytes32(b)
	pk, err := SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	pk2, err := SecretKeyFromBytes(b32[:])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pk.Marshal(), pk2.Marshal()) {
		t.Errorf("Keys not equal, received %#x == %#x", pk.Marshal(), pk2.Marshal())
	}
}

func TestSignVerify(t *testing.T) {
	priv, _ := RandKey(rand.Reader)
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	if !sig.Verify(msg, pub) {
		t.Error("Signature did not verify")
	}
}

func TestPop(t *testing.T) {
	priv, _ := RandKey(rand.Reader)
	pop := priv.PopProve()
	if !pop.PopVerify(priv.PublicKey()) {
		t.Error("PopVerify failed")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*PublicKey, 100, 100)
	sigs := make([]*Signature, 100, 100)
	vec := make([]uint32, 100, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv, _ := RandKey(rand.Reader)
		pub := priv.PublicKey()
		sig := priv.Sign(msg)
		pubkeys[i] = pub
		sigs[i] = sig
	}

	// Single signature.
	for i := 0; i < 100; i++ {
		if i == 0 {
			vec[i] = 1
		} else {
			vec[i] = 0
		}
	}
	aggSig := AggregateSignaturesVec(sigs, vec)
	aggPub := AggregatePublicKeysVec(pubkeys, vec)
	if !aggSig.s.Equals(sigs[0].s) {
		t.Fatal("sig should equal")
	}
	if !aggSig.Verify(msg, aggPub) {
		t.Error("Signature did not verify")
	}

	// Random vector
	for i := 0; i < 100; i++ {
		vec[i] = uint32(mrand.Uint64())
	}
	aggSig = AggregateSignaturesVec(sigs, vec)
	aggPub = AggregatePublicKeysVec(pubkeys, vec)
	if !aggSig.Verify(msg, aggPub) {
		t.Error("Signature did not verify")
	}

	vec[2] = vec[2] + 1
	aggPub = AggregatePublicKeysVec(pubkeys, vec)
	if aggSig.Verify(msg, aggPub) {
		t.Error("Signature should not verify")
	}

	vec[2] = 0
	aggPub = AggregatePublicKeysVec(pubkeys, vec)
	if aggSig.Verify(msg, aggPub) {
		t.Error("Signature should not verify")
	}

	vec[2] = 0
	aggSig = AggregateSignaturesVec(sigs, vec)
	vec[2] = 3
	aggPub = AggregatePublicKeysVec(pubkeys, vec)
	if aggSig.Verify(msg, aggPub) {
		t.Error("Signature should not verify")
	}

	vec[2] = 0
	aggSig = AggregateSignaturesVec(sigs, vec)
	aggPub = AggregatePublicKeysVec(pubkeys, vec)
	if !aggSig.Verify(msg, aggPub) {
		t.Error("Signature did not verify")
	}

}
