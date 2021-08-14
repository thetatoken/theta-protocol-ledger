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

package types

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	"github.com/thetatoken/theta/rlp"
)

// TranslateEthTx an ETH transaction to a Theta smart contract transaction
func TranslateEthTx(ethTxStr string) (*SmartContractTx, error) {
	var ethTx *EthTransaction

	if strings.HasPrefix(ethTxStr, "0x") {
		ethTxStr = ethTxStr[2:]
	}

	ethTxBytes, err := hex.DecodeString(ethTxStr)
	if err != nil {
		return nil, err
	}

	err = rlp.DecodeBytes(ethTxBytes, &ethTx)
	if err != nil {
		return nil, err
	}

	ethSigningHash := RLPHash([]interface{}{
		ethTx.nonce(),
		ethTx.gasPrice(),
		ethTx.gas(),
		ethTx.to(),
		ethTx.value(),
		ethTx.data(),
		ethTx.chainID(), uint(0), uint(0),
	})

	logger.Debugf("ethTx.ethSigningHash: %v", ethSigningHash.Hex())

	v, r, s := ethTx.rawSignatureValues()
	sig, err := crypto.EncodeSignature(r, s, v)
	if err != nil {
		return nil, err
	}

	logger.Debugf("ethTx.signature: %v", sig.ToBytes().String())

	fromAddr, err := crypto.HomesteadSignerSender(ethSigningHash, sig)
	if err != nil {
		return nil, err
	}

	logger.Debugf("ethTx.recoveredFromAddress: %v", fromAddr.Hex())

	coins := Coins{
		ThetaWei: big.NewInt(0),
		TFuelWei: ethTx.value(),
	}

	from := TxInput{
		Address:   fromAddr,
		Coins:     coins,
		Sequence:  ethTx.nonce() + 1, // off-by-one, ETH tx nonce starts from 0, while Theta tx sequence starts from 1
		Signature: sig,
	}

	if ethTx.to() == nil {
		//return nil, errors.New("To address is nil")
		ethTx.To = &common.Address{}
	}

	to := TxOutput{
		Address: *ethTx.to(),
		Coins:   NewCoins(0, 0),
	}

	thetaTx := SmartContractTx{
		From:     from,
		To:       to,
		GasLimit: ethTx.gas(),
		GasPrice: ethTx.gasPrice(),
		Data:     ethTx.data(),
	}

	return &thetaTx, nil
}

// TxData is the underlying data of a transaction.
//
// This is implemented by EthTransaction and AccessListTx.
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

// EthTransaction is the transaction data of regular Ethereum transactions.
type EthTransaction struct {
	Nonce    uint64          // nonce of sender account
	GasPrice *big.Int        // wei per gas
	Gas      uint64          // gas limit
	To       *common.Address `rlp:"nil"` // nil means contract creation
	Value    *big.Int        // wei amount
	Data     []byte          // contract invocation input data
	V, R, S  *big.Int        // signature values
}

// copy creates a deep copy of the transaction data and initializes all fields.
func (tx *EthTransaction) copy() TxData {
	cpy := &EthTransaction{
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

func (tx *EthTransaction) Hash() common.Hash {
	h := RLPHash(tx)
	return h
}

// accessors for innerTx.

func (tx *EthTransaction) chainID() *big.Int   { return crypto.DeriveEthChainId(tx.V) }
func (tx *EthTransaction) data() []byte        { return tx.Data }
func (tx *EthTransaction) gas() uint64         { return tx.Gas }
func (tx *EthTransaction) gasPrice() *big.Int  { return tx.GasPrice }
func (tx *EthTransaction) value() *big.Int     { return tx.Value }
func (tx *EthTransaction) nonce() uint64       { return tx.Nonce }
func (tx *EthTransaction) to() *common.Address { return tx.To }

func (tx *EthTransaction) rawSignatureValues() (v, r, s *big.Int) {
	return tx.V, tx.R, tx.S
}

func (tx *EthTransaction) setSignatureValues(chainID, v, r, s *big.Int) {
	tx.V, tx.R, tx.S = v, r, s
}
