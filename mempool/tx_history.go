package mempool

import (
	"container/list"
	"sync"

	"github.com/thetatoken/ukulele/crypto"
)

const maxCacheSize = uint(200000)

//
// txHistoryManager keeps tracks of recently seen transactions
//
type txHistoryManager struct {
	mutex *sync.Mutex

	txMap  map[string]bool // map: transaction hash -> bool
	txList list.List
}

func createTxHistoryManager() txHistoryManager {
	return txHistoryManager{
		mutex: &sync.Mutex{},
		txMap: make(map[string]bool, maxCacheSize),
	}
}

func (thm *txHistoryManager) reset() {
	thm.mutex.Lock()
	defer thm.mutex.Unlock()
	thm.txMap = make(map[string]bool, maxCacheSize)
	thm.txList.Init()
}

func (thm *txHistoryManager) exists(mptx *mempoolTransaction) bool {
	thm.mutex.Lock()
	defer thm.mutex.Unlock()
	txhash := getTransactionHash(mptx)
	_, exists := thm.txMap[txhash]
	return exists
}

func (thm *txHistoryManager) insert(mptx *mempoolTransaction) bool {
	thm.mutex.Lock()
	defer thm.mutex.Unlock()
	txhash := getTransactionHash(mptx)

	if _, exists := thm.txMap[txhash]; exists {
		return false
	}

	if uint(thm.txList.Len()) > maxCacheSize { // remove the oldest transactions
		popped := thm.txList.Front()
		poppedTxhash := popped.Value.(string)
		delete(thm.txMap, poppedTxhash)
		thm.txList.Remove(popped)
	}

	thm.txMap[txhash] = true
	thm.txList.PushBack(txhash)

	return true
}

func (thm *txHistoryManager) remove(mptx *mempoolTransaction) {
	thm.mutex.Lock()
	defer thm.mutex.Unlock()
	txhash := getTransactionHash(mptx)
	delete(thm.txMap, txhash)
}

func getTransactionHash(mptx *mempoolTransaction) string {
	txhash := crypto.Keccak256Hash(mptx.rawTransaction)
	txhashStr := string(txhash[:])
	return txhashStr
}
