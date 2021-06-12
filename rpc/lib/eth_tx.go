// Copyright 2020 The go-ethereum Authors, Adapted to Theta
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package lib

import (
	"errors"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/rlp"
)

// TranslateEthTx an ETH transaction to a Theta smart contract transaction
func TranslateEthTx(ethTxStr string) (*types.SmartContractTx, error) {
	var ethTx *EthTransaction
	err := rlp.DecodeBytes(common.Hex2Bytes(ethTxStr), &ethTx)
	if err != nil {
		return nil, err
	}

	ethTxHash := types.RLPHash([]interface{}{
		ethTx.inner.nonce(),
		ethTx.inner.gasPrice(),
		ethTx.inner.gas(),
		ethTx.inner.to(),
		ethTx.inner.value(),
		ethTx.inner.data(),
	})

	r, s, v := ethTx.inner.rawSignatureValues()
	sig, err := crypto.EncodeSignature(r, s, v)
	if err != nil {
		return nil, err
	}

	fromAddr, err := crypto.HomesteadSignerSender(ethTxHash, sig)
	if err != nil {
		return nil, err
	}

	coins := types.Coins{
		ThetaWei: big.NewInt(0),
		TFuelWei: ethTx.inner.value(),
	}

	from := types.TxInput{
		Address:   fromAddr,
		Coins:     coins,
		Sequence:  ethTx.inner.nonce(),
		Signature: sig,
	}

	if ethTx.inner.to() == nil {
		return nil, errors.New("To address is nil")
	}

	to := types.TxOutput{
		Address: *ethTx.inner.to(),
		Coins:   types.NewCoins(0, 0),
	}

	thetaTx := types.SmartContractTx{
		From:     from,
		To:       to,
		GasLimit: ethTx.inner.gas(),
		GasPrice: ethTx.inner.gasPrice(),
		Data:     ethTx.inner.data(),
	}

	return &thetaTx, nil
}

// EthTransaction is an Ethereum transaction.
type EthTransaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// setDecoded sets the inner transaction and size after decoding.
func (tx *EthTransaction) setDecoded(inner TxData, size int) {
	tx.inner = inner
	tx.time = time.Now()
	if size > 0 {
		tx.size.Store(float64(size))
	}
}

// TxData is the underlying data of a transaction.
//
// This is implemented by LegacyTx and AccessListTx.
type TxData interface {
	//txType() byte // returns the type ID
	copy() TxData // creates a deep copy and initializes all fields

	chainID() *big.Int
	data() []byte
	gas() uint64
	gasPrice() *big.Int
	value() *big.Int
	nonce() uint64
	to() *common.Address

	rawSignatureValues() (v, r, s *big.Int)
	setSignatureValues(chainID, v, r, s *big.Int)
}

// Transaction is an Ethereum transaction.
type Transaction struct {
	inner TxData    // Consensus contents of a transaction
	time  time.Time // Time first seen locally (spam avoidance)

	// caches
	hash atomic.Value
	size atomic.Value
	from atomic.Value
}

// NewTx creates a new transaction.
func NewTx(inner TxData) *EthTransaction {
	tx := new(EthTransaction)
	tx.setDecoded(inner.copy(), 0)
	return tx
}

// LegacyTx is the transaction data of regular Ethereum transactions.
type LegacyTx struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	Data     []byte          // contract invocation input data
	V, R, S  *big.Int        // signature values
}

// NewTransaction creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewTransaction(nonce uint64, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *EthTransaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// NewContractCreation creates an unsigned legacy transaction.
// Deprecated: use NewTx instead.
func NewContractCreation(nonce uint64, amount *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *EthTransaction {
	return NewTx(&LegacyTx{
		Nonce:    nonce,
		Value:    amount,
		Gas:      gasLimit,
		GasPrice: gasPrice,
		Data:     data,
	})
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *LegacyTx) copy() TxData {
	cpy := &LegacyTx{
		Nonce: tx.Nonce,
		To:    tx.To, // TODO: copy pointed-to address
		Data:  common.CopyBytes(tx.Data),
		Gas:   tx.Gas,
		// These are initialized below.
		Value:    new(big.Int),
		GasPrice: new(big.Int),
		V:        new(big.Int),
		R:        new(big.Int),
		S:        new(big.Int),
	}
	if tx.Value != nil {
		cpy.Value.Set(tx.Value)
	}
	if tx.GasPrice != nil {
		cpy.GasPrice.Set(tx.GasPrice)
	}
	if tx.V != nil {
		cpy.V.Set(tx.V)
	}
	if tx.R != nil {
		cpy.R.Set(tx.R)
	}
	if tx.S != nil {
		cpy.S.Set(tx.S)
	}
	return cpy
}

// accessors for innerTx.

func (tx *LegacyTx) chainID() *big.Int   { return deriveChainId(tx.V) }
func (tx *LegacyTx) data() []byte        { return tx.Data }
func (tx *LegacyTx) gas() uint64         { return tx.Gas }
func (tx *LegacyTx) gasPrice() *big.Int  { return tx.GasPrice }
func (tx *LegacyTx) value() *big.Int     { return tx.Value }
func (tx *LegacyTx) nonce() uint64       { return tx.Nonce }
func (tx *LegacyTx) to() *common.Address { return tx.To }

func (tx *LegacyTx) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *LegacyTx) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}

// deriveChainId derives the chain id from the given v parameter
func deriveChainId(v *big.Int) *big.Int {
	if v.BitLen() <= 64 {
		v := v.Uint64()
		if v == 27 || v == 28 {
			return new(big.Int)
		}
		return new(big.Int).SetUint64((v - 35) / 2)
	}
	v = new(big.Int).Sub(v, big.NewInt(35))
	return v.Div(v, big.NewInt(2))
}
