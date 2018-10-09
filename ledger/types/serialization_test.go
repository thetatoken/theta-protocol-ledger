package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
)

// func TestPubkey(t *testing.T) {
// 	assert := assert.New(t)

// 	_, pubkey1, err := crypto.GenerateKeyPair()
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Test conversion to/from proto message.
// 	pk := PublicKeyToProto(pubkey1)
// 	pubkey2 := PublicKeyFromProto(pk)
// 	assert.EqualValues(pubkey1, pubkey2)

// 	// Test conversion to/from bytes.
// 	b, err := ToBytes(pubkey1)
// 	assert.Nil(err)
// 	var pubkey3 crypto.PublicKey
// 	err = FromBytes(b, &pubkey3)
// 	assert.Nil(err)
// 	assert.EqualValues(*pubkey1, pubkey3)

// 	// Verify bytes are deterministic.
// 	b2, err := ToBytes(pubkey1)
// 	assert.Nil(err)
// 	assert.EqualValues(b, b2)
// }

// func TestPrivkey(t *testing.T) {
// 	assert := assert.New(t)

// 	privKey, _, err := crypto.GenerateKeyPair()
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Test conversion to/from proto message.
// 	pk := PrivateKeyToProto(privKey)
// 	privkey2 := PrivateKeyFromProto(pk)
// 	assert.EqualValues(privKey, privkey2)

// 	// Test conversion to/from bytes.
// 	b, err := ToBytes(privKey)
// 	assert.Nil(err)
// 	var privkey3 crypto.PrivateKey
// 	err = FromBytes(b, &privkey3)
// 	assert.Nil(err)
// 	assert.EqualValues(*privKey, privkey3)

// 	// Verify bytes are deterministic.
// 	b2, err := ToBytes(privKey)
// 	assert.Nil(err)
// 	assert.EqualValues(b, b2)
// }

// func TestSignature(t *testing.T) {
// 	assert := assert.New(t)

// 	var b [64]byte
// 	for i := 0; i < len(b); i++ {
// 		b[i] = byte(i)
// 	}

// 	privKey, _, err := crypto.GenerateKeyPair()
// 	if err != nil {
// 		panic(err)
// 	}

// 	sig1, err := privKey.Sign(b[:])
// 	if err != nil {
// 		panic(err)
// 	}

// 	// Test conversion to/from proto message.
// 	msg := SignatureToProto(sig1)
// 	sig2 := SignatureFromProto(msg)
// 	assert.EqualValues(sig1, sig2)

// 	// Test conversion to/from bytes.
// 	bb, err := ToBytes(sig1)
// 	assert.Nil(err)
// 	var sig3 crypto.Signature
// 	err = FromBytes(bb, &sig3)
// 	assert.Nil(err)
// 	assert.EqualValues(sig1, &sig3)

// 	// Verify bytes are deterministic.
// 	bb2, err := ToBytes(sig1)
// 	assert.Nil(err)
// 	assert.EqualValues(bb, bb2)
// }

// func TestOverSpendingProof(t *testing.T) {
// 	assert := assert.New(t)

// 	subjs := []OverspendingProof{
// 		{},
// 		{
// 			ReserveSequence: 1,
// 			ServicePayments: []ServicePaymentTx{{
// 				Fee:             Coin{Denom: "ThetaWei", Amount: 123},
// 				Gas:             123,
// 				Source:          TxInput{Address: getTestAddress("123")},
// 				Target:          TxInput{Address: getTestAddress("456")},
// 				PaymentSequence: 1,
// 				ReserveSequence: 1,
// 			}},
// 		},
// 	}

// 	for _, subj := range subjs {
// 		// Test conversion to/from bytes.
// 		b, err := ToBytes(&subj)
// 		assert.Nil(err)
// 		subj2 := &OverspendingProof{}
// 		err = FromBytes(b, subj2)
// 		assert.Nil(err)
// 		assert.EqualValues(&subj, subj2)

// 		// Verify bytes are deterministic.
// 		b2, err := ToBytes(&subj)
// 		assert.Nil(err)
// 		assert.EqualValues(b, b2)
// 	}
// }

// func TestCoinsSerialization(t *testing.T) {
// 	assert := assert.New(t)

