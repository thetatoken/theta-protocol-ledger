package types

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
	"github.com/thetatoken/ukulele/rlp"
)

// ----------------- Common -------------------

func ToBytes(a interface{}) ([]byte, error) {
	return rlp.EncodeToBytes(a)
}

func FromBytes(in []byte, a interface{}) error {
	return rlp.DecodeBytes(in, a)
}

// ----------------- Tx -------------------

type TxType uint16

const (
	TxCoinbase TxType = iota
	TxSlash
	TxSend
	TxReserveFund
	TxReleaseFund
	TxServicePayment
	TxSplitRule
	TxUpdateValidators
)

func TxFromBytes(raw []byte) (Tx, error) {
	var txType TxType
	buff := bytes.NewBuffer(raw)
	err := rlp.Decode(buff, &txType)
	if err != nil {
		return nil, err
	}
	if txType == TxCoinbase {
		data := &CoinbaseTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxSlash {
		data := &SlashTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxSend {
		data := &SendTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxReserveFund {
		data := &ReserveFundTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxReleaseFund {
		data := &ReleaseFundTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxServicePayment {
		data := &ServicePaymentTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxSplitRule {
		data := &SplitRuleTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else if txType == TxUpdateValidators {
		data := &UpdateValidatorsTx{}
		err = rlp.Decode(buff, data)
		return data, err
	} else {
		return nil, fmt.Errorf("Unknown TX type: %v", txType)
	}
}

func TxToBytes(t Tx) ([]byte, error) {
	var buf bytes.Buffer
	var txType TxType
	switch t.(type) {
	case *CoinbaseTx:
		txType = TxCoinbase
	case *SlashTx:
		txType = TxSlash
	case *SendTx:
		txType = TxSend
	case *ReserveFundTx:
		txType = TxReserveFund
	case *ReleaseFundTx:
		txType = TxReleaseFund
	case *ServicePaymentTx:
		txType = TxServicePayment
	case *SplitRuleTx:
		txType = TxSplitRule
	case *UpdateValidatorsTx:
		txType = TxUpdateValidators
	default:
		return nil, errors.New("Unsupported message type")
	}
	err := rlp.Encode(&buf, txType)
	if err != nil {
		return nil, err
	}
	err = rlp.Encode(&buf, t)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
