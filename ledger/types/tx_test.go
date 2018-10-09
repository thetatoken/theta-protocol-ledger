package types

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	expected := "8D746573745F636861696E5F69640A9F010A610A14B23369B1225E72332462A75C1B7F509A805E3D6E1200180122002A43124104F584D8724624D250A1EA8FB24BF4D934EE55E53A1D03C9386E630033079484E215E86145F4CD5BD943A75792F28643BCB1ABB37A29C6F8CF2A1EA1615A3CA9C4121B0A1476616C696461746F723100000000000000000000120310CD02121B0A1476616C696461746F723100000000000000000000120310BC03180A"

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
	expected := "8A746573745F636861696E3284010A610A14B23369B1225E72332462A75C1B7F509A805E3D6E1200180122002A43124104F584D8724624D250A1EA8FB24BF4D934EE55E53A1D03C9386E630033079484E215E86145F4CD5BD943A75792F28643BCB1ABB37A29C6F8CF2A1EA1615A3CA9C4121430313446414200000000000000000000000000001801220732333435414243"

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
	tx2 = txs.(*SlashTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}

func TestSendTxSignable(t *testing.T) {
	sendTx := &SendTx{
		Gas: 222,
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
	expected := "8A746573745F636861696E12890108DE011202106F1A230A14696E707574310000000000000000000000000000120310B96018B2920422002A001A210A14696E7075743200000000000000000000000000001202106F18DE0122002A00221B0A146F75747075743100000000000000000000000000120310CD02221B0A146F75747075743200000000000000000000000000120310BC03"

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
		Gas: 1,
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
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	assert.Equal(tx, tx2)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*SendTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Inputs[0].Signature.IsEmpty())
}

func TestReserveFundTxSignable(t *testing.T) {
	reserveFundTx := &ReserveFundTx{
		Gas: 222,
		Fee: Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{ThetaWei: Zero, GammaWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Collateral:  Coins{ThetaWei: Zero, GammaWei: big.NewInt(22897)},
		ResourceIDs: [][]byte{[]byte("rid00123")},
		Duration:    uint64(999),
	}

	signBytes := reserveFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E1A3F08DE011202106F1A230A14696E707574310000000000000000000000000000120310B96018B2920422002A00220410F1B2012A08726964303031323330E707"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReserveFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReserveFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("reservefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReserveFundTx{
		Gas:         222,
		Fee:         Coins{ThetaWei: Zero, GammaWei: big.NewInt(111)},
		Source:      NewTxInput(test1PrivAcc.PrivKey.PublicKey(), Coins{ThetaWei: Zero, GammaWei: big.NewInt(10)}, 1),
		Collateral:  Coins{ThetaWei: Zero, GammaWei: big.NewInt(22897)},
		ResourceIDs: [][]byte{[]byte("rid00123")},
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
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*ReserveFundTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Source.Signature.IsEmpty())
}

func TestReleaseFundTxSignable(t *testing.T) {
	releaseFundTx := &ReleaseFundTx{
		Gas: 222,
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
	expected := "8A746573745F636861696E222E08DE011202106F1A230A14696E707574310000000000000000000000000000120310B96018B2920422002A00200C"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReleaseFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReleaseFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("releasefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReleaseFundTx{
		Gas:             222,
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
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*ReleaseFundTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Source.Signature.IsEmpty())
}

func TestServicePaymentTxSourceSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Gas: 222,
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
	expected := "8A746573745F636861696E2A4D12001A1F0A14736F757263650000000000000000000000000000120310B96022002A00221A0A14746172676574000000000000000000000000000022002A002803300C3A087269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxTargetSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Gas: 222,
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
	expected := "8A746573745F636861696E2A5C08DE011202106F1A230A14736F757263650000000000000000000000000000120310B96018B2920422002A0022200A147461726765740000000000000000000000000000120018C5AE0122002A002803300C3A087269643030313233"

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
		Gas:             222,
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

func TestSplitContractTxSignable(t *testing.T) {
	split := Split{
		Address:    getTestAddress("splitaddr1"),
		Percentage: 30,
	}
	splitContractTx := &SplitContractTx{
		Gas:        222,
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

	signBytes := splitContractTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "8A746573745F636861696E3A5208DE011202106F1A08726964303031323322230A14736F757263650000000000000000000000000000120310B96018B2920422002A002A180A1473706C6974616464723100000000000000000000101E3063"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SplitContractTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSplitContractTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("splitcontracttx")

	// Construct a SplitContractTx signature
	split := Split{
		Address:    getTestAddress("splitaddr1"),
		Percentage: 30,
	}
	tx := &SplitContractTx{
		Gas:        222,
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
	tx2 := txs.(*SplitContractTx)

	// make sure they are the same!
	signBytes := tx.SignBytes(chainID)
	signBytes2 := tx2.SignBytes(chainID)
	assert.Equal(signBytes, signBytes2)
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*SplitContractTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
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
	assert.Equal(tx, tx2)

	// sign this thing
	sig := test1PrivAcc.Sign(signBytes)
	// we handle both raw sig and wrapped sig the same
	tx.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)
	tx2.SetSignature(test1PrivAcc.PrivKey.PublicKey().Address(), sig)

	assert.Equal(tx, tx2)

	// let's marshal / unmarshal this with signature
	b, err = TxToBytes(tx)
	require.Nil(err)
	txs, err = TxFromBytes(b)
	require.Nil(err)
	tx2 = txs.(*UpdateValidatorsTx)

	// and make sure the sig is preserved
	assert.Equal(tx, tx2)
	assert.False(tx2.Proposer.Signature.IsEmpty())
}
