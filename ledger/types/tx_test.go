package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thetatoken/theta/cmd/thetacli/cmd/utils"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

var chainID string = "test_chain"

func TestChainID(t *testing.T) {

	//
	// The IDs for the main chains
	//

	chainIDStrMainnet := "mainnet"
	chainIDStr := chainIDStrMainnet
	chainID := MapChainID(chainIDStr, common.HeightRPCCompatibility+1)
	assert.True(t, chainID.Cmp(big.NewInt(361)) == 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	chainIDStrTestnet := "testnet"
	chainIDStr = chainIDStrTestnet
	chainID = MapChainID(chainIDStr, common.HeightRPCCompatibility+1)
	assert.True(t, chainID.Cmp(big.NewInt(365)) == 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	chainIDStrPrivatenet := "privatenet"
	chainIDStr = chainIDStrPrivatenet
	chainID = MapChainID(chainIDStr, common.HeightRPCCompatibility+1)
	assert.True(t, chainID.Cmp(big.NewInt(366)) == 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	//
	// Invalid subchain IDs
	//

	var err error
	invalidSubchainID0 := "tsub_881" // with an extra underscore
	chainIDStr = invalidSubchainID0
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(881)) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	invalidSubchainID1 := "ts881"
	chainIDStr = invalidSubchainID1
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(881)) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	invalidSubchainID2 := "tsub09998" // leading digit should not be 0
	chainIDStr = invalidSubchainID2
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(9998)) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	invalidSubchainID3 := "tsubabc9" // hex not allowed in chainID
	chainIDStr = invalidSubchainID3
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(43977)) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	invalidSubchainID4 := "tsub999" // subchain ID needs to be at least 1000
	chainIDStr = invalidSubchainID4
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(999)) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	invalidSubchainID5 := "tsub34535873957238957239573985728957283957923528357238572893572983457238957238495893" // subchain ID needs to be smaller than uint64.max
	chainIDStr = invalidSubchainID5
	chainID, err = extractSubchainID(chainIDStr)
	assert.True(t, err != nil, "should be an invalid subchain ID: %v", chainIDStr)
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	cid, _ := big.NewInt(0).SetString("34535873957238957239573985728957283957923528357238572893572983457238957238495893", 10)
	assert.True(t, chainID.Cmp(cid) != 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	//
	// Valid subchain IDs
	//

	validSubchainID1 := "tsub1991"
	chainIDStr = validSubchainID1
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(1991)) == 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	validSubchainID2 := "tsub4546325235"
	chainIDStr = validSubchainID2
	chainID = MapChainID(chainIDStr, common.HeightEnableMetachainSupport+1)
	assert.True(t, chainID.Cmp(big.NewInt(4546325235)) == 0, "mapped chainID for %v is %v", chainIDStr, chainID)
	fmt.Printf("extracted chainID for %v: %v\n", chainIDStr, chainID)

	// assert.True(t, false)
}

