package types

import (
	"fmt"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
	"github.com/thetatoken/ukulele/core"
	"github.com/thetatoken/ukulele/crypto"
	s "github.com/thetatoken/ukulele/ledger/types/serialization"

	log "github.com/sirupsen/logrus"
)

// ----------------- Common -------------------

func ToBytes(a interface{}) ([]byte, error) {
	switch a.(type) {
	default:
		return nil, errors.New(fmt.Sprintf("ToBytes: Unsupported type: %v", reflect.TypeOf(a)))
	case *crypto.PublicKey:
		pk := PublicKeyToProto(a.(*crypto.PublicKey))
		return proto.Marshal(pk)
	case *crypto.PrivateKey:
		sk := PrivateKeyToProto(a.(*crypto.PrivateKey))
		return proto.Marshal(sk)
	case *crypto.Signature:
		sig := SignatureToProto(a.(*crypto.Signature))
		return proto.Marshal(sig)
	case *Account:
		acc := AccountToProto(a.(*Account))
		return proto.Marshal(acc)
	case *OverspendingProof:
		p := OverspendingProofToProto(a.(*OverspendingProof))
		return proto.Marshal(p)
	case *SplitContract:
		sc := SplitContractToProto(a.(*SplitContract))
		return proto.Marshal(sc)
	}
}

func FromBytes(in []byte, a interface{}) error {
	switch a.(type) {
	default:
		return errors.New(fmt.Sprintf("FromBytes: Unsupported type: %v", reflect.TypeOf(a)))
	case *crypto.PublicKey:
		pk := &s.PublicKey{}
		if err := proto.Unmarshal(in, pk); err != nil {
			return err
		}
		ap := a.(*crypto.PublicKey)
		p := PublicKeyFromProto(pk)
		*ap = *p
	case *crypto.PrivateKey:
		sk := &s.PrivateKey{}
		if err := proto.Unmarshal(in, sk); err != nil {
			return err
		}
		as := a.(*crypto.PrivateKey)
		s := PrivateKeyFromProto(sk)
		*as = *s
	case *crypto.Signature:
		sig := &s.Signature{}
		if err := proto.Unmarshal(in, sig); err != nil {
			return err
		}
		ap := a.(*crypto.Signature)
		p := SignatureFromProto(sig)
		*ap = *p
	case *Account:
		acc := &s.Account{}
		if err := proto.Unmarshal(in, acc); err != nil {
			return err
		}
		ap := a.(*Account)
		*ap = *AccountFromProto(acc)
	case *OverspendingProof:
		p := &s.OverspendingProof{}
		if err := proto.Unmarshal(in, p); err != nil {
			return err
		}
		ap := a.(*OverspendingProof)
		*ap = *OverspendingProofFromProto(p)
	case *SplitContract:
		sc := &s.SplitContract{}
		if err := proto.Unmarshal(in, sc); err != nil {
			return err
		}
		ap := a.(*SplitContract)
		*ap = *SplitContractFromProto(sc)
	}
	return nil
}

// ----------------- PublicKey -------------------

func PublicKeyToProto(pk *crypto.PublicKey) *s.PublicKey {
	pubKey := &s.PublicKey{}
	if pk != nil {
		pubKey.Data = pk.ToBytes()[:]
	}
	return pubKey
}

func PublicKeyFromProto(pubKey *s.PublicKey) *crypto.PublicKey {
	var pk *crypto.PublicKey
	var err error
	if pubKey != nil && len(pubKey.Data) > 0 {
		pk, err = crypto.PublicKeyFromBytes(pubKey.Data)
		if err != nil {
			log.Errorf("Error parsing pubKey from proto: %v", err)
		}
	}
	return pk
}

// ----------------- PrivateKey -------------------

func PrivateKeyToProto(sk *crypto.PrivateKey) *s.PrivateKey {
	privKey := &s.PrivateKey{}
	if sk != nil {
		privKey.Data = sk.ToBytes()[:]
	}
	return privKey
}

