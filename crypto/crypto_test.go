package crypto

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thetatoken/ukulele/common"
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

func TestSignatureVerification(t *testing.T) {
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
