package rpc

import (
	"encoding/hex"
	"net/http"

	"github.com/thetatoken/ukulele/crypto"
)

// ------------------------------- BroadcastRawTransaction -----------------------------------

type BroadcastRawTransactionArgs struct {
	TxBytes string `json:"tx_bytes"`
}

type BroadcastRawTransactionResult struct {
	TxHash string `json:"hash"`
}

func (t *ThetaRPCServer) BroadcastRawTransaction(r *http.Request, args *BroadcastRawTransactionArgs, result *BroadcastRawTransactionResult) (err error) {
	txBytes, err := hex.DecodeString(args.TxBytes)
	if err != nil {
		return err
	}

	hash := crypto.Keccak256Hash(txBytes)
	result.TxHash = hash.Hex()

	return t.mempool.InsertTransaction(txBytes)
}
