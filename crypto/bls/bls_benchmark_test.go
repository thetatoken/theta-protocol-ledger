package bls

import (
	"crypto/rand"
	"testing"
)

func BenchmarkSignature_Verify(b *testing.B) {
	sk, err := RandKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	msg := []byte("Some msg")
	domain := uint64(42)
	sig := sk.Sign(msg, domain)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sig.Verify(msg, sk.PublicKey(), domain) {
			b.Fatal("could not verify sig")
		}
	}
}

func BenchmarkSignature_VerifyAggregate(b *testing.B) {
	sigN := 1280
	msg := []byte("signed message")
	domain := uint64(0)

	var aggregated *Signature
	var pks []*PublicKey
	for i := 0; i < sigN; i++ {
		sk, err := RandKey(rand.Reader)
		if err != nil {
			b.Fatal(err)
		}
		sig := sk.Sign(msg, domain)
		aggregated = AggregateSignatures([]*Signature{aggregated, sig})
		pks = append(pks, sk.PublicKey())
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !aggregated.VerifyAggregate(pks, msg, domain) {
			b.Fatal("could not verify aggregate sig")
		}
	}
}
