package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

/*
Tx (Transaction) is an atomic operation on the ledger state.

Transaction Types:
 - CoinbaseTx           Coinbase transaction for block rewards
 - SlashTx     			Transaction for slashing dishonest user
 - SendTx               Send coins to address
 - ReserveFundTx        Reserve fund for subsequence service payments
 - ReleaseFundTx        Release fund reserved for service payments
 - ServicePaymentTx     Payments for service
 - SplitRuleTx          Payment split rule
 - DepositStakeTx       Deposit stake to a target address (e.g. a validator)
 - WithdrawStakeTx      Withdraw stake from a target address (e.g. a validator)
 - SmartContractTx      Execute smart contract
*/

// Gas of regular transactions
const (
	GasSendTxPerAccount   uint64 = 5000
	GasReserveFundTx      uint64 = 10000
	GasReleaseFundTx      uint64 = 10000
	GasServicePaymentTx   uint64 = 10000
	GasSplitRuleTx        uint64 = 10000
	GasUpdateValidatorsTx uint64 = 10000
	GasDepositStakeTx     uint64 = 10000
	GasWidthdrawStakeTx   uint64 = 10000
)

type Tx interface {
	AssertIsTx()
	SignBytes(chainID string) []byte
}

//-----------------------------------------------------------------------------

func TxID(chainID string, tx Tx) common.Hash {
	var signBytes []byte
	switch tx.(type) {
	default:
		signBytes = tx.SignBytes(chainID)
	case *ServicePaymentTx:
		spTx := tx.(*ServicePaymentTx)
		signBytes = spTx.TargetSignBytes(chainID)
	}
	return crypto.Keccak256Hash(signBytes)
}

//--------------------------------------------------------------------------------

// Contract: This function is deterministic and completely reversible.
func jsonEscape(str string) string {
	escapedBytes, err := json.Marshal(str)
	if err != nil {
		log.Panicf("Error json-escaping a string: %v", str)
	}
	return string(escapedBytes)
}

func encodeToBytes(str string) []byte {
	encodedBytes, err := rlp.EncodeToBytes(str)
	if err != nil {
		log.Panicf("Failed to encode %v: %v", str, err)
	}
	return encodedBytes
}

//-----------------------------------------------------------------------------

type TxInput struct {
	Address   common.Address // Hash of the PubKey
	Coins     Coins
	Sequence  uint64            // Must be 1 greater than the last committed TxInput
	Signature *crypto.Signature // Depends on the PubKey type and the whole Tx
}

type TxInputJSON struct {
	Address   common.Address    `json:"address"`   // Hash of the PubKey
	Coins     Coins             `json:"coins"`     //
	Sequence  common.JSONUint64 `json:"sequence"`  // Must be 1 greater than the last committed TxInput
	Signature *crypto.Signature `json:"signature"` // Depends on the PubKey type and the whole Tx
}

func NewTxInputJSON(a TxInput) TxInputJSON {
	return TxInputJSON{
		Address:   a.Address,
		Coins:     a.Coins,
		Sequence:  common.JSONUint64(a.Sequence),
		Signature: a.Signature,
	}
}

func (a TxInputJSON) TxInput() TxInput {
	return TxInput{
		Address:   a.Address,
		Coins:     a.Coins,
		Sequence:  uint64(a.Sequence),
		Signature: a.Signature,
	}
}

func (a TxInput) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewTxInputJSON(a))
}

func (a *TxInput) UnmarshalJSON(data []byte) error {
	var b TxInputJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.TxInput()
	return nil
}

func (txIn TxInput) ValidateBasic() result.Result {
	if len(txIn.Address) != 20 {
		return result.Error("Invalid address length")
	}
	if !txIn.Coins.IsValid() {
		return result.Error("Invalid coins: %v", txIn.Coins)
	}
	// if txIn.Coins.IsZero() {
	// 	return result.Error("Coins cannot be zero")
	// }

	// *************
	// We should not need to check sequence here we are checking it with account sequence later anyway.
	// Besides the sequence number can be blank for half signed tx.
	// *************
	// if txIn.Sequence <= 0 {
	// 	return result.Error("Sequence must be greater than 0")
	// }
	return result.OK
}