func PrivateKeyFromProto(privKey *s.PrivateKey) *crypto.PrivateKey {
	var sk *crypto.PrivateKey
	var err error
	if len(privKey.Data) > 0 {
		sk, err = crypto.PrivateKeyFromBytes(privKey.Data)
		if err != nil {
			log.Errorf("Error parsing privKey from proto: %v", err)
		}
	}
	return sk
}

// ----------------- Signature -------------------

func SignatureToProto(sig *crypto.Signature) *s.Signature {
	signature := &s.Signature{}
	if sig != nil {
		signature.Data = sig.ToBytes()[:]
	}
	return signature
}

func SignatureFromProto(signature *s.Signature) *crypto.Signature {
	var sig *crypto.Signature
	var err error
	if len(signature.Data) > 0 {
		sig, err = crypto.SignatureFromBytes(signature.Data)
		if err != nil {
			log.Errorf("Error parsing signature from proto: %v", err)
		}
	}
	return sig
}

// --------------- OverspendingProof --------------

func OverspendingProofFromProto(msg *s.OverspendingProof) *OverspendingProof {
	obj := &OverspendingProof{}
	obj.ReserveSequence = int(msg.ReserveSequence)
	for _, payment := range msg.ServicePayments {
		obj.ServicePayments = append(obj.ServicePayments, *ServicePaymentTxFromProto(payment))
	}
	return obj
}

func OverspendingProofToProto(obj *OverspendingProof) *s.OverspendingProof {
	msg := &s.OverspendingProof{}
	msg.ReserveSequence = int64(obj.ReserveSequence)
	for _, payment := range obj.ServicePayments {
		msg.ServicePayments = append(msg.ServicePayments, ServicePaymentTxToProto(&payment))
	}
	return msg
}

// ----------------  Split   -------------------

func SplitFromProto(split *s.Split) *Split {
	sp := &Split{}
	copy(sp.Address[:], split.Address)
	sp.Percentage = uint(split.Percentage)
	return sp
}

func SplitToProto(sp *Split) *s.Split {
	split := &s.Split{}
	split.Address = sp.Address[:]
	split.Percentage = int64(sp.Percentage)
	return split
}

// ----------------  SplitContract   -------------------

func SplitContractFromProto(splitContract *s.SplitContract) *SplitContract {
	sc := &SplitContract{}
	copy(sc.InitiatorAddress[:], splitContract.InitiatorAddress)
	sc.ResourceId = splitContract.ResourceId
	for _, split := range splitContract.Splits {
		sc.Splits = append(sc.Splits, *SplitFromProto(split))
	}
	sc.EndBlockHeight = uint32(splitContract.EndBlockHeight)
	return sc
}

func SplitContractToProto(sc *SplitContract) *s.SplitContract {
	splitContract := &s.SplitContract{}
	splitContract.InitiatorAddress = sc.InitiatorAddress[:]
	splitContract.ResourceId = sc.ResourceId
	for _, split := range sc.Splits {
		splitContract.Splits = append(splitContract.Splits, SplitToProto(&split))
	}
	splitContract.EndBlockHeight = int32(sc.EndBlockHeight)
	return splitContract
}

// ----------------  Coin   -------------------

func CoinFromProto(coin *s.Coin) *Coin {
	c := &Coin{}
	c.Denom = coin.Denom
	c.Amount = coin.Amount
	return c
}

func CoinToProto(c *Coin) *s.Coin {
	coin := &s.Coin{}
	coin.Amount = c.Amount
	coin.Denom = c.Denom
	return coin
}

// ----------------  Coins   -------------------

func CoinsFromProto(coins []*s.Coin) Coins {
	if coins == nil {
		return nil
	}
	cc := Coins{}
	for _, coin := range coins {
		cc = append(cc, *CoinFromProto(coin))
	}
	return cc
}

