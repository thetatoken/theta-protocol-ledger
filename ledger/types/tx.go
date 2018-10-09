package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/thetatoken/ukulele/common"
	"github.com/thetatoken/ukulele/common/result"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	"github.com/thetatoken/ukulele/rlp"
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
 - SplitContractTx      Payment split contract
 - UpdateValidatorsTx   Update validator set
*/

type Tx interface {
	AssertIsTx()
	SignBytes(chainID string) []byte
}

//-----------------------------------------------------------------------------

func TxID(chainID string, tx Tx) common.Hash {
	signBytes := tx.SignBytes(chainID)
	return crypto.Keccak256Hash(signBytes)
}

//--------------------------------------------------------------------------------

// Contract: This function is deterministic and completely reversible.
func jsonEscape(str string) string {
	escapedBytes, err := json.Marshal(str)
	if err != nil {
		panic(fmt.Sprintf("Error json-escaping a string: %v", str))
	}
	return string(escapedBytes)
}

func encodeToBytes(str string) []byte {
	encodedBytes, err := rlp.EncodeToBytes(str)
	if err != nil {
		panic(fmt.Sprintf("Failed to encode %v: %v", str, err))
	}
	return encodedBytes
}

//-----------------------------------------------------------------------------

type TxInput struct {
	Address   common.Address    `json:"address"`   // Hash of the PubKey
	Coins     Coins             `json:"coins"`     //
	Sequence  uint64            `json:"sequence"`  // Must be 1 greater than the last committed TxInput
	Signature *crypto.Signature `json:"signature"` // Depends on the PubKey type and the whole Tx
	PubKey    *crypto.PublicKey `json:"pub_key"`   // Is present iff Sequence == 0
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
	if txIn.Sequence == 1 && (txIn.PubKey == nil || txIn.PubKey.IsEmpty()) {
		return result.Error("PubKey must be present when Sequence == 1")
	}
	if txIn.Sequence > 1 && !(txIn.PubKey == nil || txIn.PubKey.IsEmpty()) {
		return result.Error("PubKey must be nil when Sequence > 1")
	}
	return result.OK
}

func (txIn TxInput) String() string {
	return fmt.Sprintf("TxInput{%v,%v,%v,%v,%v}", txIn.Address.Hex(), txIn.Coins, txIn.Sequence, txIn.Signature, txIn.PubKey)
}

func NewTxInput(pubKey *crypto.PublicKey, coins Coins, sequence int) TxInput {
	input := TxInput{
		Address:  pubKey.Address(),
		Coins:    coins,
		Sequence: uint64(sequence),
	}
	if sequence == 1 {
		input.PubKey = pubKey
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
	Proposer    TxInput    `json:"proposer"`
	Outputs     []TxOutput `json:"outputs"`
	BlockHeight uint64     `json:"block_height"`
}

func (_ *CoinbaseTx) AssertIsTx() {}

func (tx *CoinbaseTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Proposer.Signature
	tx.Proposer.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
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
	Proposer        TxInput        `json:"proposer"`
	SlashedAddress  common.Address `json:"slashed_address"`
	ReserveSequence uint64         `json:"reserved_sequence"`
	SlashProof      []byte         `json:"slash_proof"`
}

func (_ *SlashTx) AssertIsTx() {}

func (tx *SlashTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Proposer.Signature
	tx.Proposer.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
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
	Gas     uint64     `json:"gas"` // Gas
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
	return fmt.Sprintf("SendTx{%v/%v %v->%v}", tx.Gas, tx.Fee, tx.Inputs, tx.Outputs)
}

//-----------------------------------------------------------------------------

type ReserveFundTx struct {
	Gas         uint64   `json:"gas"`          // Gas
	Fee         Coins    `json:"fee"`          // Fee
	Source      TxInput  `json:"source"`       // Source account
	Collateral  Coins    `json:"collateral"`   // Collateral for the micropayment pool
	ResourceIDs [][]byte `json:"resource_ids"` // List of resource ID
	Duration    uint64   `json:"duration"`
}

func (_ *ReserveFundTx) AssertIsTx() {}

func (tx *ReserveFundTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
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
	return fmt.Sprintf("ReserveFundTx{%v/%v %v %v %v %v}", tx.Gas, tx.Fee, tx.Source, tx.Collateral, tx.ResourceIDs, tx.Duration)
}

//-----------------------------------------------------------------------------

type ReleaseFundTx struct {
	Gas             uint64  `json:"gas"`    // Gas
	Fee             Coins   `json:"fee"`    // Fee
	Source          TxInput `json:"source"` // source account
	ReserveSequence uint64  `json:"reserve_sequence"`
}

func (_ *ReleaseFundTx) AssertIsTx() {}

func (tx *ReleaseFundTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Source.Signature
	tx.Source.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
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
	return fmt.Sprintf("ReleaseFundTx{%v/%v %v %v}", tx.Gas, tx.Fee, tx.Source, tx.ReserveSequence)
}

//-----------------------------------------------------------------------------

type ServicePaymentTx struct {
	Gas             uint64  `json:"gas"`              // Gas
	Fee             Coins   `json:"fee"`              // Fee
	Source          TxInput `json:"source"`           // source account
	Target          TxInput `json:"target"`           // target account
	PaymentSequence uint64  `json:"payment_sequence"` // each on-chain settlement needs to increase the payment sequence by 1
	ReserveSequence uint64  `json:"reserve_sequence"` // ReserveSequence to locate the ReservedFund
	ResourceID      []byte  `json:"resource_id"`      // The corresponding resourceID
}

func (_ *ServicePaymentTx) AssertIsTx() {}

func (tx *ServicePaymentTx) SourceSignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)

	source := tx.Source
	target := tx.Target
	fee := tx.Fee
	gas := tx.Gas

	tx.Source = TxInput{Address: source.Address, Coins: source.Coins}
	tx.Target = TxInput{Address: target.Address}
	tx.Fee = NewCoins(0, 0)
	tx.Gas = uint64(0)

	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)

	tx.Source = source
	tx.Target = target
	tx.Fee = fee
	tx.Gas = gas

	return signBytes
}