// 	cc := []Coins{Coins(nil), {}, {{Denom: "ThetaWei", Amount: 123}}}

// 	for _, c := range cc {
// 		// Test conversion to/from proto message.
// 		msg := CoinsToProto(c)
// 		c2 := CoinsFromProto(msg)
// 		assert.EqualValues(c, c2)
// 	}
// }

// func TestAccount(t *testing.T) {
// 	assert := assert.New(t)

// 	_, pubkey1, err := crypto.GenerateKeyPair()
// 	if err != nil {
// 		panic(err)
// 	}

// 	rf := ReservedFund{
// 		// TargetAddresses: data.Bytes("target_address"),
// 		ReserveSequence: 123,
// 		EndBlockHeight:  456,
// 		InitialFund:     Coins{{Denom: "ThetaWei", Amount: 789}},
// 	}
// 	accounts := []Account{{
// 		PubKey:                 pubkey1,
// 		Sequence:               31,
// 		LastUpdatedBlockHeight: 22,
// 		Balance:                Coins{{Denom: "ThetaWei", Amount: 123}},
// 		ReservedFunds:          []ReservedFund{rf},
// 	}, {
// 	// Test with empty fields.
// 	}}

// 	for _, account1 := range accounts {
// 		// Test conversion to/from proto message.
// 		msg := AccountToProto(&account1)

// 		account2 := AccountFromProto(msg)
// 		assert.EqualValues(&account1, account2)

// 		// Test conversion to/from bytes.
// 		b, err := ToBytes(&account1)
// 		assert.Nil(err)
// 		account3 := &Account{}
// 		err = FromBytes(b, account3)
// 		assert.Nil(err)
// 		assert.EqualValues(&account1, account3)

// 		// Verify bytes are deterministic.
// 		b2, err := ToBytes(&account1)
// 		assert.Nil(err)
// 		assert.EqualValues(b, b2)
// 	}
// }

// func TestInput(t *testing.T) {
// 	assert := assert.New(t)

// 	sk, pk, err := crypto.GenerateKeyPair()
// 	if err != nil {
// 		panic(err)
// 	}

// 	var b [64]byte
// 	for i := 0; i < len(b); i++ {
// 		b[i] = byte(i)
// 	}
// 	sig, err := sk.Sign(b[:])
// 	if err != nil {
// 		panic(err)
// 	}

// 	inputs := []TxInput{{
// 		Sequence: 123,
// 	}, {
// 		Address:   getTestAddress("123"),
// 		Coins:     Coins{{Denom: "ThetaWei", Amount: 456}},
// 		PubKey:    pk,
// 		Signature: sig,
// 	}}

// 	for _, input1 := range inputs {
// 		// Test conversion to/from proto message.
// 		msg := InputToProto(&input1)
// 		input2 := InputFromProto(msg)
// 		assert.EqualValues(&input1, input2)
// 	}
// }

// func TestOutput(t *testing.T) {
// 	assert := assert.New(t)

// 	outputs := []TxOutput{{}, {
// 		Address: getTestAddress("123"),
// 		Coins:   Coins{{Denom: "ThetaWei", Amount: 456}},
// 	}}

// 	for _, output1 := range outputs {
// 		// Test conversion to/from proto message.
// 		msg := OutputToProto(&output1)
// 		output2 := OutputFromProto(msg)
// 		assert.EqualValues(&output1, output2)
// 	}
// }

// func TestSplit(t *testing.T) {
// 	assert := assert.New(t)

// 	addr, err := hex.DecodeString("D7D25858609A250BCD698CBAA3DB6B285586657C")
// 	assert.Equal(err, nil)
// 	var address common.Address
// 	copy(address[:], addr)

// 	split1 := Split{
// 		Address:    address,
// 		Percentage: 40,
// 	}

// 	msg := SplitToProto(&split1)
// 	split2 := SplitFromProto(msg)
// 	assert.EqualValues(&split1, split2)
// }

// func TestSplitContract(t *testing.T) {
// 	assert := assert.New(t)

// 	addr, err := hex.DecodeString("D7D25858609A250BCD698CBAA3DB6B285586657C")
// 	assert.Equal(err, nil)
// 	var address common.Address
// 	copy(address[:], addr)

