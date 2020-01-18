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

package keystore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math/big"
	"reflect"

	"github.com/thetatoken/theta/rlp"

	"github.com/golang/protobuf/proto"
	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
	tp "github.com/thetatoken/theta/ledger/types"
	"github.com/thetatoken/theta/wallet/coldwallet/keystore/trezor"
	"github.com/thetatoken/theta/wallet/types"
)

const MAX_PASSPHRASE_LENGTH = 50

// trezorDriver implements the communication with a Trezor hardware wallet.
type trezorDriver struct {
	bridge  trezor.BridgeTransport
	ui      *trezor.TrezorUI
	device  io.ReadWriter // USB device connection to communicate through
	version [3]uint32     // Current version of the Trezor firmware
	label   string        // Current textual label of the Trezor device
	pinwait bool          // Flags whether the device is waiting for PIN entry
	failure error         // Any failure that would make the device unusable
}

// newTrezorDriver creates a new instance of a Trezor USB protocol driver.
func NewTrezorDriver() Driver {
	return &trezorDriver{bridge: trezor.BridgeTransport{}, ui: trezor.NewTrezorUI(false)}
}

// Status implements keystore.Driver, always whether the Trezor is opened, closed
// or whether the Theta app was not started on it.
func (w *trezorDriver) Status() (string, error) {
	if w.failure != nil {
		return fmt.Sprintf("Failed: %v", w.failure), w.failure
	}
	if w.device == nil {
		return "Closed", w.failure
	}
	if w.pinwait {
		return fmt.Sprintf("Trezor v%d.%d.%d '%s' waiting for PIN", w.version[0], w.version[1], w.version[2], w.label), w.failure
	}
	return fmt.Sprintf("Trezor v%d.%d.%d '%s' online", w.version[0], w.version[1], w.version[2], w.label), w.failure
}

// Open implements keystore.Driver, attempting to initialize the connection to
// the Trezor hardware wallet. Initializing the Trezor is a two or three phase operation:
func (w *trezorDriver) Open(device io.ReadWriter, passphrase string) error {
	err := w.bridge.BeginSession()
	if err != nil {
		return err
	}
	defer w.bridge.EndSession()

	initialize := &trezor.Initialize{}
	res, _, err := w.trezorExchange(initialize)
	if err != nil {
		return err
	}

	w.bridge.Features = res.(*trezor.Features)
	if w.bridge.Features.Vendor != "trezor.io" && w.bridge.Features.Vendor != "bitcointrezor.com" {
		return fmt.Errorf("Unsupported device")
	}

	w.version = [3]uint32{w.bridge.Features.GetMajorVersion(), w.bridge.Features.GetMinorVersion(), w.bridge.Features.GetPatchVersion()}
	w.label = w.bridge.Features.GetLabel()
	w.device, w.failure = device, nil
	return w.bridge.CheckFirmwareVersion(w.version, false)
}

// Close implements keystore.Driver, cleaning up and metadata maintained within
// the Trezor driver.
func (w *trezorDriver) Close() error {
	w.bridge.EndSession()
	w.version, w.label, w.pinwait = [3]uint32{}, "", false
	return nil
}

// Heartbeat implements keystore.Driver, performing a sanity check against the
// Trezor to see if it's still online.
func (w *trezorDriver) Heartbeat() error {
	_, _, err := w.trezorExchange(&trezor.Ping{})
	w.failure = err
	return err
}

// Derive implements keystore.Driver, sending a derivation request to the Trezor
// and returning the Theta address located on that derivation path.
func (w *trezorDriver) Derive(path types.DerivationPath) (common.Address, error) {
	return w.trezorDerive(path)
}

// SignTx implements keystore.Driver, sending the transaction to the Trezor and
// waiting for the user to confirm or deny the transaction.
func (w *trezorDriver) SignTx(path types.DerivationPath, txrlp common.Bytes) (common.Address, *crypto.Signature, error) {
	if w.device == nil {
		return common.Address{}, nil, errors.New("wallet closed")
	}

	return w.trezorSign(path, txrlp)
}

// trezorDerive sends a derivation request to the Trezor device and returns the
// Theta address located on that path.
func (w *trezorDriver) trezorDerive(derivationPath []uint32) (common.Address, error) {
	err := w.bridge.BeginSession()
	if err != nil {
		return common.Address{}, err
	}
	defer w.bridge.EndSession()

	request := &trezor.ThetaGetAddress{
		AddressN:    derivationPath,
		ShowDisplay: false,
	}
	res, msgType, err := w.trezorExchange(request)
	if err != nil {
		return common.Address{}, err
	}

	res, err = w.handleResponse(res, msgType, err)
	if err != nil {
		return common.Address{}, nil
	}
	resp := res.(*trezor.ThetaAddress)
	addr := common.Address{}

	copy(addr[:], common.Hex2Bytes(string(resp.Address)[2:]))
	return addr, nil
}

