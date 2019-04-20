// Adapted for Theta
// Copyright 2017 The go-ethereum Authors
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

// This file contains the implementation for interacting with the Trezor hardware
// wallets. The wire protocol spec can be found on the SatoshiLabs website:
// https://doc.satoshilabs.com/trezor-tech/api-protobuf.html

//go:generate protoc --go_out=import_path=trezor:. types.proto messages.proto

// Package trezor contains the wire protocol wrapper in Go.
package trezor

import (
	"reflect"
	"strings"

	"github.com/golang/protobuf/proto"
)

// Type returns the protocol buffer type number of a specific message. If the
// message is nil, this method panics!
func Type(msg proto.Message) uint16 {
	return uint16(MessageType_value["MessageType_"+reflect.TypeOf(msg).Elem().Name()])
}

// Name returns the friendly message type name of a specific protocol buffer
// type number.
func Name(kind uint16) string {
	name := MessageType_name[int32(kind)]
	if len(name) < 12 {
		return name
	}
	return name[12:]
}

func GetEmptyObj(msgType MessageType) interface{} {
	tname := MessageType_name[int32(msgType)]
	tname = strings.Split(tname, "_")[1]
	tname = "trezor." + tname

	t := proto.MessageType(tname).Elem()

	switch msgType {
	case 0:
		v := reflect.Indirect(reflect.New(t)).Interface().(Initialize)
		return &v
	case 1:
		v := reflect.Indirect(reflect.New(t)).Interface().(Ping)
		return &v
	case 2:
		v := reflect.Indirect(reflect.New(t)).Interface().(Success)
		return &v
	case 3:
		v := reflect.Indirect(reflect.New(t)).Interface().(Failure)
		return &v
	case 4:
		v := reflect.Indirect(reflect.New(t)).Interface().(ChangePin)
		return &v
	case 5:
		v := reflect.Indirect(reflect.New(t)).Interface().(WipeDevice)
		return &v
	case 6:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareErase)
		return &v
	case 7:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareUpload)
		return &v
	case 8:
		v := reflect.Indirect(reflect.New(t)).Interface().(FirmwareRequest)
		return &v
	case 9:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetEntropy)
		return &v
	case 10:
		v := reflect.Indirect(reflect.New(t)).Interface().(Entropy)
		return &v
	case 11:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetPublicKey)
		return &v
	case 12:
		v := reflect.Indirect(reflect.New(t)).Interface().(PublicKey)
		return &v
	case 13:
		v := reflect.Indirect(reflect.New(t)).Interface().(LoadDevice)
		return &v
	case 14:
		v := reflect.Indirect(reflect.New(t)).Interface().(ResetDevice)
		return &v
	case 15:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignTx)
		return &v
	case 16:
		v := reflect.Indirect(reflect.New(t)).Interface().(SimpleSignTx)
		return &v
	case 17:
		v := reflect.Indirect(reflect.New(t)).Interface().(Features)
		return &v
	case 18:
		v := reflect.Indirect(reflect.New(t)).Interface().(PinMatrixRequest)
		return &v
	case 19:
		v := reflect.Indirect(reflect.New(t)).Interface().(PinMatrixAck)
		return &v
	case 20:
		v := reflect.Indirect(reflect.New(t)).Interface().(Cancel)
		return &v
	case 21:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxRequest)
		return &v
	case 22:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxAck)
		return &v
	case 23:
		v := reflect.Indirect(reflect.New(t)).Interface().(CipherKeyValue)
		return &v
	case 24:
		v := reflect.Indirect(reflect.New(t)).Interface().(ClearSession)
		return &v
	case 25:
		v := reflect.Indirect(reflect.New(t)).Interface().(ApplySettings)
		return &v
	case 26:
		v := reflect.Indirect(reflect.New(t)).Interface().(ButtonRequest)
		return &v
	case 27:
		v := reflect.Indirect(reflect.New(t)).Interface().(ButtonAck)
		return &v
	case 28:
		v := reflect.Indirect(reflect.New(t)).Interface().(ApplyFlags)
		return &v
	case 29:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetAddress)
		return &v
	case 30:
		v := reflect.Indirect(reflect.New(t)).Interface().(Address)
		return &v
	case 32:
		v := reflect.Indirect(reflect.New(t)).Interface().(SelfTest)
		return &v
	case 34:
		v := reflect.Indirect(reflect.New(t)).Interface().(BackupDevice)
		return &v
	case 35:
		v := reflect.Indirect(reflect.New(t)).Interface().(EntropyRequest)
		return &v
	case 36:
		v := reflect.Indirect(reflect.New(t)).Interface().(EntropyAck)
		return &v
	case 38:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignMessage)
		return &v
	case 39:
		v := reflect.Indirect(reflect.New(t)).Interface().(VerifyMessage)
		return &v
	case 40:
		v := reflect.Indirect(reflect.New(t)).Interface().(MessageSignature)
		return &v
	case 41:
		v := reflect.Indirect(reflect.New(t)).Interface().(PassphraseRequest)
		return &v
	case 42:
		v := reflect.Indirect(reflect.New(t)).Interface().(PassphraseAck)
		return &v
	case 43:
		v := reflect.Indirect(reflect.New(t)).Interface().(EstimateTxSize)
		return &v
	case 44:
		v := reflect.Indirect(reflect.New(t)).Interface().(TxSize)
		return &v
	case 45:
		v := reflect.Indirect(reflect.New(t)).Interface().(RecoveryDevice)
		return &v
	case 46:
		v := reflect.Indirect(reflect.New(t)).Interface().(WordRequest)
		return &v
	case 47:
		v := reflect.Indirect(reflect.New(t)).Interface().(WordAck)
		return &v
	case 48:
		v := reflect.Indirect(reflect.New(t)).Interface().(CipheredKeyValue)
		return &v
	case 49:
		v := reflect.Indirect(reflect.New(t)).Interface().(EncryptMessage)
		return &v
	case 50:
		v := reflect.Indirect(reflect.New(t)).Interface().(EncryptedMessage)
		return &v
	case 51:
		v := reflect.Indirect(reflect.New(t)).Interface().(DecryptMessage)
		return &v
	case 52:
		v := reflect.Indirect(reflect.New(t)).Interface().(DecryptedMessage)
		return &v
	case 53:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignIdentity)
		return &v
	case 54:
		v := reflect.Indirect(reflect.New(t)).Interface().(SignedIdentity)
		return &v
	case 55:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetFeatures)
		return &v
	case 56:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaGetAddress)
		return &v
	case 57:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaAddress)
		return &v
	case 58:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaSignTx)
		return &v
	case 59:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaTxRequest)
		return &v
	case 60:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaTxAck)
		return &v
	case 61:
		v := reflect.Indirect(reflect.New(t)).Interface().(GetECDHSessionKey)
		return &v
	case 62:
		v := reflect.Indirect(reflect.New(t)).Interface().(ECDHSessionKey)
		return &v
	case 63:
		v := reflect.Indirect(reflect.New(t)).Interface().(SetU2FCounter)
		return &v
	case 64:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaSignMessage)
		return &v
	case 65:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaVerifyMessage)
		return &v
	case 66:
		v := reflect.Indirect(reflect.New(t)).Interface().(ThetaMessageSignature)
		return &v
	case 100:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkDecision)
		return &v
	case 101:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkGetState)
		return &v
	case 102:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkState)
		return &v
	case 103:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkStop)
		return &v
	case 104:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkLog)
		return &v
	case 110:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemoryRead)
		return &v
	case 111:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemory)
		return &v
	case 112:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkMemoryWrite)
		return &v
	case 113:
		v := reflect.Indirect(reflect.New(t)).Interface().(DebugLinkFlashErase)
		return &v
	}
	return nil
}
