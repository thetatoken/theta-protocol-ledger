package crypto

import (
	"encoding/hex"
	"math/big"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/theta/common"
)

func TestHash(t *testing.T) {
	assert := assert.New(t)

	// Short message
	shortMsg := common.Bytes("Hello world!")

	hashBytes1 := Keccak256(shortMsg)
	expectedHashBytes1, err := hex.DecodeString("ecd0e108a98e192af1d2c25055f4e3bed784b5c877204e73219a5203251feaab")
	assert.Nil(err)
	assert.Equal(32, len(hashBytes1))
	assert.Equal(expectedHashBytes1, hashBytes1)

	hash1 := Keccak256Hash(shortMsg)
	expectedHash1, err := hex.DecodeString("ecd0e108a98e192af1d2c25055f4e3bed784b5c877204e73219a5203251feaab")
	assert.Nil(err)
	assert.Equal(32, len(hash1))
	assert.Equal(expectedHash1, hash1[:])

	// Long message
	longMsg := common.Bytes("Bitcoin Price Ends September 2018 At Lowest Volatility in 15 Months. Bitcoin traded in a range of just under $1,500 over the course of the month of September, its narrowest monthly trading range since July 2017, data reveals. At close of trading Sunday, bitcoin (BTC) officially ended the 30-day period with a trading range of $1,329, with prices oscillating between a low of $6,100 and a high of $7,429. Overall, this was the lowest one-month range since July 2017, when bitcoin traded in a $1,095.8 window, according to data from Bitfinex. Further, the monthly trading volume throughout September marked its lowest amount since April 2017, according to the exchange, one of the world's largest. Periods of low volatility often come to a boisterous end for bitcoin especially when accompanied by low volume, so it seems the cryptocurrency is gearing up for a decisive move in either direction.")

	hashBytes2 := Keccak256(longMsg)
	expectedHashBytes2, err := hex.DecodeString("26d7f2c2b3d1f4abfe0aa14e2fdeaf80d3dfb45a6e269476efed53edec603fed")
	assert.Nil(err)
	assert.Equal(32, len(hashBytes2))
	assert.Equal(expectedHashBytes2, hashBytes2)

	hash2 := Keccak256Hash(longMsg)
	expectedHash2, err := hex.DecodeString("26d7f2c2b3d1f4abfe0aa14e2fdeaf80d3dfb45a6e269476efed53edec603fed")
	assert.Nil(err)
	assert.Equal(32, len(hash2))
	assert.Equal(expectedHash2, hash2[:])
}

func TestKeyBasics(t *testing.T) {
	assert := assert.New(t)

	randPrivKey, randPubKey, err := GenerateKeyPair()
	assert.NotNil(randPrivKey)
	assert.NotNil(randPubKey)
	assert.Nil(err)
	assert.Equal(randPrivKey.PublicKey(), randPubKey)
	assert.False(randPubKey.IsEmpty())
	assert.Equal(common.AddressLength, len(randPubKey.Address()))
	assert.Equal(65, len(randPubKey.ToBytes()))

	seed := "niceseed123"
	seededPrivKey1, seededPubKey1, err := TEST_GenerateKeyPairWithSeed(seed)
	assert.NotNil(seededPrivKey1)
	assert.NotNil(seededPubKey1)
	assert.Nil(err)
	seededPrivKey2, seededPubKey2, err := TEST_GenerateKeyPairWithSeed(seed)
	assert.NotNil(seededPrivKey2)
	assert.NotNil(seededPubKey2)
	assert.Nil(err)

	assert.Equal(seededPrivKey1, seededPrivKey2) // repeatability test
	assert.Equal(seededPubKey1, seededPubKey2)   // repeatability test
	assert.Equal(seededPrivKey1.PublicKey(), seededPubKey1)
	assert.Equal(seededPrivKey2.PublicKey(), seededPubKey2)
	assert.Equal(common.AddressLength, len(seededPubKey1.Address()))
	assert.Equal(common.AddressLength, len(seededPubKey2.Address()))
	assert.Equal(65, len(seededPubKey1.ToBytes()))
	assert.Equal(65, len(seededPubKey2.ToBytes()))
}

func TestToAndFromBytes(t *testing.T) {
	assert := assert.New(t)

	privKey, pubKey, err := TEST_GenerateKeyPairWithSeed("USLawmakers")
	assert.Nil(err)

	msg := common.Bytes("US Lawmakers Move Forward on Crypto Task Force Proposal")
	sig, err := privKey.Sign(msg)
	assert.Nil(err)

	privKeyBytes := privKey.ToBytes()
	recoveredPrivKey, err := PrivateKeyFromBytes(privKeyBytes)
	assert.Nil(err)
	assert.Equal(privKey, recoveredPrivKey)
	t.Logf("PrivateBytes  : %v", hex.EncodeToString(privKeyBytes))

	pubKeyBytes := pubKey.ToBytes()
	recoveredPubKey, err := PublicKeyFromBytes(pubKeyBytes)
	assert.Nil(err)
	assert.Equal(pubKey, recoveredPubKey)
	t.Logf("PublicBytes   : %v", hex.EncodeToString(pubKeyBytes))

	sigBytes := sig.ToBytes()
	recoveredSig, err := SignatureFromBytes(sigBytes)
	assert.Nil(err)
	assert.Equal(sig, recoveredSig)
	t.Logf("SignatureBytes: %v", hex.EncodeToString(sigBytes))
}