func (txIn TxInput) String() string {
	return fmt.Sprintf("TxInput{%v,%v,%v,%v}", txIn.Address.Hex(), txIn.Coins, txIn.Sequence, txIn.Signature)
}

func NewTxInput(address common.Address, coins Coins, sequence int) TxInput {
	input := TxInput{
		Address:  address,
		Coins:    coins,
		Sequence: uint64(sequence),
	}
	return input
}

//-----------------------------------------------------------------------------

type TxOutput struct {
	Address common.Address `json:"address"` // Hash of the PubKey
	Coins   Coins          `json:"coins"`   // Amount of coins
}

func (txOut TxOutput) ValidateBasic() result.Result {
	if len(txOut.Address) != 20 {
		return result.Error("Invalid address length")
	}

	if !txOut.Coins.IsValid() {
		return result.Error("Invalid coins: %v", txOut.Coins)
	}
	// if txOut.Coins.IsZero() {
	// 	return result.Error("Coins cannot be zero")
	// }
	return result.OK
}

func (txOut TxOutput) String() string {
	return fmt.Sprintf("TxOutput{%v,%v}", txOut.Address.Hex(), txOut.Coins)
}

//-----------------------------------------------------------------------------

type CoinbaseTx struct {
	Proposer    TxInput
	Outputs     []TxOutput
	BlockHeight uint64
}

type CoinbaseTxJSON struct {
	Proposer    TxInput           `json:"proposer"`
	Outputs     []TxOutput        `json:"outputs"`
	BlockHeight common.JSONUint64 `json:"block_height"`
}

func NewCoinbaseTxJSON(a CoinbaseTx) CoinbaseTxJSON {
	return CoinbaseTxJSON{
		Proposer:    a.Proposer,
		Outputs:     a.Outputs,
		BlockHeight: common.JSONUint64(a.BlockHeight),
	}
}

func (a CoinbaseTxJSON) CoinbaseTx() CoinbaseTx {
	return CoinbaseTx{
		Proposer:    a.Proposer,
		Outputs:     a.Outputs,
		BlockHeight: uint64(a.BlockHeight),
	}
}

func (a CoinbaseTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewCoinbaseTxJSON(a))
}

func (a *CoinbaseTx) UnmarshalJSON(data []byte) error {
	var b CoinbaseTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.CoinbaseTx()
	return nil
}

func (_ *CoinbaseTx) AssertIsTx() {}

func (tx *CoinbaseTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Proposer.Signature
	tx.Proposer.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Proposer.Signature = sig
	return signBytes
}

func (tx *CoinbaseTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Proposer.Address == addr {
		tx.Proposer.Signature = sig
		return true
	}
	return false
}

func (tx *CoinbaseTx) String() string {
	return fmt.Sprintf("CoinbaseTx{0x0->%v}", tx.Outputs)
}

//-----------------------------------------------------------------------------

type SlashTx struct {
	Proposer        TxInput
	SlashedAddress  common.Address
	ReserveSequence uint64
	SlashProof      common.Bytes
}

type SlashTxJSON struct {
	Proposer        TxInput           `json:"proposer"`
	SlashedAddress  common.Address    `json:"slashed_address"`
	ReserveSequence common.JSONUint64 `json:"reserved_sequence"`
	SlashProof      common.Bytes      `json:"slash_proof"`
}

func NewSlashTxJSON(a SlashTx) SlashTxJSON {
	return SlashTxJSON{
		Proposer:        a.Proposer,
		SlashedAddress:  a.SlashedAddress,
		ReserveSequence: common.JSONUint64(a.ReserveSequence),
		SlashProof:      a.SlashProof,
	}
}

func (a SlashTxJSON) SlashTx() SlashTx {
	return SlashTx{
		Proposer:        a.Proposer,
		SlashedAddress:  a.SlashedAddress,
		ReserveSequence: uint64(a.ReserveSequence),
		SlashProof:      a.SlashProof,
	}
}

func (a SlashTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewSlashTxJSON(a))
}

func (a *SlashTx) UnmarshalJSON(data []byte) error {
	var b SlashTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.SlashTx()
	return nil
}

func (_ *SlashTx) AssertIsTx() {}