func (w *trezorDriver) trezorSignMsg(derivationPath []uint32, txrlp common.Bytes) (common.Address, *crypto.Signature, error) {
	err := w.bridge.BeginSession()
	if err != nil {
		return common.Address{}, nil, err
	}
	defer w.bridge.EndSession()

	request := &trezor.ThetaSignMessage{
		AddressN: derivationPath,
		Message:  txrlp,
	}

	res, msgType, err := w.trezorExchange(request)
	if err != nil {
		return common.Address{}, nil, err
	}
	res, err = w.handleResponse(res, msgType, err)
	if err != nil {
		return common.Address{}, nil, err
	}
	response := res.(*trezor.ThetaMessageSignature)
	responseSig := response.Signature
	if len(responseSig) != 65 {
		return common.Address{}, nil, errors.New("Signature should be 65 bytes long")
	}
	sigBytes := responseSig

	// Create the correct signer and signature
	signature, err := crypto.SignatureFromBytes(sigBytes)
	if err != nil {
		return common.Address{}, nil, err
	}

	sender, err := signature.RecoverSignerAddress(txrlp)
	logger.Infof("Sender address: %v", sender.Hex())

	if err != nil {
		return common.Address{}, nil, err
	}
	return sender, signature, nil
}

// trezorSign sends the transaction to the Trezor wallet, and waits for the user
// to confirm or deny the transaction.
func (w *trezorDriver) trezorSign(derivationPath []uint32, txrlp common.Bytes) (common.Address, *crypto.Signature, error) {
	err := w.bridge.BeginSession()
	if err != nil {
		return common.Address{}, nil, err
	}
	defer w.bridge.EndSession()

	tx := &tp.EthereumTxWrapper{}
	err = rlp.DecodeBytes(txrlp, tx)

	// Create the transaction initiation message
	data := tx.Payload
	length := uint32(len(data))

	request := &trezor.ThetaSignTx{
		AddressN: derivationPath,
		Nonce:    new(big.Int).SetUint64(0).Bytes(),
		GasPrice: new(big.Int).SetUint64(0).Bytes(),
		GasLimit: new(big.Int).SetUint64(0).Bytes(),
		To:       []byte("0000000000000000000000000000000000000000"),
		Value:    new(big.Int).SetUint64(0).Bytes(),
		// ChainId:    uint32(1),
		DataLength: length,
	}

	if length > 1024 { // Send the data chunked if that was requested
		request.DataInitialChunk, data = data[:1024], data[1024:]
	} else {
		request.DataInitialChunk, data = data, nil
	}

	res, msgType, err := w.trezorExchange(request)
	if err != nil {
		return common.Address{}, nil, err
	}

	res, err = w.handleResponse(res, msgType, err)
	if err != nil {
		return common.Address{}, nil, err
	}
	response := res.(*trezor.ThetaTxRequest)

	for response.DataLength != 0 && int(response.DataLength) <= len(data) {
		chunk := data[:response.DataLength]
		data = data[response.DataLength:]

		request := &trezor.ThetaTxAck{DataChunk: chunk}
		res, _, err := w.trezorExchange(request)
		if err != nil {
			return common.Address{}, nil, err
		}
		response = res.(*trezor.ThetaTxRequest)
	}

	// Extract the Theta signature and do a sanity validation
	if len(response.GetSignatureR()) == 0 || len(response.GetSignatureS()) == 0 || response.GetSignatureV() == 0 {
		return common.Address{}, nil, errors.New("reply lacks signature")
	}
	sigBytes := append(append(response.GetSignatureR(), response.GetSignatureS()...), byte(response.GetSignatureV()))

	if len(sigBytes) != 65 {
		return common.Address{}, nil, errors.New("Signature bytes should be 65 bytes lone")
	}
	sigBytes[64] -= byte(27)

	// Create the correct signer and signature
	signature, err := crypto.SignatureFromBytes(sigBytes)
	if err != nil {
		return common.Address{}, nil, err
	}

	sender, err := signature.RecoverSignerAddress(txrlp)
	logger.Infof("Sender address: %v", sender.Hex())

	if err != nil {
		return common.Address{}, nil, err
	}
	return sender, signature, nil
}