func CoinsToProto(cc Coins) []*s.Coin {
	if cc == nil {
		return nil
	}
	coins := []*s.Coin{}
	for _, c := range cc {
		coins = append(coins, CoinToProto(&c))
	}
	return coins
}

// ---------------- TransferRecord --------------

func TransferRecordFromProto(r *s.TransferRecord) *TransferRecord {
	ret := TransferRecord{}
	ret.ServicePayment = *ServicePaymentTxFromProto(r.ServicePayment)
	return &ret
}

func TransferRecordToProto(r *TransferRecord) *s.TransferRecord {
	ret := s.TransferRecord{}
	ret.ServicePayment = ServicePaymentTxToProto(&r.ServicePayment)
	return &ret
}

// ---------------- ReservedFund ---------------

func ReserveFundFromProto(rf *s.ReservedFund) *ReservedFund {
	r := ReservedFund{}
	r.Collateral = CoinsFromProto(rf.Collateral)
	r.InitialFund = CoinsFromProto(rf.InitialFund)
	r.UsedFund = CoinsFromProto(rf.UsedFund)
	for _, rsid := range rf.ResourceIds {
		r.ResourceIds = append(r.ResourceIds, rsid)
	}
	r.EndBlockHeight = uint32(rf.EndBlockHeight)
	r.ReserveSequence = int(rf.ReserveSequence)
	for _, record := range rf.TransferRecord {
		r.TransferRecords = append(r.TransferRecords, *TransferRecordFromProto(record))
	}
	return &r
}

func ReserveFundToProto(r *ReservedFund) *s.ReservedFund {
	rf := &s.ReservedFund{}
	rf.Collateral = CoinsToProto(r.Collateral)
	rf.InitialFund = CoinsToProto(r.InitialFund)
	rf.UsedFund = CoinsToProto(r.UsedFund)
	for _, rsid := range r.ResourceIds {
		rf.ResourceIds = append(rf.ResourceIds, rsid)
	}
	rf.EndBlockHeight = int32(r.EndBlockHeight)
	rf.ReserveSequence = int32(r.ReserveSequence)
	for _, record := range r.TransferRecords {
		rf.TransferRecord = append(rf.TransferRecord, TransferRecordToProto(&record))
	}
	return rf
}

// ---------------- Account -------------------

func AccountFromProto(account *s.Account) *Account {
	acc := Account{}
	acc.PubKey = PublicKeyFromProto(account.PubKey)
	acc.Sequence = int(account.Sequence)
	acc.LastUpdatedBlockHeight = uint32(account.LastUpdatedBlockHeight)
	acc.Balance = CoinsFromProto(account.Balance)
	for _, fund := range account.ReservedFunds {
		acc.ReservedFunds = append(acc.ReservedFunds, *ReserveFundFromProto(fund))
	}
	return &acc
}

func AccountToProto(acc *Account) *s.Account {
	account := &s.Account{}
	pubkey := PublicKeyToProto(acc.PubKey)
	account.PubKey = pubkey
	account.Sequence = int64(acc.Sequence)
	account.LastUpdatedBlockHeight = int32(acc.LastUpdatedBlockHeight)
	account.Balance = CoinsToProto(acc.Balance)
	for _, fund := range acc.ReservedFunds {
		account.ReservedFunds = append(account.ReservedFunds, ReserveFundToProto(&fund))
	}
	return account
}

// ----------------- TxInput -------------------

func InputFromProto(ti *s.TxInput) *TxInput {
	txInput := &TxInput{}
	copy(txInput.Address[:], ti.Address)
	txInput.Sequence = int(ti.Sequence)
	txInput.Coins = CoinsFromProto(ti.Coins)
	txInput.Signature = SignatureFromProto(ti.Signature)
	txInput.PubKey = PublicKeyFromProto(ti.Pubkey)
	return txInput
}

