package mempool

import (
	"container/list"
	"encoding/hex"
	"sync"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

const defaultMaxNumTxs = uint(200000)

//
// transactionBookkeeper keeps tracks of recently seen transactions
//
type transactionBookkeeper struct {
	mutex *sync.Mutex

	txMap  map[string]TxStatus // map: transaction hash -> bool
	txList list.List           // FIFO list of transaction hashes

	maxNumTxs uint
}

type TxStatus int

const (
	TxStatusPending TxStatus = iota
	TxStatusAbandoned
)

func createTransactionBookkeeper(maxNumTxs uint) transactionBookkeeper {
	return transactionBookkeeper{
		mutex:     &sync.Mutex{},
		txMap:     make(map[string]TxStatus),
		maxNumTxs: maxNumTxs,
	}
}

func (tb *transactionBookkeeper) reset() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.txMap = make(map[string]TxStatus)
	tb.txList.Init()
}

func (tb *transactionBookkeeper) hasSeen(rawTx common.Bytes) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txhash := getTransactionHash(rawTx)
	_, exists := tb.txMap[txhash]
	return exists
}

// getStatus returns a tx status and a boolean of whether the tx is known.
func (tb *transactionBookkeeper) getStatus(txhash string) (TxStatus, bool) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txStatus, ok := tb.txMap[txhash]
	return txStatus, ok
}

func (tb *transactionBookkeeper) record(rawTx common.Bytes) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txhash := getTransactionHash(rawTx)

	if _, exists := tb.txMap[txhash]; exists {
		return false
	}

	if uint(tb.txList.Len()) >= tb.maxNumTxs { // remove the oldest transactions
		popped := tb.txList.Front()
		poppedTxhash := popped.Value.(string)
		delete(tb.txMap, poppedTxhash)
		tb.txList.Remove(popped)
	}

	tb.txMap[txhash] = TxStatusPending
	tb.txList.PushBack(txhash)

	return true
}

func (tb *transactionBookkeeper) markAbandoned(rawTx common.Bytes) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	txhash := getTransactionHash(rawTx)
	if _, exists := tb.txMap[txhash]; !exists {
		return
	}
	tb.txMap[txhash] = TxStatusAbandoned
}

func (tb *transactionBookkeeper) remove(rawTx common.Bytes) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txhash := getTransactionHash(rawTx)
	delete(tb.txMap, txhash)
}

func getTransactionHash(rawTx common.Bytes) string {
	txhash := crypto.Keccak256Hash(rawTx)
	txhashStr := hex.EncodeToString(txhash[:])
	return txhashStr
}