func (tx *SlashTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Proposer.Signature
	tx.Proposer.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Proposer.Signature = sig
	return signBytes
}

func (tx *SlashTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Proposer.Address == addr {
		tx.Proposer.Signature = sig
		return true
	}
	return false
}

func (tx *SlashTx) String() string {
	return fmt.Sprintf("SlashTx{%v->%v, reserve_sequence: %v, slash_proof: %v}",
		tx.SlashedAddress.Hex(), tx.Proposer.Address[:],
		tx.ReserveSequence, hex.EncodeToString(tx.SlashProof))
}

//-----------------------------------------------------------------------------

type SendTx struct {
	Fee     Coins      `json:"fee"` // Fee
	Inputs  []TxInput  `json:"inputs"`
	Outputs []TxOutput `json:"outputs"`
}

func (_ *SendTx) AssertIsTx() {}

func (tx *SendTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sigz := make([]*crypto.Signature, len(tx.Inputs))
	for i := range tx.Inputs {
		sigz[i] = tx.Inputs[i].Signature
		tx.Inputs[i].Signature = nil
	}
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	for i := range tx.Inputs {
		tx.Inputs[i].Signature = sigz[i]
	}
	return signBytes
}

func (tx *SendTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	for i, input := range tx.Inputs {
		if input.Address == addr {
			tx.Inputs[i].Signature = sig
			return true
		}
	}
	return false
}

func (tx *SendTx) String() string {
	return fmt.Sprintf("SendTx{fee: %v, %v->%v}", tx.Fee, tx.Inputs, tx.Outputs)
}

//-----------------------------------------------------------------------------

type ReserveFundTx struct {
	Fee         Coins    // Fee
	Source      TxInput  // Source account
	Collateral  Coins    // Collateral for the micropayment pool
	ResourceIDs []string // List of resource ID
	Duration    uint64
}

type ReserveFundTxJSON struct {
	Fee         Coins             `json:"fee"`          // Fee
	Source      TxInput           `json:"source"`       // Source account
	Collateral  Coins             `json:"collateral"`   // Collateral for the micropayment pool
	ResourceIDs []string          `json:"resource_ids"` // List of resource ID
	Duration    common.JSONUint64 `json:"duration"`
}

func NewReserveFundTxJSON(a ReserveFundTx) ReserveFundTxJSON {
	return ReserveFundTxJSON{
		Fee:         a.Fee,
		Source:      a.Source,
		Collateral:  a.Collateral,
		ResourceIDs: a.ResourceIDs,
		Duration:    common.JSONUint64(a.Duration),
	}
}

func (a ReserveFundTxJSON) ReserveFundTx() ReserveFundTx {
	return ReserveFundTx{
		Fee:         a.Fee,
		Source:      a.Source,
		Collateral:  a.Collateral,
		ResourceIDs: a.ResourceIDs,
		Duration:    uint64(a.Duration),
	}
}

func (a ReserveFundTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewReserveFundTxJSON(a))
}

func (a *ReserveFundTx) UnmarshalJSON(data []byte) error {
	var b ReserveFundTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.ReserveFundTx()
	return nil
}

func (_ *ReserveFundTx) AssertIsTx() {}

func (tx *ReserveFundTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Source.Signature = sig
	return signBytes
}

func (tx *ReserveFundTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Source.Address == addr {
		tx.Source.Signature = sig
		return true
	}
	return false
}

func (tx *ReserveFundTx) String() string {
	return fmt.Sprintf("ReserveFundTx{fee: %v, source: %v, collateral: %v, resource_ids: %v, duration: %v}",
		tx.Fee, tx.Source, tx.Collateral, tx.ResourceIDs, tx.Duration)
}

//-----------------------------------------------------------------------------

type ReleaseFundTx struct {
	Fee             Coins   // Fee
	Source          TxInput // source account
	ReserveSequence uint64
}

type ReleaseFundTxJSON struct {
	Fee             Coins             `json:"fee"`    // Fee
	Source          TxInput           `json:"source"` // source account
	ReserveSequence common.JSONUint64 `json:"reserve_sequence"`
}