func InputToProto(txInput *TxInput) *s.TxInput {
	ti := &s.TxInput{}
	ti.Address = txInput.Address[:]
	ti.Sequence = int64(txInput.Sequence)
	ti.Coins = CoinsToProto(txInput.Coins)
	ti.Pubkey = PublicKeyToProto(txInput.PubKey)
	ti.Signature = SignatureToProto(txInput.Signature)
	return ti
}

// ----------------- TxOutput -------------------

func OutputFromProto(to *s.TxOutput) *TxOutput {
	txOutput := &TxOutput{}
	copy(txOutput.Address[:], to.Address)
	txOutput.Coins = CoinsFromProto(to.Coins)
	return txOutput
}

func OutputToProto(txOutput *TxOutput) *s.TxOutput {
	to := &s.TxOutput{}
	to.Address = txOutput.Address[:]
	to.Coins = CoinsToProto(txOutput.Coins)
	return to
}

// ----------------- Tx -------------------

func TxFromProto(tx *s.Tx) Tx {
	switch tx.Tx.(type) {
	default:
		panic(fmt.Sprintf("TxFromProto: Unsupported Tx type: %v", reflect.TypeOf(tx.Tx)))
	case *s.Tx_Coinbase:
		return CoinbaseTxFromProto(tx.GetCoinbase())
	case *s.Tx_Slash:
		return SlashTxFromProto(tx.GetSlash())
	case *s.Tx_Send:
		return SendTxFromProto(tx.GetSend())
	case *s.Tx_Reserve:
		return ReserveFundTxFromProto(tx.GetReserve())
	case *s.Tx_Release:
		return ReleaseFundTxFromProto(tx.GetRelease())
	case *s.Tx_ServicePayment:
		return ServicePaymentTxFromProto(tx.GetServicePayment())
	case *s.Tx_SplitContract:
		return SplitContractTxFromProto(tx.GetSplitContract())
	case *s.Tx_UpdateValidators:
		return UpdateValidatorsTxFromProto(tx.GetUpdateValidators())
	}
}

func TxToProto(t Tx) *s.Tx {
	switch t.(type) {
	default:
		panic(fmt.Sprintf("TxToProto: Unsupported Tx type: %v", reflect.TypeOf(t)))
	case *CoinbaseTx:
		return &s.Tx{&s.Tx_Coinbase{CoinbaseTxToProto(t.(*CoinbaseTx))}}
	case *SlashTx:
		return &s.Tx{&s.Tx_Slash{SlashTxToProto(t.(*SlashTx))}}
	case *SendTx:
		return &s.Tx{&s.Tx_Send{SendTxToProto(t.(*SendTx))}}
	case *ReserveFundTx:
		return &s.Tx{&s.Tx_Reserve{ReserveFundTxToProto(t.(*ReserveFundTx))}}
	case *ReleaseFundTx:
		return &s.Tx{&s.Tx_Release{ReleaseFundTxToProto(t.(*ReleaseFundTx))}}
	case *ServicePaymentTx:
		return &s.Tx{&s.Tx_ServicePayment{ServicePaymentTxToProto(t.(*ServicePaymentTx))}}
	case *SplitContractTx:
		return &s.Tx{&s.Tx_SplitContract{SplitContractTxToProto(t.(*SplitContractTx))}}
	case *UpdateValidatorsTx:
		return &s.Tx{&s.Tx_UpdateValidators{UpdateValidatorsTxToProto(t.(*UpdateValidatorsTx))}}
	}
}

func TxFromBytes(in []byte) (Tx, error) {
	msg := &s.Tx{}
	if err := proto.Unmarshal(in, msg); err != nil {
		return nil, err
	}
	return TxFromProto(msg), nil
}

func TxToBytes(tx Tx) []byte {
	msg := TxToProto(tx)
	b, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}

// ----------------- CoinbaseTx -------------------

