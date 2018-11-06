package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
)

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

	tx1 = &ReserveFundTx{
		Fee:         NewCoins(123, 0),
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
	assert.Equal(tx1.(*ReserveFundTx).Source.Address, tx2.(*ReserveFundTx).Source.Address)
	assert.Equal(tx1.(*ReserveFundTx).Collateral, tx2.(*ReserveFundTx).Collateral)
	assert.Equal(tx1.(*ReserveFundTx).ResourceIDs, tx2.(*ReserveFundTx).ResourceIDs)
	assert.Equal(tx1.(*ReserveFundTx).Duration, tx2.(*ReserveFundTx).Duration)

	tx1 = &ReleaseFundTx{
		Fee:             NewCoins(123, 0),
		Source:          TxInput{Address: getTestAddress("123")},
		ReserveSequence: 1,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*ReleaseFundTx).Fee, tx2.(*ReleaseFundTx).Fee)
	assert.Equal(tx1.(*ReleaseFundTx).Source.Address, tx2.(*ReleaseFundTx).Source.Address)
	assert.Equal(tx1.(*ReleaseFundTx).ReserveSequence, tx2.(*ReleaseFundTx).ReserveSequence)

	tx1 = &ServicePaymentTx{
		Fee:             NewCoins(123, 0),
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
	assert.Equal(tx1.(*ServicePaymentTx).Source.Address, tx2.(*ServicePaymentTx).Source.Address)
	assert.Equal(tx1.(*ServicePaymentTx).Target.Address, tx2.(*ServicePaymentTx).Target.Address)
	assert.Equal(tx1.(*ServicePaymentTx).PaymentSequence, tx2.(*ServicePaymentTx).PaymentSequence)
	assert.Equal(tx1.(*ServicePaymentTx).ReserveSequence, tx2.(*ServicePaymentTx).ReserveSequence)

	tx1 = &SplitRuleTx{
		Fee:        NewCoins(123, 0),
		ResourceID: []byte("rid789"),
		Initiator:  TxInput{Address: getTestAddress("123")},
		Splits:     []Split{Split{Address: getTestAddress("456"), Percentage: 40}, Split{Address: getTestAddress("777"), Percentage: 20}},
		Duration:   1000,
	}
	b, err = TxToBytes(tx1)
	require.Nil(err)
	tx2, err = TxFromBytes(b)
	require.Nil(err)
	assert.Equal(tx1.(*SplitRuleTx).Fee, tx2.(*SplitRuleTx).Fee)
	assert.Equal(tx1.(*SplitRuleTx).ResourceID, tx2.(*SplitRuleTx).ResourceID)
	assert.Equal(tx1.(*SplitRuleTx).Initiator.Address, tx2.(*SplitRuleTx).Initiator.Address)
	assert.Equal(tx1.(*SplitRuleTx).Splits, tx2.(*SplitRuleTx).Splits)
	assert.Equal(tx1.(*SplitRuleTx).Duration, tx2.(*SplitRuleTx).Duration)
}

func getTestAddress(addr string) common.Address {
	var address common.Address
	copy(address[:], addr)
	return address
}