func TestPrivKeyFromBytes(t *testing.T) {
	assert := assert.New(t)

	Zero := new(big.Int).SetUint64(0)

	// --------------------------- Private Key #1 --------------------------- //

	privKeyBytes, _ := hex.DecodeString("f3e4bfb656a98beac6931c86f48de62a7e469624d359f6b067b7f4a45a136446")
	privKey, err := PrivateKeyFromBytes(privKeyBytes)
	assert.Nil(err)

	// the key string should always be interpreted as a positive big int
	assert.True(privKey.D().Cmp(Zero) > 0)
	t.Logf("privKey.D : %v", privKey.D())

	pubKey := privKey.PublicKey()
	pubKeyBytes := pubKey.ToBytes()
	t.Logf("PublicByte: %v", hex.EncodeToString(pubKeyBytes))
	assert.Equal("046adefd8a2b7a581fab692cae4160d7399bc8280972122206a968762f2898bd2376879a7430520e6477a1d6c3d07f2688d6d71a83848b5086808b2d04847bea8e", hex.EncodeToString(pubKeyBytes))

	address := pubKey.Address()
	t.Logf("Address   : %v", address.Hex())
	assert.Equal("0x511f5B5aF946eDca88217EB9404477a95CB5C3F4", address.Hex())

	// --------------------------- Private Key #2 --------------------------- //

	privKeyBytes, _ = hex.DecodeString("93a90ea508331dfdf27fb79757d4250b4e84954927ba0073cd67454ac432c737")
	privKey, err = PrivateKeyFromBytes(privKeyBytes)
	assert.Nil(err)

	// the key string should always be interpreted as a positive big int
	assert.True(privKey.D().Cmp(Zero) > 0)
	t.Logf("privKey.D : %v", privKey.D())

	pubKey = privKey.PublicKey()
	pubKeyBytes = pubKey.ToBytes()
	t.Logf("PublicByte: %v", hex.EncodeToString(pubKeyBytes))
	assert.Equal("048e8d53fd435265ad074597cc3e202f8e935cfb57925bb51316252027cb08767fb8099226414732543c4b5cbaa64b4ee8f173ba559258a0b5f633a0d11509e78b", hex.EncodeToString(pubKeyBytes))

	address = pubKey.Address()
	t.Logf("Address   : %v", address.Hex())
	assert.Equal("0x2E833968E5bB786Ae419c4d13189fB081Cc43bab", address.Hex())
}

func TestAddressRecovery(t *testing.T) {
	assert := assert.New(t)

	privKey, pubKey, err := TEST_GenerateKeyPairWithSeed("test_seed_xyz")
	assert.Nil(err)

	msg1 := common.Bytes("ABCD has four letters")
	msg2 := common.Bytes("Hello World!")

	sig, err := privKey.Sign(msg1)

	address, err := sig.RecoverSignerAddress(msg1)
	assert.Nil(err)
	assert.True(address == pubKey.Address())

	fakeAddr, err := sig.RecoverSignerAddress(msg2)
	assert.True(address != fakeAddr) // privKey did not sign msg2

	log.Infof("real address: %v", address)
	log.Infof("fake address: %v", fakeAddr)
}

func TestSignaureVerifyBytes(t *testing.T) {
	assert := assert.New(t)

	privKey, pubKey, err := TEST_GenerateKeyPairWithSeed("test_seed")
	assert.Nil(err)
	addr := pubKey.Address()

	msg1 := common.Bytes("Hello world!")
	msg2 := common.Bytes("Foo bar!")
	sig1, err := privKey.Sign(msg1)
	assert.Nil(err)
	sig2, err := privKey.Sign(msg2)
	assert.Nil(err)
	assert.True(sig1.Verify(msg1, addr))
	assert.False(sig2.Verify(msg1, addr))

	// Should not panic
	nilSig := (*Signature)(nil)
	assert.False(nilSig.Verify(msg1, addr))

	emptySig, err := SignatureFromBytes(common.Bytes{})
	assert.Nil(err)
	assert.False(emptySig.Verify(msg1, addr))

	emptyAddr := common.BytesToAddress(common.Bytes{})
	assert.False(sig1.Verify(msg1, emptyAddr))

	anotherAddr := common.BytesToAddress(common.Bytes("hello"))
	assert.False(sig1.Verify(msg1, anotherAddr))
}

