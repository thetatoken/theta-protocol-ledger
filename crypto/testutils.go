package crypto

import (
	"crypto/ecdsa"

	"github.com/thetatoken/theta/common"
)

//
// ----------------------------- APIs ONLY for TESTs ----------------------------- //
//
// WARNING: The following APIs are intended only for unit test case for better repeatibility.
//          They should NOT be used in the production code.

// TEST_GenerateKeyPairWithSeed generates a random private/public key pair with the given seed string
func TEST_GenerateKeyPairWithSeed(seed string) (*PrivateKey, *PublicKey, error) {
	trr := newTestRandReader(seed)
	ske, err := ecdsa.GenerateKey(s256(), trr)
	pke := &(ske.PublicKey)
	return &PrivateKey{privKey: ske}, &PublicKey{pubKey: pke}, err
}

type testRandReader struct {
	seed common.Bytes
}

func newTestRandReader(seedStr string) *testRandReader {
	return &testRandReader{
		seed: []byte(seedStr),
	}
}

func (trr *testRandReader) Read(b []byte) (int, error) {
	n := copy(b[:], trr.seed)
	return n, nil
}
