package bls

import (
	mrand "math/rand"
	"testing"
)

func TestMarshalUnmarshal(t *testing.T) {
	sk, _ := RandKey()

	b := sk.ToBytes()
	sk1, err := SecretKeyFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if !sk1.Equals(sk) {
		t.Fatal()
	}

	pk := sk.PublicKey()
	b = pk.ToBytes()
	pk1, err := PublicKeyFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if !pk1.Equals(pk) {
		t.Fatal()
	}

	sig := sk.Sign([]byte("hello"))
	b = sig.ToBytes()
	sig1, err := SignatureFromBytes(b)
	if err != nil {
		t.Fatal(err)
	}
	if !sig1.Equals(sig) {
		t.Fatal()
	}
}

func TestSignVerify(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	if !sig.Verify(msg, pub) {
		t.Error("Signature did not verify")
	}
}

func TestPop(t *testing.T) {
	priv, _ := RandKey()
	pop := priv.PopProve()
	if !pop.PopVerify(priv.PublicKey()) {
		t.Error("PopVerify failed")
	}
}

func TestCopy(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)

	pub2 := pub.Copy()
	pub2.Aggregate(pub)
	if pub2.Equals(pub) {
		t.Error("Copy failed")
	}

	sig2 := sig.Copy()
	sig2.Aggregate(sig)
	if sig2.Equals(sig) {
		t.Error("Copy failed")
	}
}

func TestSignIdentity(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	if !sig.Verify(msg, pub) {
		t.Error("Signature did not verify")
	}

	identity := NewAggregateSignature()
	sig2 := sig.Copy()
	sig2.Aggregate(identity)
	if !sig.Equals(sig2) {
		t.Error("Identity operation failed")
	}
}

func TestPubkeyIdentity(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()

	identity := NewAggregatePubkey()
	pub2 := pub.Copy()
	pub2.Aggregate(identity)
	if !pub.Equals(pub2) {
		t.Error("Identity operation failed")
	}
}

func TestSigExp(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()
	msg := []byte("hello")
	sig := priv.Sign(msg)
	if !sig.Verify(msg, pub) {
		t.Error("Signature did not verify")
	}

	// exp 0
	sig1 := NewAggregateSignature()
	tmp := sig.Copy()
	sig2 := sigExp(tmp, 0)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(1) is invalid")
	}

	// exp 1
	sig1 = sig.Copy()
	tmp = sig.Copy()
	sig2 = sigExp(tmp, 1)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(1) is invalid")
	}

	// exp 2
	sig1 = sig.Copy()
	sig1.Aggregate(sig)
	tmp = sig.Copy()
	sig2 = sigExp(tmp, 2)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(2) is invalid")
	}

	// exp 3
	sig1 = sig.Copy()
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	tmp = sig.Copy()
	sig2 = sigExp(tmp, 3)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(3) is invalid")
	}

	// exp 4
	sig1 = sig.Copy()
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	tmp = sig.Copy()
	sig2 = sigExp(tmp, 4)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(4) is invalid")
	}

	// exp 5
	sig1 = sig.Copy()
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	sig1.Aggregate(sig)
	tmp = sig.Copy()
	sig2 = sigExp(tmp, 5)
	if !sig1.Equals(sig2) {
		t.Fatal("sigExp(5) is invalid")
	}
}

func TestPubExp(t *testing.T) {
	priv, _ := RandKey()
	pub := priv.PublicKey()

	// exp 0
	pub1 := NewAggregatePubkey()
	tmp := pub.Copy()
	pub2 := pubkeyExp(tmp, 0)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExps(1) is invalid")
	}

	// exp 1
	pub1 = pub.Copy()
	tmp = pub.Copy()
	pub2 = pubkeyExp(tmp, 1)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExp(1) is invalid")
	}

	// exp 2
	pub1 = pub.Copy()
	tmp = pub.Copy()
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExp(2) is invalid")
	}

	// exp 3
	pub1 = pub.Copy()
	tmp = pub.Copy()
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExp(3) is invalid")
	}

	// exp 4
	pub1 = pub.Copy()
	tmp = pub.Copy()
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExp(4) is invalid")
	}

	// exp 5
	pub1 = pub.Copy()
	tmp = pub.Copy()
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	pub2 = pubkeyExp(tmp, 1)
	if !pub1.Equals(pub2) {
		t.Fatal("pubExp(5) is invalid")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*PublicKey, 100, 100)
	sigs := make([]*Signature, 100, 100)
	vec := make([]uint32, 100, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv, _ := RandKey()
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
	if !aggSig.Equals(sigs[0]) {
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