func NewReleaseFundTxJSON(a ReleaseFundTx) ReleaseFundTxJSON {
	return ReleaseFundTxJSON{
		Fee:             a.Fee,
		Source:          a.Source,
		ReserveSequence: common.JSONUint64(a.ReserveSequence),
	}
}

func (a ReleaseFundTxJSON) ReleaseFundTx() ReleaseFundTx {
	return ReleaseFundTx{
		Fee:             a.Fee,
		Source:          a.Source,
		ReserveSequence: uint64(a.ReserveSequence),
	}
}

func (a ReleaseFundTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewReleaseFundTxJSON(a))
}

func (a *ReleaseFundTx) UnmarshalJSON(data []byte) error {
	var b ReleaseFundTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.ReleaseFundTx()
	return nil
}

func (_ *ReleaseFundTx) AssertIsTx() {}

func (tx *ReleaseFundTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Source.Signature = sig
	return signBytes
}

func (tx *ReleaseFundTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Source.Address == addr {
		tx.Source.Signature = sig
		return true
	}
	return false
}

func (tx *ReleaseFundTx) String() string {
	return fmt.Sprintf("ReleaseFundTx{fee: %v, source: %v, reserve_sequence: %v}", tx.Fee, tx.Source, tx.ReserveSequence)
}

//-----------------------------------------------------------------------------

type ServicePaymentTx struct {
	Fee             Coins   // Fee
	Source          TxInput // source account
	Target          TxInput // target account
	PaymentSequence uint64  // each on-chain settlement needs to increase the payment sequence by 1
	ReserveSequence uint64  // ReserveSequence to locate the ReservedFund
	ResourceID      string  // The corresponding resourceID
}

type ServicePaymentTxJSON struct {
	Fee             Coins             `json:"fee"`              // Fee
	Source          TxInput           `json:"source"`           // source account
	Target          TxInput           `json:"target"`           // target account
	PaymentSequence common.JSONUint64 `json:"payment_sequence"` // each on-chain settlement needs to increase the payment sequence by 1
	ReserveSequence common.JSONUint64 `json:"reserve_sequence"` // ReserveSequence to locate the ReservedFund
	ResourceID      string            `json:"resource_id"`      // The corresponding resourceID
}

func NewServicePaymentTxJSON(a ServicePaymentTx) ServicePaymentTxJSON {
	return ServicePaymentTxJSON{
		Fee:             a.Fee,
		Source:          a.Source,
		Target:          a.Target,
		PaymentSequence: common.JSONUint64(a.PaymentSequence),
		ReserveSequence: common.JSONUint64(a.ReserveSequence),
		ResourceID:      a.ResourceID,
	}
}

func (a ServicePaymentTxJSON) ServicePaymentTx() ServicePaymentTx {
	return ServicePaymentTx{
		Fee:             a.Fee,
		Source:          a.Source,
		Target:          a.Target,
		PaymentSequence: uint64(a.PaymentSequence),
		ReserveSequence: uint64(a.ReserveSequence),
		ResourceID:      a.ResourceID,
	}
}

func (a ServicePaymentTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewServicePaymentTxJSON(a))
}

func (a *ServicePaymentTx) UnmarshalJSON(data []byte) error {
	var b ServicePaymentTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.ServicePaymentTx()
	return nil
}

func (_ *ServicePaymentTx) AssertIsTx() {}

func (tx *ServicePaymentTx) SourceSignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)

	source := tx.Source
	target := tx.Target
	fee := tx.Fee

	tx.Source = TxInput{Address: source.Address, Coins: source.Coins}
	tx.Target = TxInput{Address: target.Address}
	tx.Fee = NewCoins(0, 0)

	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)

	tx.Source = source
	tx.Target = target
	tx.Fee = fee

	signBytes = addPrefixForSignBytes(signBytes)

	return signBytes
}

func (tx *ServicePaymentTx) SetSourceSignature(sig *crypto.Signature) {
	tx.Source.Signature = sig
}

func (tx *ServicePaymentTx) TargetSignBytes(chainID string) []byte {
	// TODO: remove chainID from all Tx sign bytes.
	signBytes := encodeToBytes(chainID)
	targetSig := tx.Target.Signature

	tx.Target.Signature = nil

	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Target.Signature = targetSig

	return signBytes
}

