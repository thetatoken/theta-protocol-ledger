package bls

import (
	"testing"
)

func BenchmarkSignature_Verify(b *testing.B) {
	sk, err := RandKey()
	if err != nil {
		b.Fatal(err)
	}
	msg := []byte("Some msg")
	sig := sk.Sign(msg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !sig.Verify(msg, sk.PublicKey()) {
			b.Fatal("could not verify sig")
		}
	}
}

// func BenchmarkSignature_VerifyAggregate(b *testing.B) {
// 	sigN := 1280
// 	msg := []byte("signed message")
// 	domain := uint64(0)

// 	var aggregated *Signature
// 	var pks []*PublicKey
// 	for i := 0; i < sigN; i++ {
// 		sk, err := RandKey(rand.Reader)
// 		if err != nil {
// 			b.Fatal(err)
// 		}
// 		sig := sk.SignWithDomain(msg, domain)
// 		aggregated = AggregateSignatures([]*Signature{aggregated, sig})
// 		pks = append(pks, sk.PublicKey())
// 	}

// 	b.ResetTimer()
// 	for i := 0; i < b.N; i++ {
// 		if !aggregated.VerifyWithDomain(pks, msg, domain) {
// 			b.Fatal("could not verify aggregate sig")
// 		}
// 	}
// }