func (tx *ServicePaymentTx) TargetSignBytes(chainID string) []byte {
	// TODO: remove chainID from all Tx sign bytes.
	signBytes := encodeToBytes(chainID)
	targetSig := tx.Target.Signature

	tx.Target.Signature = nil

	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)

	tx.Target.Signature = targetSig

	return signBytes
}

// SignBytes this method only exists to satisfy the interface and should never be called.
// Call SourceSignBytes or TargetSignBytes instead.
func (tx *ServicePaymentTx) SignBytes(chainID string) []byte {
	panic("ServicePaymentTx.SignBytes() should not be called. Call SourceSignBytes or TargetSignBytes instead.")
}

func (tx *ServicePaymentTx) String() string {
	return fmt.Sprintf("ServicePaymentTx{%v/%v %v %v %v %v}", tx.Gas, tx.Fee, tx.Source, tx.Target, tx.ReserveSequence, tx.ResourceID)
}

// TxBytes returns the transaction data as well as all signatures
// It should return an error if Sign was never called
func (tx *ServicePaymentTx) TxBytes() ([]byte, error) {
	// TODO: verify it is signed
	return TxToBytes(tx)
}

//-----------------------------------------------------------------------------

type SplitContractTx struct {
	Gas        uint64       `json:"gas"`         // Gas
	Fee        Coins        `json:"fee"`         // Fee
	ResourceID common.Bytes `json:"resource_id"` // ResourceID of the payment to be split
	Initiator  TxInput      `json:"initiator"`   // Initiator of the split contract
	Splits     []Split      `json:"splits"`      // Agreed splits
	Duration   uint64       `json:"duration"`    // Duration of the payment split in terms of blocks
}

func (_ *SplitContractTx) AssertIsTx() {}

func (tx *SplitContractTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	sig := tx.Initiator.Signature
	tx.Initiator.Signature = nil
	txBytes, _ := TxToBytes(tx)
	signBytes = append(signBytes, txBytes...)
	tx.Initiator.Signature = sig
	return signBytes
}

func (tx *SplitContractTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Initiator.Address == addr {
		tx.Initiator.Signature = sig
		return true
	}
	return false
}

func (tx *SplitContractTx) String() string {
	return fmt.Sprintf("SplitContractTx{%v/%v %v %v %v %v}", tx.Gas, tx.Fee, tx.ResourceID, tx.Initiator, tx.Splits, tx.Duration)
}

//-----------------------------------------------------------------------------

type UpdateValidatorsTx struct {
	Gas        uint64            `json:"gas"`        // Gas
	Fee        Coins             `json:"fee"`        // Fee
	Validators []*core.Validator `json:"validators"` // validators diff
	Proposer   TxInput           `json:"source"`     // source account
}

func (_ *UpdateValidatorsTx) AssertIsTx() {}

func (tx *UpdateValidatorsTx) SignBytes(chainID string) []byte {
	signBytes := encodeToBytes(chainID)
	for _, v := range tx.Validators {
		bytes, err := rlp.EncodeToBytes(v)
		if err != nil {
			signBytes = append(signBytes, bytes...)
		}
	}
	return signBytes
}

func (tx *UpdateValidatorsTx) SetSignature(addr common.Address, sig *crypto.Signature) bool {
	if tx.Proposer.Address == addr {
		tx.Proposer.Signature = sig
		return true
	}
	return false
}

func (tx *UpdateValidatorsTx) String() string {
	return fmt.Sprintf("UpdateValidatorsTx{%v}", tx.Validators)
}
