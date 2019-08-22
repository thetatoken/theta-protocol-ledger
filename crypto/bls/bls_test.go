package bls

import (
	"bytes"
	"crypto/rand"
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
	sig := priv.Sign(msg, 0)
	if !sig.Verify(msg, pub, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate(t *testing.T) {
	pubkeys := make([]*PublicKey, 0, 100)
	sigs := make([]*Signature, 0, 100)
	msg := []byte("hello")
	for i := 0; i < 100; i++ {
		priv, _ := RandKey(rand.Reader)
		pub := priv.PublicKey()
		sig := priv.Sign(msg, 0)
		pubkeys = append(pubkeys, pub)
		sigs = append(sigs, sig)
	}
	aggSig := AggregateSignatures(sigs)
	if !aggSig.VerifyAggregate(pubkeys, msg, 0) {
		t.Error("Signature did not verify")
	}
}

func TestVerifyAggregate_ReturnsFalseOnEmptyPubKeyList(t *testing.T) {
	var pubkeys []*PublicKey
	sigs := make([]*Signature, 0, 100)
	msg := []byte("hello")

	aggSig := AggregateSignatures(sigs)
	if aggSig.VerifyAggregate(pubkeys, msg, 0 /*domain*/) != false {
		t.Error("Expected VerifyAggregate to return false with empty input " +
			"of public keys.")
	}
}