func TestCoinbaseTxSignable(t *testing.T) {
	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	coinbaseTx := &CoinbaseTx{
		Proposer: NewTxInput(va1PrivAcc.Address, NewCoins(0, 0), 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{ThetaWei: big.NewInt(333), TFuelWei: big.NewInt(0)},
			},
			TxOutput{
				Address: getTestAddress("validator1"),
				Coins:   Coins{ThetaWei: big.NewInt(444), TFuelWei: big.NewInt(0)},
			},
		},
		BlockHeight: 10,
	}
	signBytes := coinbaseTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F87F80808094000000000000000000000000000000000000000080B8648D746573745F636861696E5F696480F853DA94B23369B1225E72332462A75C1B7F509A805E3D6EC280800180F6DA9476616C696461746F723100000000000000000000C482014D80DA9476616C696461746F723100000000000000000000C48201BC800A"

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
		Proposer: NewTxInput(va1PrivAcc.Address, NewCoins(0, 0), 1),
		Outputs: []TxOutput{
			TxOutput{
				Address: va2PrivAcc.PrivKey.PublicKey().Address(),
				Coins:   Coins{ThetaWei: big.NewInt(8), TFuelWei: big.NewInt(0)},
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
		Proposer:        NewTxInput(va1PrivAcc.Address, NewCoins(0, 0), 1),
		SlashedAddress:  getTestAddress("014FAB"),
		ReserveSequence: 1,
		SlashProof:      []byte("2345ABC"),
	}
	signBytes := slashTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F86280808094000000000000000000000000000000000000000080B8478A746573745F636861696E01F839DA94B23369B1225E72332462A75C1B7F509A805E3D6EC280800180943031344641420000000000000000000000000000018732333435414243"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for CoinbaseTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSlashTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	va1PrivAcc := PrivAccountFromSecret("validator1")

	// Construct a SlashTx signature
	tx := &SlashTx{
		Proposer:        NewTxInput(va1PrivAcc.Address, Coins{}, 1),
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
		Fee: Coins{ThetaWei: big.NewInt(111), TFuelWei: big.NewInt(0)},
		Inputs: []TxInput{
			TxInput{
				Address:  getTestAddress("input1"),
				Coins:    Coins{ThetaWei: big.NewInt(12345)},
				Sequence: 67890,
			},
			TxInput{
				Address:  getTestAddress("input2"),
				Coins:    Coins{ThetaWei: big.NewInt(111), TFuelWei: big.NewInt(0)},
				Sequence: 222,
			},
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: getTestAddress("output1"),
				Coins:   Coins{ThetaWei: big.NewInt(333), TFuelWei: big.NewInt(0)},
			},
			TxOutput{
				Address: getTestAddress("output2"),
				Coins:   Coins{ThetaWei: big.NewInt(444), TFuelWei: big.NewInt(0)},
			},
		},
	}
	signBytes := sendTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F8A180808094000000000000000000000000000000000000000080B8868A746573745F636861696E02F878C26F80F83CDF94696E707574310000000000000000000000000000C4823039808301093280DB94696E707574320000000000000000000000000000C26F8081DE80F6DA946F75747075743100000000000000000000000000C482014D80DA946F75747075743200000000000000000000000000C48201BC80"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SendTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestSendTxSignable2(t *testing.T) {
	chainID := "privatenet"
	ten18 := new(big.Int).SetUint64(1000000000000000000) // 10^18
	thetaWei := new(big.Int).Mul(new(big.Int).SetUint64(10), ten18)
	tfuelWei := new(big.Int).Mul(new(big.Int).SetUint64(20), ten18)
	feeInTFuelWei := new(big.Int).SetUint64(1000000000000) // 10^12

	senderAddr := common.HexToAddress("2E833968E5bB786Ae419c4d13189fB081Cc43bab")
	receiverAddr := common.HexToAddress("9F1233798E905E173560071255140b4A8aBd3Ec6")
	sendTx := &SendTx{
		Fee: Coins{ThetaWei: big.NewInt(0), TFuelWei: feeInTFuelWei},
		Inputs: []TxInput{
			TxInput{
				Address:  senderAddr,
				Coins:    Coins{ThetaWei: thetaWei, TFuelWei: new(big.Int).Add(tfuelWei, feeInTFuelWei)},
				Sequence: 2,
			},
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: receiverAddr,
				Coins:   Coins{ThetaWei: thetaWei, TFuelWei: tfuelWei},
			},
		},
	}
	signBytes := sendTx.SignBytes(chainID)
	signBytesHex := hex.EncodeToString(signBytes)
	expected := "f88980808094000000000000000000000000000000000000000080b86e8a707269766174656e657402f860c78085e8d4a51000eceb942e833968e5bb786ae419c4d13189fb081cc43babd3888ac7230489e800008901158e46f1e87510000280eae9949f1233798e905e173560071255140b4a8abd3ec6d3888ac7230489e800008901158e460913d00000"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for SendTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)

	t.Logf("Tx SignBytes            : %v", signBytesHex)

	feeEncoded, _ := rlp.EncodeToBytes(sendTx.Fee)
	t.Logf("sendTx.Fee              : %v", hex.EncodeToString(feeEncoded))

	inputsEncoded, _ := rlp.EncodeToBytes(sendTx.Inputs)
	t.Logf("sendTx.Inputs           : %v", hex.EncodeToString(inputsEncoded))

	inputs0Encoded, _ := rlp.EncodeToBytes(sendTx.Inputs[0])
	t.Logf("sendTx.Inputs[0]        : %v", hex.EncodeToString(inputs0Encoded))

	inputs0CoinsEncoded, _ := rlp.EncodeToBytes(sendTx.Inputs[0].Coins)
	t.Logf("sendTx.Inputs[0].Coins  : %v", hex.EncodeToString(inputs0CoinsEncoded))

	inputs0AddrEncoded, _ := rlp.EncodeToBytes(sendTx.Inputs[0].Address)
	t.Logf("sendTx.Inputs[0].Addr   : %v", hex.EncodeToString(inputs0AddrEncoded))

	outputsEncoded, _ := rlp.EncodeToBytes(sendTx.Outputs)
	t.Logf("sendTx.Outputs          : %v", hex.EncodeToString(outputsEncoded))

	outputs0Encoded, _ := rlp.EncodeToBytes(sendTx.Outputs[0])
	t.Logf("sendTx.Outputs[0]       : %v", hex.EncodeToString(outputs0Encoded))

	outputs0CoinsEncoded, _ := rlp.EncodeToBytes(sendTx.Outputs[0].Coins)
	t.Logf("sendTx.Outputs[0].Coins : %v", hex.EncodeToString(outputs0CoinsEncoded))

	senderSkBytes, _ := hex.DecodeString("93a90ea508331dfdf27fb79757d4250b4e84954927ba0073cd67454ac432c737")
	senderPrivKey, _ := crypto.PrivateKeyFromBytes(senderSkBytes)
	senderSignature, _ := senderPrivKey.Sign(signBytes)

	signBytesHash := crypto.Keccak256(signBytes)
	t.Logf("signBytesHash : %v", hex.EncodeToString(signBytesHash))

	sendTx.SetSignature(senderAddr, senderSignature)

	raw, err := TxToBytes(sendTx)
	if err != nil {
		utils.Error("Failed to encode transaction: %v\n", err)
	}
	t.Logf("sendTx.Inputs[0].Signature : %v", hex.EncodeToString(senderSignature.ToBytes()))

	signedTxBytesHex := hex.EncodeToString(raw)
	t.Logf("Signed Tx: %v", signedTxBytesHex)

	expectedSignedTxBytes := "02f8a4c78085e8d4a51000f86ff86d942e833968e5bb786ae419c4d13189fb081cc43babd3888ac7230489e800008901158e46f1e875100002b8415a6e9a2e93487c786f07175998493161e61a5d9613745aa0e2fe51e5db1eaf626f72bfae41d971e88ff3b2c217cf611c2addb266e7d7ebda29cb0e9e5a2f482800eae9949f1233798e905e173560071255140b4a8abd3ec6d3888ac7230489e800008901158e460913d00000"
	assert.Equal(t, expectedSignedTxBytes, signedTxBytesHex,
		"Got unexpected signed raw bytes for SendTx. Expected:\n%v\nGot:\n%v", expectedSignedTxBytes, signedTxBytesHex)

}

func TestSendTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("sendtx1")
	test2PrivAcc := PrivAccountFromSecret("sendtx2")

	// Construct a SendTx signature
	tx := &SendTx{
		Fee: Coins{TFuelWei: big.NewInt(2)},
		Inputs: []TxInput{
			NewTxInput(test1PrivAcc.Address, Coins{ThetaWei: big.NewInt(0), TFuelWei: big.NewInt(10)}, 1),
		},
		Outputs: []TxOutput{
			TxOutput{
				Address: test2PrivAcc.Address,
				Coins:   Coins{ThetaWei: big.NewInt(0), TFuelWei: big.NewInt(8)},
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
		Fee: Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{ThetaWei: Zero, TFuelWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Collateral:  Coins{ThetaWei: Zero, TFuelWei: big.NewInt(22897)},
		ResourceIDs: []string{"rid00123"},
		Duration:    uint64(999),
	}

	signBytes := reserveFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F85D80808094000000000000000000000000000000000000000080B8428A746573745F636861696E03F5C2806FDF94696E707574310000000000000000000000000000C4808230398301093280C480825971C98872696430303132338203E7"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReserveFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReserveFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("reservefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReserveFundTx{
		Fee:         Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source:      NewTxInput(test1PrivAcc.Address, Coins{ThetaWei: Zero, TFuelWei: big.NewInt(10)}, 1),
		Collateral:  Coins{ThetaWei: Zero, TFuelWei: big.NewInt(22897)},
		ResourceIDs: []string{"rid00123"},
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
		Fee: Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("input1"),
			Coins:    Coins{ThetaWei: Zero, TFuelWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		ReserveSequence: 12,
	}

	signBytes := releaseFundTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F84B80808094000000000000000000000000000000000000000080B18A746573745F636861696E04E4C2806FDF94696E707574310000000000000000000000000000C48082303983010932800C"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ReleaseFundTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestReleaseFundTxProto(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	chainID := "test_chain_id"
	test1PrivAcc := PrivAccountFromSecret("releasefundtx")

	// Construct a ReserveFundTx transaction
	tx := &ReleaseFundTx{
		Fee:             Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source:          NewTxInput(test1PrivAcc.Address, Coins{ThetaWei: Zero, TFuelWei: big.NewInt(10)}, 1),
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
		Fee: Coins{TFuelWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, TFuelWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    NewCoins(0, 0),
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      "rid00123",
	}

	signBytes := servicePaymentTx.SourceSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F86F80808094000000000000000000000000000000000000000080B8548A746573745F636861696E05F846C28080DC94736F757263650000000000000000000000000000C4808230398080DA947461726765740000000000000000000000000000C280808080030C887269643030313233"

	assert.Equal(t, expected, signBytesHex,
		"Got unexpected sign string for ServicePaymentTx. Expected:\n%v\nGot:\n%v", expected, signBytesHex)
}

func TestServicePaymentTxTargetSignable(t *testing.T) {
	servicePaymentTx := &ServicePaymentTx{
		Fee: Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, TFuelWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Target: TxInput{
			Address:  getTestAddress("target"),
			Coins:    NewCoins(0, 0),
			Sequence: 22341,
		},
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      "rid00123",
	}

	signBytes := servicePaymentTx.TargetSignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F87480808094000000000000000000000000000000000000000080B8598A746573745F636861696E05F84BC2806FDF94736F757263650000000000000000000000000000C4808230398301093280DC947461726765740000000000000000000000000000C2808082574580030C887269643030313233"

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
		Fee:             Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		Source:          NewTxInput(sourcePrivAcc.Address, Coins{ThetaWei: Zero, TFuelWei: big.NewInt(10000)}, 1),
		Target:          NewTxInput(targetPrivAcc.Address, NewCoins(0, 0), 1),
		PaymentSequence: 3,
		ReserveSequence: 12,
		ResourceID:      "rid00123",
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
		Fee:        Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		ResourceID: "rid00123",
		Initiator: TxInput{
			Address:  getTestAddress("source"),
			Coins:    Coins{ThetaWei: Zero, TFuelWei: big.NewInt(12345)},
			Sequence: 67890,
		},
		Splits:   []Split{split},
		Duration: 99,
	}

	signBytes := splitRuleTx.SignBytes(chainID)
	signBytesHex := fmt.Sprintf("%X", signBytes)
	expected := "F86E80808094000000000000000000000000000000000000000080B8538A746573745F636861696E06F845C2806F887269643030313233DF94736F757263650000000000000000000000000000C4808230398301093280D7D69473706C69746164647231000000000000000000001E63"

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
		Fee:        Coins{ThetaWei: Zero, TFuelWei: big.NewInt(111)},
		ResourceID: "rid00123",
		Initiator:  NewTxInput(test1PrivAcc.Address, Coins{ThetaWei: Zero, TFuelWei: big.NewInt(10)}, 1),
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

func TestCoinbaseTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := CoinbaseTx{
		BlockHeight: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d CoinbaseTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.BlockHeight)
}

func TestSlashTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := SlashTx{
		ReserveSequence: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d SlashTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.ReserveSequence)
}

func TestReserveFundTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := ReserveFundTx{
		Duration: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d ReserveFundTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.Duration)
}

func TestReleaseFundTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := ReleaseFundTx{
		ReserveSequence: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d ReleaseFundTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.ReserveSequence)
}

func TestServicePaymentTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := ServicePaymentTx{
		ReserveSequence: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d ServicePaymentTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.ReserveSequence)
}

func TestSplitRuleTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	a := SplitRuleTx{
		Duration: math.MaxUint64,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d SplitRuleTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.Duration)
}

func TestSmartContractTxJSON(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	gasPrice, _ := new(big.Int).SetString("12312312312312312312331231231231212312312312312313213", 10)
	a := SmartContractTx{
		GasLimit: math.MaxUint64,
		GasPrice: gasPrice,
	}
	s, err := json.Marshal(a)
	require.Nil(err)

	var d SmartContractTx
	err = json.Unmarshal(s, &d)
	require.Nil(err)
	assert.Equal(uint64(math.MaxUint64), d.GasLimit)
	assert.Equal(0, gasPrice.Cmp(d.GasPrice))
}