func TestSignatureVerification1(t *testing.T) {
	assert := assert.New(t)

	privKey, pubKey, err := TEST_GenerateKeyPairWithSeed("test_seed")
	assert.Nil(err)

	shortMsg := common.Bytes("Hello world!")
	sig1, err := privKey.Sign(shortMsg)
	assert.Nil(err)
	assert.True(pubKey.VerifySignature(shortMsg, sig1))
	assert.False(pubKey.VerifySignature(shortMsg, nil))
	fakeShortMsgSig1Data, err := hex.DecodeString("1234567890123456789012345678901234567890123456789012345678901234123456789012345678901234567890123456789012345678901234567890120400")
	assert.Nil(err)
	fakeShortMsgSig1 := &Signature{data: fakeShortMsgSig1Data}
	assert.True(len(fakeShortMsgSig1.ToBytes()) == 65)
	assert.False(pubKey.VerifySignature(shortMsg, fakeShortMsgSig1))
	fakeShortMsgSig2 := &Signature{data: common.Bytes("82ksiwpskfa")}
	assert.True(len(fakeShortMsgSig2.ToBytes()) != 65)
	assert.False(pubKey.VerifySignature(shortMsg, fakeShortMsgSig2))

	longMsg := common.Bytes("Bitcoin Price Ends September 2018 At Lowest Volatility in 15 Months. Bitcoin traded in a range of just under $1,500 over the course of the month of September, its narrowest monthly trading range since July 2017, data reveals. At close of trading Sunday, bitcoin (BTC) officially ended the 30-day period with a trading range of $1,329, with prices oscillating between a low of $6,100 and a high of $7,429. Overall, this was the lowest one-month range since July 2017, when bitcoin traded in a $1,095.8 window, according to data from Bitfinex. Further, the monthly trading volume throughout September marked its lowest amount since April 2017, according to the exchange, one of the world's largest. Periods of low volatility often come to a boisterous end for bitcoin especially when accompanied by low volume, so it seems the cryptocurrency is gearing up for a decisive move in either direction.")
	sig2, err := privKey.Sign(longMsg)
	assert.Nil(err)
	assert.True(pubKey.VerifySignature(longMsg, sig2))
	assert.False(pubKey.VerifySignature(longMsg, nil))
	fakeLongMsgSig1Data, err := hex.DecodeString("abcd1234e5abcd1234e5abcd1234e5abcd1234e5abcd1234e5abcd1234e5abcdeabcd1234e5abcd1234e5abcd1234e5abcd1234e5abcd1234e5abcd1234e5abcde")
	assert.Nil(err)
	fakeLongMsgSig1 := &Signature{data: fakeLongMsgSig1Data}
	assert.True(len(fakeLongMsgSig1.ToBytes()) == 65)
	assert.False(pubKey.VerifySignature(longMsg, fakeLongMsgSig1))
	fakeLongMsgSig2 := &Signature{data: common.Bytes("iwk29fiwkw")}
	assert.True(len(fakeLongMsgSig2.ToBytes()) != 65)
	assert.False(pubKey.VerifySignature(longMsg, fakeLongMsgSig2))
}

func TestSignatureVerification2(t *testing.T) {
	assert := assert.New(t)

	privKeyA, pubKeyA, err := TEST_GenerateKeyPairWithSeed("test_seed_A")
	assert.Nil(err)
	privKeyB, pubKeyB, err := TEST_GenerateKeyPairWithSeed("test_seed_B")
	assert.Nil(err)

	msg1 := common.Bytes("ABCD has four letters")
	msg2 := common.Bytes("USA has 50 states")

	// Cross-Message Checks
	sig1A, err := privKeyA.Sign(msg1)
	assert.True(pubKeyA.VerifySignature(msg1, sig1A))
	sig2A, err := privKeyA.Sign(msg2)
	assert.True(pubKeyA.VerifySignature(msg2, sig2A))
	assert.False(pubKeyA.VerifySignature(msg1, sig2A))
	assert.False(pubKeyA.VerifySignature(msg2, sig1A))

	// Cross-PublicKey Checks
	sig1B, err := privKeyB.Sign(msg1)
	assert.True(pubKeyB.VerifySignature(msg1, sig1B))
	sig2B, err := privKeyB.Sign(msg2)
	assert.True(pubKeyB.VerifySignature(msg2, sig2B))
	assert.False(pubKeyA.VerifySignature(msg1, sig1B))
	assert.False(pubKeyB.VerifySignature(msg1, sig1A))
	assert.False(pubKeyA.VerifySignature(msg2, sig2B))
	assert.False(pubKeyB.VerifySignature(msg2, sig2A))
}
