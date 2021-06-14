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

func aggregatePublicKeysVecWithoutSkippingZeroEntries(p []*PublicKey, vec []uint32) *PublicKey {
	if len(p) != len(vec) {
		panic("len(pubkeys) must be equal to len(vec)")
	}
	newPub := NewAggregatePubkey()
	for i, pub := range p {
		newPub.Aggregate(pubkeyExp(pub, vec[i]))
	}
	return newPub
}

func aggregateSignaturesVecWithoutSkippingZeroEntries(s []*Signature, vec []uint32) *Signature {
	if len(s) != len(vec) {
		panic("len(sigs) must be equal to len(vec)")
	}
	newSig := NewAggregateSignature()
	for i, sig := range s {
		newSig.Aggregate(sigExp(sig, vec[i]))
	}
	return newSig
}

func TestAggregatePublicKeysVecSkipZeroEntries(t *testing.T) {
	numEntries := 300
	pubkeys := make([]*PublicKey, numEntries, numEntries)
	vec := make([]uint32, numEntries, numEntries)
	for i := 0; i < numEntries; i++ {
		priv, _ := RandKey()
		pub := priv.PublicKey()
		pubkeys[i] = pub
	}

	// ------- TEST A ------- //

	// Deterministic vector
	for i := 0; i < numEntries; i++ {
		if i%3 == 0 || i%7 == 1 {
			vec[i] = uint32(3*i*i*i + 4532*i*i + 2342*i)
		} else {
			vec[i] = 0
		}
	}

	aggPub1 := AggregatePublicKeysVec(pubkeys, vec)
	aggPub2 := aggregatePublicKeysVecWithoutSkippingZeroEntries(pubkeys, vec)
	if !aggPub1.Equals(aggPub2) {
		t.Errorf("Public key not equal, aggPub1: %v, aggPub2: %v", aggPub1, aggPub2)
	}

	t.Logf("TEST A - vec: %v", vec)
	t.Logf("TEST A - aggPub1: %v", aggPub1)
	t.Logf("TEST A - aggPub2: %v", aggPub2)

	// ------- TEST B ------- //

	// Random vector
	for i := 0; i < numEntries; i++ {
		vec[i] = 0
		if mrand.Intn(100) < 30 {
			vec[i] = uint32(mrand.Uint64())
		}
	}

	aggPub3 := AggregatePublicKeysVec(pubkeys, vec)
	aggPub4 := aggregatePublicKeysVecWithoutSkippingZeroEntries(pubkeys, vec)
	if !aggPub3.Equals(aggPub4) {
		t.Errorf("Public key not equal, aggPub1: %v, aggPub2: %v", aggPub3, aggPub4)
	}

	t.Logf("TEST B - vec: %v", vec)
	t.Logf("TEST B - aggPub3: %v", aggPub3)
	t.Logf("TEST B - aggPub4: %v", aggPub4)
}

func TestAggregateSignaturesVecSkipZeroEntries1(t *testing.T) {
	numEntries := 500
	pubkeys := make([]*PublicKey, numEntries, numEntries)
	sigs := make([]*Signature, numEntries, numEntries)
	vec := make([]uint32, numEntries, numEntries)
	msg := []byte("hello world")
	for i := 0; i < numEntries; i++ {
		priv, _ := RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg)
		pubkeys[i] = pub
		sigs[i] = sig
	}

	// Deterministic vector
	for i := 0; i < numEntries; i++ {
		if i%5 == 0 || i%6 == 1 {
			vec[i] = uint32(2*i*i*i + 762*i*i + 9872*i)
		} else {
			vec[i] = 0
		}
	}

	aggPub1 := AggregatePublicKeysVec(pubkeys, vec)
	aggPub2 := aggregatePublicKeysVecWithoutSkippingZeroEntries(pubkeys, vec)
	if !aggPub1.Equals(aggPub2) {
		t.Errorf("Public key not equal, aggPub1: %v, aggPub2: %v", aggPub1, aggPub2)
	}

	aggSig1 := AggregateSignaturesVec(sigs, vec)
	aggSig2 := aggregateSignaturesVecWithoutSkippingZeroEntries(sigs, vec)
	if !aggSig1.Equals(aggSig2) {
		t.Errorf("Signature not equal, aggSig1: %v, aggSig2: %v", aggSig1, aggSig2)
	}

	// Should cross verify
	if !aggSig1.Verify(msg, aggPub1) {
		t.Error("Signature did not verify")
	}
	if !aggSig1.Verify(msg, aggPub2) {
		t.Error("Signature did not verify")
	}
	if !aggSig2.Verify(msg, aggPub1) {
		t.Error("Signature did not verify")
	}
	if !aggSig2.Verify(msg, aggPub2) {
		t.Error("Signature did not verify")
	}

	t.Logf("TEST - vec: %v", vec)
	t.Logf("TEST - aggSig1: %v", aggSig1)
	t.Logf("TEST - aggSig2: %v", aggSig2)

	//t.Errorf("INTENTIONAL ERROR FOR LOG PRINTING")
}

func TestAggregateSignaturesVecSkipZeroEntries2(t *testing.T) {
	numEntries := 1000
	pubkeys := make([]*PublicKey, numEntries, numEntries)
	sigs := make([]*Signature, numEntries, numEntries)
	vec := make([]uint32, numEntries, numEntries)
	msg := []byte("The Times 03/Jan/2009 Chancellor on brink of second bailout for banks")
	for i := 0; i < numEntries; i++ {
		priv, _ := RandKey()
		pub := priv.PublicKey()
		sig := priv.Sign(msg)
		pubkeys[i] = pub
		sigs[i] = sig

		//t.Logf("TEST - sig[%v]: %v", i, sig)
	}

	// Random vector
	for i := 0; i < numEntries; i++ {
		vec[i] = 0
		if mrand.Intn(100) < 30 {
			vec[i] = uint32(mrand.Uint64())
		}
	}

	aggPub1 := AggregatePublicKeysVec(pubkeys, vec)
	aggPub2 := aggregatePublicKeysVecWithoutSkippingZeroEntries(pubkeys, vec)
	if !aggPub1.Equals(aggPub2) {
		t.Errorf("Public key not equal, aggPub1: %v, aggPub2: %v", aggPub1, aggPub2)
	}

	aggSig1 := AggregateSignaturesVec(sigs, vec)
	aggSig2 := aggregateSignaturesVecWithoutSkippingZeroEntries(sigs, vec)
	if !aggSig1.Equals(aggSig2) {
		t.Errorf("Signature not equal, aggSig1: %v, aggSig2: %v", aggSig1, aggSig2)
	}

	// Should cross verify
	if !aggSig1.Verify(msg, aggPub1) {
		t.Error("Signature did not verify")
	}
	if !aggSig1.Verify(msg, aggPub2) {
		t.Error("Signature did not verify")
	}
	if !aggSig2.Verify(msg, aggPub1) {
		t.Error("Signature did not verify")
	}
	if !aggSig2.Verify(msg, aggPub2) {
		t.Error("Signature did not verify")
	}

	t.Logf("TEST - vec: %v", vec)
	t.Logf("TEST - aggSig1: %v", aggSig1)
	t.Logf("TEST - aggSig2: %v", aggSig2)

	//t.Errorf("INTENTIONAL ERROR FOR LOG PRINTING")
}
