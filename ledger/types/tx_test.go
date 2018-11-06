package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/core"
)

var chainID string = "test_chain"

func TestCoinbaseTxSignable(t *testing.T) {
	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	coinbaseTx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.PrivKey.PublicKey(), NewCoins(0, 0), 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{ThetaWei: big.NewInt(333), GammaWei: big.NewInt(0)},
			},
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{ThetaWei: big.NewInt(444), GammaWei: big.NewInt(0)},
			},
		},
		BlockHeight: 10,
	}
	signBytes := coinbaseTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8D746573745F636861696E5F696480F897F85D94B23369B1225E72332462A75C1B7F509A805E3D6EC280800180B84104F584D8724624D250A1EA8FB24BF4D934EE55E53A1D03C9386E630033079484E215E86145F4CD5BD943A75792F28643BCB1ABB37A29C6F8CF2A1EA1615A3CA9C4F6DA9476616C696461746F723100000000000000000000C482014D80DA9476616C696461746F723100000000000000000000C48201BC800A"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for CoinbaseTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestCoinbaseTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")
	va2PrivAcc := PrivAccountFromSecret("validator2")

	// Construct a CoinbaseTx signature
	tx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.PrivKey.PublicKey(), NewCoins(0, 0), 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: va2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{ThetaWei: big.NewInt(8), GammaWei: big.NewInt(0)},
			},
		},
		BlockHeight: 10,
	}
	tx.Proposer.Signature = va1PrivAcc.Sign(tx.SignBytes(chainID))

	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*CoinbaseTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)

	fmt.Printf(">>>>> tx : %v\n", tx)
	fmt.Printf(">>>>> tx2: %v\n", tx2)

	fmt.Printf(">>>>> signBytes : %v\n", hex.EncodeToString(signBytes))
	fmt.Printf(">>>>> signBytes2: %v\n", hex.EncodeToString(signBytes2))

	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := va1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	assert.Equal(tx, tx2)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*CoinbaseTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}

/*
func TestCoinbaseTxRLP(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")
	va2PrivAcc := PrivAccountFromSecret("validator2")

	// Construct a CoinbaseTx signature
	tx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{{"", 0}}, 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: va2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{{"foo", 8}},
			},
		},
		BlockHeight: 10,
	}
	tx.Proposer.Signature = va1PrivAcc.Sign(tx.SignBytes(chainID))

	b, err := rlp.EncodeToBytes(tx)
	require.Nil(err)

	var txs Tx
	err = rlp.DecodeBytes(b, &txs)
	require.Nil(err, &txs)
	tx2 := txs.(*CoinbaseTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)

	fmt.Printf(">>>>> tx : %v\n", tx)
	fmt.Printf(">>>>> tx2: %v\n", tx2)

	fmt.Printf(">>>>> signBytes : %v\n", hex.EncodeToString(signBytes))
	fmt.Printf(">>>>> signBytes2: %v\n", hex.EncodeToString(signBytes2))

	//assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// // sign this thing
	// sig := va1PrivAcc.Sign(signBytes)
	// // we handle both raw sig and wrapped sig the same
	// tx.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	// tx2.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	// assert.Equal(tx, &tx2)

	// // let's marshal / unmarshal this with signature
	// js, err = json.Marshal(tx)
	// require.Nil(err)
	// // fmt.Println(string(js))
	// err = json.Unmarshal(js, &tx2)
	// require.Nil(err)

	// // and make sure the sig is preserved
	// assert.Equal(tx, &tx2)
	// assert.False(tx2.Proposer.Signature.IsEmpty())
}
*/