func CoinbaseTxFromProto(tx *s.CoinbaseTx) *CoinbaseTx {
	st := &CoinbaseTx{}
	st.Proposer = *InputFromProto(tx.Proposer)
	for _, output := range tx.Outputs {
		st.Outputs = append(st.Outputs, *OutputFromProto(output))
	}
	st.BlockHeight = uint32(tx.BlockHeight)
	return st
}

func CoinbaseTxToProto(t *CoinbaseTx) *s.CoinbaseTx {
	tx := &s.CoinbaseTx{}
	tx.Proposer = InputToProto(&t.Proposer)
	for _, output := range t.Outputs {
		tx.Outputs = append(tx.Outputs, OutputToProto(&output))
	}
	tx.BlockHeight = int32(t.BlockHeight)
	return tx
}

// ----------------- SlashTx -------------------

func SlashTxFromProto(tx *s.SlashTx) *SlashTx {
	st := &SlashTx{}
	st.Proposer = *InputFromProto(tx.Proposer)
	copy(st.SlashedAddress[:], tx.SlashedAddress)
	st.ReserveSequence = int(tx.ReserveSequence)
	st.SlashProof = tx.SlashProof
	return st
}

func SlashTxToProto(t *SlashTx) *s.SlashTx {
	tx := &s.SlashTx{}
	tx.Proposer = InputToProto(&t.Proposer)
	tx.SlashedAddress = t.SlashedAddress[:]
	tx.ReserveSequence = int64(t.ReserveSequence)
	tx.SlashProof = t.SlashProof
	return tx
}

// ----------------- SendTx -------------------

func SendTxFromProto(tx *s.SendTx) *SendTx {
	st := &SendTx{}
	st.Gas = tx.Gas
	st.Fee = *CoinFromProto(tx.Fee)
	for _, input := range tx.Inputs {
		st.Inputs = append(st.Inputs, *InputFromProto(input))
	}
	for _, output := range tx.Outputs {
		st.Outputs = append(st.Outputs, *OutputFromProto(output))
	}
	return st
}

func SendTxToProto(st *SendTx) *s.SendTx {
	tx := &s.SendTx{}
	tx.Gas = st.Gas
	tx.Fee = CoinToProto(&st.Fee)
	for _, input := range st.Inputs {
		tx.Inputs = append(tx.Inputs, InputToProto(&input))
	}
	for _, output := range st.Outputs {
		tx.Outputs = append(tx.Outputs, OutputToProto(&output))
	}
	return tx
}

// ----------------- ReserveFundTx -------------------

func ReserveFundTxFromProto(tx *s.ReserveFundTx) *ReserveFundTx {
	rf := &ReserveFundTx{}
	rf.Gas = tx.Gas
	rf.Fee = *CoinFromProto(tx.Fee)
	rf.Source = *InputFromProto(tx.Source)
	rf.Collateral = CoinsFromProto(tx.Collateral)
	for _, rsid := range tx.ResourceIds {
		rf.ResourceIds = append(rf.ResourceIds, rsid)
	}
	rf.Duration = uint32(tx.Duration)
	return rf
}

func ReserveFundTxToProto(rf *ReserveFundTx) *s.ReserveFundTx {
	tx := &s.ReserveFundTx{}
	tx.Gas = rf.Gas
	tx.Fee = CoinToProto(&rf.Fee)
	tx.Source = InputToProto(&rf.Source)
	tx.Collateral = CoinsToProto(rf.Collateral)
	for _, rsid := range rf.ResourceIds {
		tx.ResourceIds = append(tx.ResourceIds, rsid)
	}
	tx.Duration = int64(rf.Duration)
	return tx
}

// ----------------- ReleaseFundTx -------------------

func ReleaseFundTxFromProto(tx *s.ReleaseFundTx) *ReleaseFundTx {
	rf := &ReleaseFundTx{}
	rf.Gas = tx.Gas
	rf.Fee = *CoinFromProto(tx.Fee)
	rf.Source = *InputFromProto(tx.Source)
	rf.ReserveSequence = int(tx.ReserveSequence)
	return rf
}

