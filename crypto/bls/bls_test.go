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
	pubkeys := make([]*PublicKey, 0, 100)
	sigs := make([]*Signature, 0, 100)
	vec := make([]uint64, 0, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv, _ := RandKey(rand.Reader)
		pub := priv.PublicKey()
		sig := priv.Sign(msg)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
		vec = append(vec, mrand.Uint64())
	}
	aggSig := AggregateSignaturesVec(sigs, vec)
	aggPub := AggregatePublicKeysVec(pubkeys, vec)
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

// func TestVerifyAggregate(t *testing.T) {
// 	pubkeys := make([]*PublicKey, 0, 100)
// 	sigs := make([]*Signature, 0, 100)
// 	msg := []byte("hello")
// 	for i := 0; i < 100; i++ {
// 		priv, _ := RandKey(rand.Reader)
// 		pub := priv.PublicKey()
// 		sig := priv.Sign(msg, 0)
// 		pubkeys = append(pubkeys, pub)
// 		sigs = append(sigs, sig)
// 	}
// 	aggSig := AggregateSignatures(sigs)
// 	if !aggSig.VerifyWithDomain(pubkeys, msg, 0) {
// 		t.Error("Signature did not verify")
// 	}
// }

// func TestVerifyIncrementalAggregate(t *testing.T) {
// 	pubkeys := make([]*PublicKey, 0, 100)
// 	sigs := make([]*Signature, 0, 100)
// 	msg := []byte("hello")
// 	agPubkey := PubkeyZero()
// 	agSig := SignatureZero()

// 	// Aggregate pubkey/signature from all signers.
// 	for i := 0; i < 100; i++ {
// 		priv, _ := RandKey(rand.Reader)
// 		pub := priv.PublicKey()
// 		sig := priv.Sign(msg, 0)
// 		agPubkey.Aggregate(pub)
// 		agSig.Aggregate(sig)
// 		pubkeys = append(pubkeys, pub)
// 		sigs = append(sigs, sig)
// 	}
// 	if !agSig.Verify(msg, agPubkey, 0) {
// 		t.Error("Signature should verify")
// 	}

// 	// Aggregate pub/sig from the same signer more than once.
// 	for i := 0; i < 100; i++ {
// 		agPubkey.Aggregate(pubkeys[i])
// 		agSig.Aggregate(sigs[i])
// 	}
// 	if !agSig.Verify(msg, agPubkey, 0) {
// 		t.Error("Signature should verify")
// 	}

// 	// Aggregate pubkey without aggregating corresponding signature.
// 	agPubkey.Aggregate(pubkeys[0])
// 	if agSig.Verify(msg, agPubkey, 0) {
// 		t.Error("Signature should not verify")
// 	}
// }

// func TestVerifyAggregate_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
// 	var pubkeys []*PublicKey
// 	sigs := make([]*Signature, 0, 100)
// 	msg := []byte("hello")

// 	aggSig := AggregateSignatures(sigs)
// 	if aggSig.VerifyAggregate(pubkeys, msg, 0 /*domain*/) != false {
// 		t.Error("Expected VerifyAggregate to return false with empty input " +
// 			"of public keys.")
// 	}
// }