func (tx *ServicePaymentTx) SetTargetSignature(sig *crypto.Signature) {
	tx.Target.Signature = sig
}

// SignBytes this method only exists to satisfy the interface and should never be called.
// Call SourceSignBytes or TargetSignBytes instead.
func (tx *ServicePaymentTx) SignBytes(chainID string) []byte {
	panic("ServicePaymentTx.SignBytes() should not be called. Call SourceSignBytes or TargetSignBytes instead.")
}

func (tx *ServicePaymentTx) String() string {
	return fmt.Sprintf("ServicePaymentTx{fee: %v, source: %v, target: %v, reserve_sequence: %v, resource_id: %v}",
		tx.Fee, tx.Source, tx.Target, tx.ReserveSequence, tx.ResourceID)
}

// TxBytes returns the transaction data as well as all signatures
// It should return an error if Sign was never called
func (tx *ServicePaymentTx) TxBytes() ([]byte, error) {
	// TODO: verify it is signed
	return TxToBytes(tx)
}

//-----------------------------------------------------------------------------

type SplitRuleTx struct {
	Fee        Coins   // Fee
	ResourceID string  // ResourceID of the payment to be split
	Initiator  TxInput // Initiator of the split rule
	Splits     []Split // Agreed splits
	Duration   uint64  // Duration of the payment split in terms of blocks
}

type SplitRuleTxJSON struct {
	Fee        Coins             `json:"fee"`         // Fee
	ResourceID string            `json:"resource_id"` // ResourceID of the payment to be split
	Initiator  TxInput           `json:"initiator"`   // Initiator of the split rule
	Splits     []Split           `json:"splits"`      // Agreed splits
	Duration   common.JSONUint64 `json:"duration"`    // Duration of the payment split in terms of blocks
}

func NewSplitRuleTxJSON(a SplitRuleTx) SplitRuleTxJSON {
	return SplitRuleTxJSON{
		Fee:        a.Fee,
		ResourceID: a.ResourceID,
		Initiator:  a.Initiator,
		Splits:     a.Splits,
		Duration:   common.JSONUint64(a.Duration),
	}
}

func (a SplitRuleTxJSON) SplitRuleTx() SplitRuleTx {
	return SplitRuleTx{
		Fee:        a.Fee,
		ResourceID: a.ResourceID,
		Initiator:  a.Initiator,
		Splits:     a.Splits,
		Duration:   uint64(a.Duration),
	}
}

func (a SplitRuleTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewSplitRuleTxJSON(a))
}

func (a *SplitRuleTx) UnmarshalJSON(data []byte) error {
	var b SplitRuleTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.SplitRuleTx()
	return nil
}

func (_ *SplitRuleTx) AssertIsTx() {}

func (tx *SplitRuleTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Initiator.Signature
	tx.Initiator.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Initiator.Signature = sig
	return signBytes
}

func (tx *SplitRuleTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Initiator.Address == addr {
		tx.Initiator.Signature = sig
		return true
	}
	return false
}

func (tx *SplitRuleTx) String() string {
	return fmt.Sprintf("SplitRuleTx{fee: %v, resource_id: %v, initiator: %v, splits: %v, duration: %v}",
		tx.Fee, tx.ResourceID, tx.Initiator, tx.Splits, tx.Duration)
}

//-----------------------------------------------------------------------------

type SmartContractTx struct {
	From     TxInput
	To       TxOutput
	GasLimit uint64
	GasPrice *big.Int
	Data     common.Bytes
}

type SmartContractTxJSON struct {
	From     TxInput           `json:"from"`
	To       TxOutput          `json:"to"`
	GasLimit common.JSONUint64 `json:"gas_limit"`
	GasPrice *common.JSONBig   `json:"gas_price"`
	Data     common.Bytes      `json:"data"`
}

func NewSmartContractTxJSON(a SmartContractTx) SmartContractTxJSON {
	return SmartContractTxJSON{
		From:     a.From,
		To:       a.To,
		GasLimit: common.JSONUint64(a.GasLimit),
		GasPrice: (*common.JSONBig)(a.GasPrice),
		Data:     a.Data,
	}
}