// 	split := Split{
// 		Address:    address,
// 		Percentage: 40,
// 	}

// 	splitContract1 := SplitContract{
// 		ResourceID:     []byte("rid0000001"),
// 		Splits:         []Split{split},
// 		EndBlockHeight: 1006,
// 	}

// 	msg := SplitContractToProto(&splitContract1)
// 	splitContract2 := SplitContractFromProto(msg)
// 	assert.EqualValues(&splitContract1, splitContract2)
// }
func TestPubKey(t *testing.T) {
	assert := assert.New(t)

	_, pubKey, _ := crypto.GenerateKeyPair()
	b, err := rlp.EncodeToBytes(pubKey)
	assert.Nil(err)

	ret := &crypto.PublicKey{}
	err = rlp.DecodeBytes(b, ret)
	assert.Nil(err)
	assert.Equal(pubKey.ToBytes().String(), ret.ToBytes().String())
}

func TestSignature(t *testing.T) {
	assert := assert.New(t)

	sig, err := crypto.SignatureFromBytes([]byte("I am a signature"))
	assert.Nil(err)

	b, err := rlp.EncodeToBytes(sig)
	assert.Nil(err)

	ret := &crypto.Signature{}
	err = rlp.DecodeBytes(b, ret)
	assert.Nil(err)
	assert.Equal(sig.ToBytes().String(), ret.ToBytes().String())
}