func (w *trezorDriver) handleResponse(res interface{}, msgType trezor.MessageType, err error) (interface{}, error) {
	for {
		if msgType == trezor.MessageType_MessageType_PinMatrixRequest {
			response := res.(*trezor.PinMatrixRequest)
			res, msgType, err = w.callbackPin(response)
			if err != nil {
				return nil, err
			}
		} else if msgType == trezor.MessageType_MessageType_PassphraseRequest {
			response := res.(*trezor.PassphraseRequest)
			res, msgType, err = w.callbackPassphrase(response)
			if err != nil {
				return nil, err
			}
		} else if msgType == trezor.MessageType_MessageType_ButtonRequest {
			response := res.(*trezor.ButtonRequest)
			res, msgType, err = w.callbackButton(response)
			if err != nil {
				return nil, err
			}
		} else if msgType == trezor.MessageType_MessageType_Failure {
			response := res.(*trezor.Failure)
			if response.Code == trezor.FailureType_Failure_ActionCancelled {
				return nil, nil //TODO
			}
			return nil, fmt.Errorf("Failed to sign tx, %v", response.Message)
		} else {
			break
		}
	}
	return res, nil
}

func (w *trezorDriver) callbackButton(msg *trezor.ButtonRequest) (interface{}, trezor.MessageType, error) {
	request := &trezor.ButtonAck{}
	err := w.trezorWrite(request)
	if err != nil {
		return nil, 0, err
	}
	w.ui.ButtonRequest()
	return w.trezorRead()
}

func (w *trezorDriver) callbackPin(msg *trezor.PinMatrixRequest) (interface{}, trezor.MessageType, error) {
	pin := w.ui.GetPin(msg.Type)
	// except exceptions.Cancelled:
	//     self.call_raw(messages.Cancel())

	request := &trezor.PinMatrixAck{Pin: string(pin)}
	res, msgType, err := w.trezorExchange(request)
	if err != nil {
		return nil, 0, err
	}
	if msgType == trezor.MessageType_MessageType_Failure {
		response := res.(*trezor.Failure)
		//and resp.code in (
		// messages.FailureType.PinInvalid,
		// messages.FailureType.PinCancelled,
		// messages.FailureType.PinExpected,
		return nil, 0, fmt.Errorf("Pin failed (%v), %v", response.Code, response.Message)
	}
	return res, msgType, nil
}

func (w *trezorDriver) callbackPassphrase(msg *trezor.PassphraseRequest) (interface{}, trezor.MessageType, error) {
	// if msg.on_device:
	//     passphrase = None
	// else:

	passphrase := w.ui.GetPassphrase()
	// except exceptions.Cancelled:
	//     self.call_raw(messages.Cancel())

	// passphrase = Mnemonic.normalize_string(passphrase)

	if len(passphrase) > MAX_PASSPHRASE_LENGTH {
		w.trezorWrite(&trezor.Cancel{})
		return nil, 0, fmt.Errorf("Passphrase too long")
	}

	return w.trezorExchange(&trezor.PassphraseAck{Passphrase: passphrase})

	// if isinstance(resp, messages.PassphraseStateRequest):
	//     self.state = resp.state
	//     return self.call_raw(messages.PassphraseStateAck())
	// else:
	//     return resp
}

func getMessageName(v interface{}) string {
	var name string
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		name = t.Elem().Name()
	} else {
		name = t.Name()
	}
	return "MessageType_" + name
}

func (w *trezorDriver) trezorExchange(request proto.Message) (interface{}, trezor.MessageType, error) {
	var b bytes.Buffer
	trezor.DumpMessage(io.Writer(&b), request)
	data := b.Bytes()
	tname := getMessageName(request)

	var header [6]byte
	trezor.Pack(&header, uint16(trezor.MessageType_value[tname]), uint32(len(data)))
	data = append(header[:], data...)

	return w.bridge.CallRaw(data)
}

func (w *trezorDriver) trezorWrite(request proto.Message) error {
	var b bytes.Buffer
	trezor.DumpMessage(io.Writer(&b), request)
	data := b.Bytes()
	tname := getMessageName(request)

	var header [6]byte
	trezor.Pack(&header, uint16(trezor.MessageType_value[tname]), uint32(len(data)))
	data = append(header[:], data...)

	return w.bridge.WriteRaw(data)
}

func (w *trezorDriver) trezorRead() (interface{}, trezor.MessageType, error) {
	return w.bridge.ReadRaw()
}