func ReleaseFundTxToProto(rf *ReleaseFundTx) *s.ReleaseFundTx {
	tx := &s.ReleaseFundTx{}
	tx.Gas = rf.Gas
	tx.Fee = CoinToProto(&rf.Fee)
	tx.Source = InputToProto(&rf.Source)
	tx.ReserveSequence = int64(rf.ReserveSequence)
	return tx
}

// ----------------- ServicePaymentTx -------------------

func ServicePaymentTxFromProto(tx *s.ServicePaymentTx) *ServicePaymentTx {
	sp := &ServicePaymentTx{}
	sp.Gas = tx.Gas
	sp.Fee = *CoinFromProto(tx.Fee)
	sp.Source = *InputFromProto(tx.Source)
	sp.Target = *InputFromProto(tx.Target)
	sp.PaymentSequence = int(tx.PaymentSequence)
	sp.ReserveSequence = int(tx.ReserveSequence)
	sp.ResourceId = tx.ResourceId
	return sp
}

func ServicePaymentTxToProto(sp *ServicePaymentTx) *s.ServicePaymentTx {
	tx := &s.ServicePaymentTx{}
	tx.Gas = sp.Gas
	tx.Fee = CoinToProto(&sp.Fee)
	tx.Source = InputToProto(&sp.Source)
	tx.Target = InputToProto(&sp.Target)
	tx.PaymentSequence = int64(sp.PaymentSequence)
	tx.ReserveSequence = int64(sp.ReserveSequence)
	tx.ResourceId = sp.ResourceId
	return tx
}

// ----------------- SplitContract -------------------

func SplitContractTxFromProto(tx *s.SplitContractTx) *SplitContractTx {
	sc := &SplitContractTx{}
	sc.Gas = tx.Gas
	sc.Fee = *(CoinFromProto)(tx.Fee)
	sc.ResourceId = tx.ResourceId
	sc.Initiator = *InputFromProto(tx.Initiator)
	for _, sp := range tx.Splits {
		sc.Splits = append(sc.Splits, *SplitFromProto(sp))
	}
	sc.Duration = uint32(tx.Duration)
	return sc
}

func SplitContractTxToProto(sc *SplitContractTx) *s.SplitContractTx {
	tx := &s.SplitContractTx{}
	tx.Gas = sc.Gas
	tx.Fee = CoinToProto(&sc.Fee)
	tx.ResourceId = sc.ResourceId
	tx.Initiator = InputToProto(&sc.Initiator)
	for _, sp := range sc.Splits {
		tx.Splits = append(tx.Splits, SplitToProto(&sp))
	}
	tx.Duration = int64(sc.Duration)

	return tx
}

// ----------------- UpdateValidatorsTx -------------------

func UpdateValidatorsTxFromProto(tx *s.UpdateValidatorsTx) *UpdateValidatorsTx {
	sp := &UpdateValidatorsTx{}
	sp.Proposer = *InputFromProto(tx.Proposer)
	for _, v := range tx.Validators {
		vaPubKey := v.PubKey
		stake := uint64(v.GetStake())
		va := core.NewValidator(vaPubKey, stake)
		sp.Validators = append(sp.Validators, &va)
	}

	return sp
}

func UpdateValidatorsTxToProto(tx *UpdateValidatorsTx) *s.UpdateValidatorsTx {
	msg := &s.UpdateValidatorsTx{}
	msg.Proposer = InputToProto(&tx.Proposer)
	for _, va := range tx.Validators {
		v := &s.Validator{}
		pubKey := va.PublicKey()
		v.PubKey = (&pubKey).ToBytes()
		v.Stake = int64(va.Stake())
		msg.Validators = append(msg.Validators, v)
	}
	return msg
}