func TestTx(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	var tx1 Tx

	_, pubKey, _ := crypto.GenerateKeyPair()
	sig, _ := crypto.SignatureFromBytes([]byte("i am signature"))
	tx1 = &CoinbaseTx{
		Proposer: TxInput{
			Address:   getTestAddress("123"),
			PubKey:    pubKey,
			Signature: sig,
		},
		Outputs:     []TxOutput{{Address: getTestAddress("456")}, {Address: getTestAddress("888")}, {Address: getTestAddress("999")}},
		BlockHeight: uint64(999),
	}
	b, err := TxToBytes(tx1)
	require.Nil(err)
	tx2, err := TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*CoinbaseTx).Proposer.Address, tx2.(*CoinbaseTx).Proposer.Address)
	assert.Equal(tx1.(*CoinbaseTx).Proposer.PubKey, tx2.(*CoinbaseTx).Proposer.PubKey)
	assert.Equal(tx1.(*CoinbaseTx).Proposer.Signature, tx2.(*CoinbaseTx).Proposer.Signature)
	assert.Equal(tx1.(*CoinbaseTx).BlockHeight, tx2.(*CoinbaseTx).BlockHeight)

	tx1 = &SlashTx{
		Proposer:        TxInput{Address: getTestAddress("123")},
		SlashedAddress:  getTestAddress("456"),
		SlashProof:      common.Bytes("789"),
		ReserveSequence: 1,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*SlashTx).Proposer.Address, tx2.(*SlashTx).Proposer.Address)
	assert.Equal(tx1.(*SlashTx).ReserveSequence, tx2.(*SlashTx).ReserveSequence)
	assert.Equal(tx1.(*SlashTx).SlashedAddress, tx2.(*SlashTx).SlashedAddress)
	assert.Equal(tx1.(*SlashTx).SlashProof, tx2.(*SlashTx).SlashProof)

	tx1 = &SendTx{
		Fee:     NewCoins(123, 0),
		Gas:     123,
		Inputs:  []TxInput{{Address: getTestAddress("123")}, {Address: getTestAddress("798")}},
		Outputs: []TxOutput{{Address: getTestAddress("456")}, {Address: getTestAddress("888")}, {Address: getTestAddress("999")}},
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*SendTx).Inputs[0].Address, tx2.(*SendTx).Inputs[0].Address)
	assert.Equal(tx1.(*SendTx).Inputs[1].Address, tx2.(*SendTx).Inputs[1].Address)
	assert.Equal(tx1.(*SendTx).Outputs[0].Address, tx2.(*SendTx).Outputs[0].Address)
	assert.Equal(tx1.(*SendTx).Outputs[1].Address, tx2.(*SendTx).Outputs[1].Address)
	assert.Equal(tx1.(*SendTx).Outputs[2].Address, tx2.(*SendTx).Outputs[2].Address)
	assert.Equal(tx1.(*SendTx).Fee, tx2.(*SendTx).Fee)
	assert.Equal(tx1.(*SendTx).Gas, tx2.(*SendTx).Gas)

	tx1 = &ReserveFundTx{
		Fee:         NewCoins(123, 0),
		Gas:         123,
		Source:      TxInput{Address: getTestAddress("123")},
		Collateral:  NewCoins(456, 0),
		ResourceIDs: []common.Bytes{common.Bytes("789")},
		Duration:    1,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*ReserveFundTx).Fee, tx2.(*ReserveFundTx).Fee)
	assert.Equal(tx1.(*ReserveFundTx).Gas, tx2.(*ReserveFundTx).Gas)
	assert.Equal(tx1.(*ReserveFundTx).Source.Address, tx2.(*ReserveFundTx).Source.Address)
	assert.Equal(tx1.(*ReserveFundTx).Collateral, tx2.(*ReserveFundTx).Collateral)
	assert.Equal(tx1.(*ReserveFundTx).ResourceIDs, tx2.(*ReserveFundTx).ResourceIDs)
	assert.Equal(tx1.(*ReserveFundTx).Duration, tx2.(*ReserveFundTx).Duration)

	tx1 = &ReleaseFundTx{
		Fee:             NewCoins(123, 0),
		Gas:             123,
		Source:          TxInput{Address: getTestAddress("123")},
		ReserveSequence: 1,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*ReleaseFundTx).Fee, tx2.(*ReleaseFundTx).Fee)
	assert.Equal(tx1.(*ReleaseFundTx).Gas, tx2.(*ReleaseFundTx).Gas)
	assert.Equal(tx1.(*ReleaseFundTx).Source.Address, tx2.(*ReleaseFundTx).Source.Address)
	assert.Equal(tx1.(*ReleaseFundTx).ReserveSequence, tx2.(*ReleaseFundTx).ReserveSequence)

	tx1 = &ServicePaymentTx{
		Fee:             NewCoins(123, 0),
		Gas:             123,
		Source:          TxInput{Address: getTestAddress("123")},
		Target:          TxInput{Address: getTestAddress("456")},
		PaymentSequence: 1,
		ReserveSequence: 2,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*ServicePaymentTx).Fee, tx2.(*ServicePaymentTx).Fee)
	assert.Equal(tx1.(*ServicePaymentTx).Gas, tx2.(*ServicePaymentTx).Gas)
	assert.Equal(tx1.(*ServicePaymentTx).Source.Address, tx2.(*ServicePaymentTx).Source.Address)
	assert.Equal(tx1.(*ServicePaymentTx).Target.Address, tx2.(*ServicePaymentTx).Target.Address)
	assert.Equal(tx1.(*ServicePaymentTx).PaymentSequence, tx2.(*ServicePaymentTx).PaymentSequence)
	assert.Equal(tx1.(*ServicePaymentTx).ReserveSequence, tx2.(*ServicePaymentTx).ReserveSequence)

	tx1 = &SplitContractTx{
		Fee:        NewCoins(123, 0),
		Gas:        123,
		ResourceID: []byte("rid789"),
		Initiator:  TxInput{Address: getTestAddress("123")},
		Splits:     []Split{Split{Address: getTestAddress("456"), Percentage: 40}, Split{Address: getTestAddress("777"), Percentage: 20}},
		Duration:   1000,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*SplitContractTx).Fee, tx2.(*SplitContractTx).Fee)
	assert.Equal(tx1.(*SplitContractTx).Gas, tx2.(*SplitContractTx).Gas)
	assert.Equal(tx1.(*SplitContractTx).ResourceID, tx2.(*SplitContractTx).ResourceID)
	assert.Equal(tx1.(*SplitContractTx).Initiator.Address, tx2.(*SplitContractTx).Initiator.Address)
	assert.Equal(tx1.(*SplitContractTx).Splits, tx2.(*SplitContractTx).Splits)
	assert.Equal(tx1.(*SplitContractTx).Duration, tx2.(*SplitContractTx).Duration)
}

func getTestAddress(addr string) common.Address {
	var address common.Address
	copy(address[:], addr)
	return address
}
