package rpc

import (
	"encoding/hex"
	"net/http"

	"github.com/thetatoken/ukulele/mempool"
)

// ------------------------------- BroadcastRawTransaction -----------------------------------

type BroadcastRawTransactionArgs struct {
	TxBytes string `json:"tx_bytes"`
}

type BroadcastRawTransactionResult struct{}

func (t *ThetaRPCServer) BroadcastRawTransaction(r *http.Request, args *BroadcastRawTransactionArgs, result *BroadcastRawTransactionResult) (err error) {
	txBytes, err := hex.DecodeString(args.TxBytes)
	if err != nil {
		return err
	}
	return t.mempool.InsertTransaction(mempool.CreateMempoolTransaction(txBytes))
}
