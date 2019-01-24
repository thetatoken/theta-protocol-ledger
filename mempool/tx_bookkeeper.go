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

	txMap  map[string]bool // map: transaction hash -> bool
	txList list.List       // FIFO list of transaction hashes

	maxNumTxs uint
}

func createTransactionBookkeeper(maxNumTxs uint) transactionBookkeeper {
	return transactionBookkeeper{
		mutex:     &sync.Mutex{},
		txMap:     make(map[string]bool),
		maxNumTxs: maxNumTxs,
	}
}

func (tb *transactionBookkeeper) reset() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.txMap = make(map[string]bool)
	tb.txList.Init()
}

func (tb *transactionBookkeeper) hasSeen(rawTx common.Bytes) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txhash := getTransactionHash(rawTx)
	_, exists := tb.txMap[txhash]
	return exists
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

	tb.txMap[txhash] = true
	tb.txList.PushBack(txhash)

	return true
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
