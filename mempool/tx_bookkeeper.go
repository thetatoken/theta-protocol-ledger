package mempool

import (
	"container/list"
	"encoding/hex"
	"sync"
	"time"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/crypto"
)

const defaultMaxNumTxs = uint(200000)

const maxTxLife = 1 * time.Minute

//
// transactionBookkeeper keeps tracks of recently seen transactions
//
type transactionBookkeeper struct {
	mutex *sync.Mutex

	txMap  map[string]*TxRecord // map: transaction hash -> bool
	txList list.List            // FIFO list of transaction hashes

	maxNumTxs uint
}

type TxRecord struct {
	Hash      string
	Status    TxStatus
	CreatedAt time.Time
}

func (r *TxRecord) IsOutdated() bool {
	return time.Since(r.CreatedAt) > maxTxLife
}

type TxStatus int

const (
	TxStatusPending TxStatus = iota
	TxStatusAbandoned
)

func createTransactionBookkeeper(maxNumTxs uint) transactionBookkeeper {
	return transactionBookkeeper{
		mutex:     &sync.Mutex{},
		txMap:     make(map[string]*TxRecord),
		maxNumTxs: maxNumTxs,
	}
}

func (tb *transactionBookkeeper) reset() {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	tb.txMap = make(map[string]*TxRecord)
	tb.txList.Init()
}

func (tb *transactionBookkeeper) hasSeen(rawTx common.Bytes) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Remove outdated Tx records
	tb.removeOutdatedTxsUnsafe()

	txhash := getTransactionHash(rawTx)
	_, exists := tb.txMap[txhash]
	return exists
}

// getStatus returns a tx status and a boolean of whether the tx is known.
func (tb *transactionBookkeeper) getStatus(txhash string) (TxStatus, bool) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	// Remove outdated Tx records
	tb.removeOutdatedTxsUnsafe()

	txRecord, exists := tb.txMap[txhash]

	if !exists {
		return TxStatusAbandoned, false
	}
	return txRecord.Status, true
}

func (tb *transactionBookkeeper) removeOutdatedTxsUnsafe() {
	// Loop and remove all outdated Tx records
	for {
		el := tb.txList.Front()
		if el == nil {
			return
		}
		txRecord := el.Value.(*TxRecord)
		if !txRecord.IsOutdated() {
			return
		}

		if _, exists := tb.txMap[txRecord.Hash]; exists {
			delete(tb.txMap, txRecord.Hash)
		}
		tb.txList.Remove(el)
	}
}

func (tb *transactionBookkeeper) record(rawTx common.Bytes) bool {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()
	txhash := getTransactionHash(rawTx)

	// Remove outdated Tx records
	tb.removeOutdatedTxsUnsafe()

	if _, exists := tb.txMap[txhash]; exists {
		return false
	}

	if uint(tb.txList.Len()) >= tb.maxNumTxs { // remove the oldest transactions
		popped := tb.txList.Front()
		poppedTxhash := popped.Value.(*TxRecord).Hash
		delete(tb.txMap, poppedTxhash)
		tb.txList.Remove(popped)
	}

	record := &TxRecord{
		Hash:      txhash,
		Status:    TxStatusPending,
		CreatedAt: time.Now(),
	}
	tb.txMap[txhash] = record

	tb.txList.PushBack(record)

	return true
}

func (tb *transactionBookkeeper) markAbandoned(rawTx common.Bytes) {
	tb.mutex.Lock()
	defer tb.mutex.Unlock()

	txhash := getTransactionHash(rawTx)
	if _, exists := tb.txMap[txhash]; !exists {
		return
	}
	tb.txMap[txhash].Status = TxStatusAbandoned
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
