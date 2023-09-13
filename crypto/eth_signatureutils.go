package crypto

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/thetatoken/theta/common"

	log "github.com/sirupsen/logrus"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "crypto"})

// ----------------------- ETH signature utils ----------------------- //

func isProtectedV(V *big.Int) bool {
	if V.BitLen() <= 8 {
		v := V.Uint64()
		return v != 27 && v != 28 && v != 1 && v != 0
	}
	// anything not 27 or 28 is considered protected
	return true
}

// DeriveEthChainId derives the chain id from the given v parameter
func DeriveEthChainId(v *big.Int) *big.Int {
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

func EncodeSignature(R, S, Vb *big.Int) (*Signature, error) {
	VAdj := adjustV(Vb)
	if VAdj.BitLen() > 8 {
		return nil, errors.New("invalid v, r, s values")
	}
	V := byte(VAdj.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, true) {
		return nil, errors.New("invalid v, r, s values")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sigBytes := make([]byte, SignatureLength)
	copy(sigBytes[32-len(r):32], r)
	copy(sigBytes[64-len(s):64], s)
	sigBytes[64] = V + 27

	sig, err := SignatureFromBytes(sigBytes)
	if err != nil {
		return nil, err
	}

	return sig, nil
}

func DecodeSignature(sig *Signature) (r, s, v *big.Int) {
	sigBytes := sig.ToBytes()
	if len(sigBytes) != SignatureLength {
		panic(fmt.Sprintf("wrong size for signature: got %d, want %d", len(sigBytes), SignatureLength))
	}
	r = new(big.Int).SetBytes(sigBytes[:32])
	s = new(big.Int).SetBytes(sigBytes[32:64])
	v = new(big.Int).SetBytes([]byte{sigBytes[64]})
	//v = new(big.Int).SetBytes([]byte{sigBytes[64] + 27})
	return r, s, v
}

func recoverPlain(signingHash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, errors.New("invalid transaction v, r, s values")
	}
	// encode the signature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, SignatureLength)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := Ecrecover(signingHash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], Keccak256(pub[1:])[12:])
	return addr, nil
}

func HomesteadSignerSender(signingHash common.Hash, sig *Signature) (common.Address, error) {
	r, s, v := DecodeSignature(sig)
	vadj := adjustV(v)

	logger.Debugf("ethTx.DecodeSignature: r = %v, s = %v, v = %v", r, s, v)
	logger.Debugf("ethTx.DecodeSignature: vadj = %v", vadj)

	return recoverPlain(signingHash, r, s, vadj, true)
}

func ValidateEthSignature(sender common.Address, signingHash common.Hash, sig *Signature) error {
	recoveredSender, err := HomesteadSignerSender(signingHash, sig)
	if err != nil {
		return err
	}

	logger.Debugf("ethTx.ValidateEthSignature: recoveredSender = %v, sender = %v, signingHash = %v, signature = %v",
		recoveredSender.Hex(), sender.Hex(), signingHash.Hex(), sig.ToBytes().String())

	if recoveredSender != sender {
		return errors.New(fmt.Sprintf("Recovered sender mismatch, recovered sender: %v, sender: %v", recoveredSender.Hex(), sender.Hex()))
	}

	return nil
}

// References:
// https://github.com/ethereum/go-ethereum/blob/087ed9c92ecfe41109c1e039693fc126952a3718/core/types/transaction_signing.go#L263
// https://github.com/ethereum/go-ethereum/blob/087ed9c92ecfe41109c1e039693fc126952a3718/core/types/transaction_signing.go#L344
// https://github.com/ethereum/go-ethereum/blob/087ed9c92ecfe41109c1e039693fc126952a3718/core/types/transaction_signing.go#L370
func adjustV(v *big.Int) *big.Int {
	if isProtectedV(v) {
		chainID := DeriveEthChainId(v)
		chainIDMul := new(big.Int).Mul(chainID, big.NewInt(2))
		v = new(big.Int).Sub(v, chainIDMul)
		v = v.Sub(v, big.NewInt(8))
	}
	return v
}