func (a SmartContractTxJSON) SmartContractTx() SmartContractTx {
	return SmartContractTx{
		From:     a.From,
		To:       a.To,
		GasLimit: uint64(a.GasLimit),
		GasPrice: (*big.Int)(a.GasPrice),
		Data:     a.Data,
	}
}

func (a SmartContractTx) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewSmartContractTxJSON(a))
}

func (a *SmartContractTx) UnmarshalJSON(data []byte) error {
	var b SmartContractTxJSON
	if err := json.Unmarshal(data, &b); err != nil {
		return err
	}
	*a = b.SmartContractTx()
	return nil
}

func (_ *SmartContractTx) AssertIsTx() {}

func (tx *SmartContractTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.From.Signature
	tx.From.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.From.Signature = sig
	return signBytes
}

func (tx *SmartContractTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.From.Address == addr {
		tx.From.Signature = sig
		return true
	}
	return false
}

func (tx *SmartContractTx) String() string {
	return fmt.Sprintf("SmartContractTx{%v -> %v, value: %v, gas_limit: %v, gas_price: %v, data: %v}",
		tx.From.Address.Hex(), tx.To.Address.Hex(), tx.From.Coins.TFuelWei, tx.GasLimit, tx.GasPrice, tx.Data)
}

//-----------------------------------------------------------------------------

type DepositStakeTx struct {
	Fee     Coins    `json:"fee"`     // Fee
	Source  TxInput  `json:"source"`  // source staker account
	Holder  TxOutput `json:"holder"`  // stake holder account
	Purpose uint8    `json:"purpose"` // purpose e.g. stake for validator/guardian
}

func (_ *DepositStakeTx) AssertIsTx() {}

func (tx *DepositStakeTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Source.Signature = sig
	return signBytes
}

func (tx *DepositStakeTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Source.Address == addr {
		tx.Source.Signature = sig
		return true
	}
	return false
}

func (tx *DepositStakeTx) String() string {
	return fmt.Sprintf("DepositStakeTx{%v -> %v, stake: %v, purpose: %v}",
		tx.Source.Address, tx.Holder.Address, tx.Source.Coins.ThetaWei, tx.Purpose)
}

//-----------------------------------------------------------------------------

type WithdrawStakeTx struct {
	Fee     Coins    `json:"fee"`     // Fee
	Source  TxInput  `json:"source"`  // source staker account
	Holder  TxOutput `json:"holder"`  // stake holder account
	Purpose uint8    `json:"purpose"` // purpose e.g. stake for validator/guardian
}

func (_ *WithdrawStakeTx) AssertIsTx() {}

func (tx *WithdrawStakeTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	signBytes = addPrefixForSignBytes(signBytes)

	tx.Source.Signature = sig
	return signBytes
}

func (tx *WithdrawStakeTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Source.Address == addr {
		tx.Source.Signature = sig
		return true
	}
	return false
}

func (tx *WithdrawStakeTx) String() string {
	return fmt.Sprintf("DepositStakeTx{%v <- %v, stake: %v, purpose: %v}",
		tx.Source.Address, tx.Holder.Address, tx.Source.Coins.ThetaWei, tx.Purpose)
}

// --------------- Utils --------------- //

type EthereumTxWrapper struct {
	AccountNonce uint64          `json:"nonce"    gencodec:"required"`
	Price        *big.Int        `json:"gasPrice" gencodec:"required"`
	GasLimit     uint64          `json:"gas"      gencodec:"required"`
	Recipient    *common.Address `json:"to"       rlp:"nil"` // nil means contract creation
	Amount       *big.Int        `json:"value"    gencodec:"required"`
	Payload      []byte          `json:"input"    gencodec:"required"`
}

// Need to add the following prefix to the tx signbytes to be compatible with
// the Ethereum tx format
func addPrefixForSignBytes(signBytes common.Bytes) common.Bytes {
	ethTx := EthereumTxWrapper{
		AccountNonce: uint64(0),
		Price:        new(big.Int).SetUint64(0),
		GasLimit:     uint64(0),
		Recipient:    &common.Address{},
		Amount:       new(big.Int).SetUint64(0),
		Payload:      signBytes,
	}
	signBytes, err := rlp.EncodeToBytes(ethTx)
	if err != nil {
		log.Panic(err)
	}
	return signBytes
}