func TestSlashTxSignable(t *testing.T) {
	va1PrivAcc := PrivAccountFromSecret("validator1")
	slashTx := &SlashTx{
		Proposer:        NewTxInput(va1PrivAcc.PrivKey.PublicKey(), NewCoins(0, 0), 1),
		SlashedAddress:  getTestAddress("014FAB"),
		ReserveSequence: 1,
		SlashProof:      []byte("2345ABC"),
	}
	signBytes := slashTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E01F87DF85D94B23369B1225E72332462A75C1B7F509A805E3D6EC280800180B84104F584D8724624D250A1EA8FB24BF4D934EE55E53A1D03C9386E630033079484E215E86145F4CD5BD943A75792F28643BCB1ABB37A29C6F8CF2A1EA1615A3CA9C4943031344641420000000000000000000000000000018732333435414243"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for CoinbaseTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSlashTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	// Construct a SlashTx signature
	tx := &SlashTx{
		Proposer:        NewTxInput(va1PrivAcc.PrivKey.PublicKey(), Coins{}, 1),
		SlashedAddress:  getTestAddress("014FAB"),
		ReserveSequence: 1,
		SlashProof:      []byte("2345ABC"),
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*SlashTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := va1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(va1PrivAcc.PrivKey.PublicKey().Address(), sig)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*SlashTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Proposer.Signature, tx2.Proposer.Signature)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}

func TestSendTxSignable(t *testing.T) {
	sendTx := &SendTx{
		Fee: Coins{ThetaWei: big.NewInt(111), GammaWei: big.NewInt(0)},
		Inputs: []TxInput{
			TxInput{
				Address:  getTestAddress("input1"),
				Coins:    Coins{ThetaWei: big.NewInt(12345)},
				Sequence: 67890,
			},
			TxInput{
				Address:  getTestAddress("input2"),
				Coins:    Coins{ThetaWei: big.NewInt(111), GammaWei: big.NewInt(0)},
				Sequence: 222,
			},
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("output1"),
				Coins:   Coins{ThetaWei: big.NewInt(333), GammaWei: big.NewInt(0)},
			},
			TxOutput{
				Address: getTestAddress("output2"),
				Coins:   Coins{ThetaWei: big.NewInt(444), GammaWei: big.NewInt(0)},
			},
		},
	}
	signBytes := sendTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E02F87AC26F80F83EE094696E707574310000000000000000000000000000C482303980830109328080DC94696E707574320000000000000000000000000000C26F8081DE8080F6DA946F75747075743100000000000000000000000000C482014D80DA946F75747075743200000000000000000000000000C48201BC80"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SendTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSendTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("sendtx1")
	test2PrivAcc := PrivAccountFromSecret("sendtx2")

	// Construct a SendTx signature
	tx := &SendTx{
		Fee: Coins{GammaWei: big.NewInt(2)},
		Inputs: []TxInput{
			NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: big.NewInt(0), GammaWei: big.NewInt(10)}, 1),
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: test2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{ThetaWei: big.NewInt(0), GammaWei: big.NewInt(8)},
			},
		},
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*SendTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*SendTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Inputs[0].Signature, tx2.Inputs[0].Signature)
	assert.False(tx2.Inputs[0].Signature.IsEmpty())
}

func TestReserveFundTxSignable(t *testing.T) {
	reserveFundTx := &ReserveFundTx{
		Fee: Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Collateral:  Coins{ThetaWei: Zero, GammaWei: big.NewInt(22897)},
		ResourceIDs: []common.Bytes{common.Bytes("rid00123")},
		Duration:    uint64(999),
	}

	signBytes := reserveFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E03F6C2806FE094696E707574310000000000000000000000000000C480823039830109328080C480825971C98872696430303132338203E7"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReserveFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReserveFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("reservefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReserveFundTx{
		Fee:         Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source:      NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10)}, 1),
		Collateral:  Coins{ThetaWei: Zero, GammaWei: big.NewInt(22897)},
		ResourceIDs: []common.Bytes{common.Bytes("rid00123")},
		Duration:    uint64(999),
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*ReserveFundTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*ReserveFundTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Source.Signature, tx2.Source.Signature)
	assert.False(tx2.Source.Signature.IsEmpty())
}

func TestReleaseFundTxSignable(t *testing.T) {
	releaseFundTx := &ReleaseFundTx{
		Fee: Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		ReserveSequence: 12,
	}

	signBytes := releaseFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E04E5C2806FE094696E707574310000000000000000000000000000C4808230398301093280800C"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReleaseFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReleaseFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("releasefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReleaseFundTx{
		Fee:             Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source:          NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10)}, 1),
		ReserveSequence: 1,
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*ReleaseFundTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*ReleaseFundTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Source.Signature, tx2.Source.Signature)
	assert.False(tx2.Source.Signature.IsEmpty())
}

func TestServicePaymentTxSourceSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Fee: Coins{GammaWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    NewCoins(0, 0),
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      []byte("rid00123"),
	}

	signBytes := servicePaymentTx.SourceSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E05F848C28080DD94736F757263650000000000000000000000000000C480823039808080DB947461726765740000000000000000000000000000C28080808080030C887269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxTargetSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Fee: Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    NewCoins(0, 0),
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      []byte("rid00123"),
	}

	signBytes := servicePaymentTx.TargetSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E05F84DC2806FE094736F757263650000000000000000000000000000C480823039830109328080DD947461726765740000000000000000000000000000C280808257458080030C887269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	sourcePrivAcc := PrivAccountFromSecret("servicepaymenttxsource")
	targetPrivAcc := PrivAccountFromSecret("servicepaymenttxtarget")

	// Construct a ReserveFundTx signature
	tx := &ServicePaymentTx{
		Fee:             Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source:          NewTxInput(sourcePrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10000)}, 1),
		Target:          NewTxInput(targetPrivAcc.PrivKey.PublicKey(), NewCoins(0, 0), 1),
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      []byte("rid00123"),
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*ServicePaymentTx)

	// make sure they are the same!
	sourceSignBytes := tx.SourceSignBytes(chainID)
	sourceSignBytes2 := tx2.SourceSignBytes(chainID)
	assert.Equal(sourceSignBytes, sourceSignBytes2)

	targetSignBytes := tx.TargetSignBytes(chainID)
	targetSignBytes2 := tx2.TargetSignBytes(chainID)
	assert.Equal(targetSignBytes, targetSignBytes2)
}

func TestSplitRuleTxSignable(t *testing.T) {
	split := Split{
		Address:    getTestAddress("splitaddr1"),
		Percentage: 30,
	}
	splitRuleTx := &SplitRuleTx{
		Fee:        Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		ResourceID: []byte("rid00123"),
		Initiator: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Splits:   []Split{split},
		Duration: 99,
	}

	signBytes := splitRuleTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E06F846C2806F887269643030313233E094736F757263650000000000000000000000000000C480823039830109328080D7D69473706C69746164647231000000000000000000001E63"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SplitRuleTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSplitRuleTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("splitruletx")

	// Construct a SplitRuleTx signature
	split := Split{
		Address:    getTestAddress("splitaddr1"),
		Percentage: 30,
	}
	tx := &SplitRuleTx{
		Fee:        Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		ResourceID: []byte("rid00123"),
		Initiator:  NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10)}, 1),
		Splits:     []Split{split},
		Duration:   99,
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*SplitRuleTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*SplitRuleTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Initiator.Signature, tx2.Initiator.Signature)
	assert.False(tx2.Initiator.Signature.IsEmpty())
}

func TestUpdateValidatorsTxSignable(t *testing.T) {
	updateValidatorsTx := &UpdateValidatorsTx{
		Validators: []*core.Validator{},
		Proposer: TxInput{
			Address:  getTestAddress("validator1"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
	}

	signBytes := updateValidatorsTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for UpdateValidatorsTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestUpdateValidatorsTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("updatevalidatorstx")

	// Construct a UpdateValidatorsTx signature
	// idBytes, err := hex.DecodeString("id123")
	// if err != nil {
	// 	panic(fmt.Sprintf("Unable to decode public key: %v", id))
	// }
	// va := core.NewValidator(idBytes, uint64(100))
	// tx := &UpdateValidatorsTx{
	// 	Validators: []*core.Validator{&va},
	// 	Proposer:   NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{{"", 10}}, 1),
	// }

	tx := &UpdateValidatorsTx{
		Proposer: NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10)}, 1),
	}

	// serialize this and back
	b, err := TxToBytes(tx)
	require.Nil(err)
	txs, err := TxFromBytes(b)
	require.Nil(err)
	tx2 := txs.(*UpdateValidatorsTx)

	fmt.Printf(">>> tx.Validators:  %v\n", tx.Validators)
	fmt.Printf(">>> tx2.Validators: %v\n", tx2.Validators)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	// let's marshal / unmarshal this with signature
	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*UpdateValidatorsTx)

	// and make sure the sig is preserved
	assert.Equal(tx.Proposer.Signature, tx2.Proposer.Signature)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}
