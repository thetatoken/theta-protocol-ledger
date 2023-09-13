package mempool

import (
	"context"
	"encoding/hex"
	"errors"
	"math/big"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/thetatoken/theta/common"
	"github.com/thetatoken/theta/common/clist"
	"github.com/thetatoken/theta/common/math"
	"github.com/thetatoken/theta/common/pqueue"
	"github.com/thetatoken/theta/common/result"
	"github.com/thetatoken/theta/consensus"
	"github.com/thetatoken/theta/core"
	dp "github.com/thetatoken/theta/dispatcher"
)

var logger *log.Entry = log.WithFields(log.Fields{"prefix": "mempool"})

type MempoolError string

func (m MempoolError) Error() string {
	return string(m)
}

const DuplicateTxError = MempoolError("Transaction already seen")
const FastsyncSkipTxError = MempoolError("Skip tx during fastsync")

const MaxMempoolTxCount int = 25600

//
// mempoolTransaction implements the pqueue.Element interface
//
type mempoolTransaction struct {
	index          int
	rawTransaction common.Bytes
	txInfo         *core.TxInfo
}

var _ pqueue.Element = (*mempoolTransaction)(nil)

func (mt *mempoolTransaction) Priority() *big.Int {
	seq := new(big.Int).SetUint64(mt.txInfo.Sequence)
	return seq.Neg(seq)
}

func (mt *mempoolTransaction) SetIndex(index int) {
	mt.index = index
}

func (mt *mempoolTransaction) GetIndex() int {
	return mt.index
}

func createMempoolTransaction(rawTransaction common.Bytes, txInfo *core.TxInfo) *mempoolTransaction {
	return &mempoolTransaction{
		rawTransaction: rawTransaction,
		txInfo:         txInfo,
	}
}

//
// mempoolTransactionGroup holds a sequenece of transactions from one account. We sort transaction groups by the priority of
// their lowest sequence transaction.
//
type mempoolTransactionGroup struct {
	address common.Address
	txs     *pqueue.PriorityQueue
	index   int
}

var _ pqueue.Element = (*mempoolTransactionGroup)(nil)

func (mtg *mempoolTransactionGroup) Priority() *big.Int {
	if mtg.IsEmpty() {
		return new(big.Int).SetInt64(-1)
	}
	return mtg.txs.Peek().(*mempoolTransaction).txInfo.EffectiveGasPrice
}

func (mtg *mempoolTransactionGroup) SetIndex(index int) {
	mtg.index = index
}

func (mtg *mempoolTransactionGroup) GetIndex() int {
	return mtg.index
}

func (mtg *mempoolTransactionGroup) AddTx(rawTx common.Bytes, txInfo *core.TxInfo) {
	mpx := createMempoolTransaction(rawTx, txInfo)
	mtg.txs.Push(mpx)
}

func (mtg *mempoolTransactionGroup) PopTx() (common.Bytes, *core.TxInfo) {
	mptx := mtg.txs.Pop().(*mempoolTransaction)
	return mptx.rawTransaction, mptx.txInfo
}

func (mtg *mempoolTransactionGroup) IsEmpty() bool {
	return mtg.txs.IsEmpty()
}

// RemoveTxs removes matching Txs from transaction group. Returns number of Txs removed.
func (mtg *mempoolTransactionGroup) RemoveTxs(committedRawTxMap map[string]bool) (numRemoved int) {
	elementList := mtg.txs.ElementList()
	elemsTobeRemoved := []pqueue.Element{}
	for _, elem := range *elementList {
		mptx := elem.(*mempoolTransaction)
		rawTx := mptx.rawTransaction
		if _, exists := committedRawTxMap[string(rawTx[:])]; exists {
			elemsTobeRemoved = append(elemsTobeRemoved, elem)
			logger.Debugf("tx to be removed: %v, txInfo: %v", hex.EncodeToString(rawTx), mptx.txInfo)
		}
	}
	for _, elem := range elemsTobeRemoved {
		mtg.txs.Remove(elem.GetIndex())
		numRemoved++
	}
	return
}

func createMempoolTransactionGroup(rawTx common.Bytes, txInfo *core.TxInfo) *mempoolTransactionGroup {
	txGroup := &mempoolTransactionGroup{
		address: txInfo.Address,
		txs:     pqueue.CreatePriorityQueue(),
	}
	txGroup.AddTx(rawTx, txInfo)
	return txGroup
}

//
// Mempool manages the transactions submitted by the clients
// or relayed from peers
//
type Mempool struct {
	mutex *sync.Mutex

	consensus  *consensus.ConsensusEngine
	ledger     core.Ledger
	dispatcher *dp.Dispatcher

	newTxs           *clist.CList          // new transactions, to be gossiped to other nodes
	candidateTxs     *pqueue.PriorityQueue // candidate transactions for new block assembly, ordered by the transaction fee (high to low)
	txBookeepper     transactionBookkeeper
	addressToTxGroup map[common.Address]*mempoolTransactionGroup
	size             int

	// Life cycle
	wg      *sync.WaitGroup
	quit    chan struct{}
	ctx     context.Context
	cancel  context.CancelFunc
	stopped bool
}

// CreateMempool creates an instance of Mempool
func CreateMempool(dispatcher *dp.Dispatcher, engine *consensus.ConsensusEngine) *Mempool {
	return &Mempool{
		mutex:            &sync.Mutex{},
		consensus:        engine,
		dispatcher:       dispatcher,
		newTxs:           clist.New(),
		candidateTxs:     pqueue.CreatePriorityQueue(),
		addressToTxGroup: make(map[common.Address]*mempoolTransactionGroup),
		txBookeepper:     createTransactionBookkeeper(defaultMaxNumTxs),
		wg:               &sync.WaitGroup{},
	}
}

// SetLedger sets the ledger for the mempool
func (mp *Mempool) SetLedger(ledger core.Ledger) {
	mp.ledger = ledger
}

// InsertTransaction inserts the incoming transaction to mempool (submitted by the clients or relayed from peers)
func (mp *Mempool) InsertTransaction(rawTx common.Bytes) error {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	if mp.txBookeepper.hasSeen(rawTx) {
		logger.Debugf("Transaction already seen: %v, hash: 0x%v",
			hex.EncodeToString(rawTx), getTransactionHash(rawTx))
		return DuplicateTxError
	}

	// if mp.size >= MaxMempoolTxCount {
	// 	logger.Debugf("Mempool is full")
	// 	return errors.New("mempool is full, please submit your transaction again later")
	// }

	var txInfo *core.TxInfo
	var checkTxRes result.Result

	// Delay tx verification when in fast sync
	if mp.consensus.HasSynced() {
		txInfo, checkTxRes = mp.ledger.ScreenTx(rawTx)
		if !checkTxRes.IsOK() {
			logger.Debugf("Transaction screening failed, tx: %v, error: %v", hex.EncodeToString(rawTx), checkTxRes.Message)
			return errors.New(checkTxRes.Message)
		}

		// only record the transactions that passed the screening. This is because that
		// an invalid transaction could becoume valid later on. For example, assume expected
		// sequence for an account is 6. The account accidentally submits txA (seq = 7), got rejected.
		// He then submit txB(seq = 6), and then txA(seq = 7) again. For the second submission, txA
		// should not be rejected even though it has been submitted earlier.
		mp.txBookeepper.record(rawTx)

		txGroup, ok := mp.addressToTxGroup[txInfo.Address]
		if ok {
			txGroup.AddTx(rawTx, txInfo)
			mp.candidateTxs.Remove(txGroup.index) // Need to re-insert txGroup into queue since its priority could change.
		} else {
			txGroup = createMempoolTransactionGroup(rawTx, txInfo)
			mp.addressToTxGroup[txInfo.Address] = txGroup
		}
		mp.candidateTxs.Push(txGroup)
		logger.Debugf("rawTx: %v, txInfo: %v", hex.EncodeToString(rawTx), txInfo)
		logger.Infof("Insert tx, tx.hash: 0x%v", getTransactionHash(rawTx))
		mp.size++

		return nil
	}

	return FastsyncSkipTxError
}

// Start needs to be called when the Mempool starts
func (mp *Mempool) Start(ctx context.Context) error {
	c, cancel := context.WithCancel(ctx)
	mp.ctx = c
	mp.cancel = cancel

	return nil
}

// Stop needs to be called when the Mempool stops
func (mp *Mempool) Stop() {
	mp.cancel()
}

// Wait suspends the caller goroutine
func (mp *Mempool) Wait() {
	mp.wg.Wait()
}

// Lock is for the caller to lock/unlock the Mempool and perform safely update
func (mp *Mempool) Lock() {
	mp.mutex.Lock()
}

// Unlock is for the caller to lock/unlock the Mempool and perform safely update
func (mp *Mempool) Unlock() {
	mp.mutex.Unlock()
}

// Size returns the number of transactions in the Mempool
func (mp *Mempool) Size() int {
	return mp.size
}

// Reap returns a list of valid raw transactions and remove these
// transactions from the candidate pool. maxNumTxs == 0 means
// none, maxNumTxs < 0 means uncapped. Note that Reap does NOT remove
// the transactions from the candidateTxs list. Instead, the consensus engine needs
// to call the Mempool.Update() function to remove the committed transactions
// RUNTIME COMPLEXITY: k*log(n), where k is the number transactions to reap,
// and n is the number of transactions in the candidate pool.
func (mp *Mempool) Reap(maxNumTxs int) []common.Bytes {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	return mp.ReapUnsafe(maxNumTxs)
}

// ReapUnsafe is the non-locking version of Reap.
func (mp *Mempool) ReapUnsafe(maxNumTxs int) []common.Bytes {
	if maxNumTxs == 0 {
		return []common.Bytes{}
	} else if maxNumTxs < 0 {
		maxNumTxs = mp.Size()
	} else {
		maxNumTxs = math.MinInt(mp.Size(), maxNumTxs)
	}

	txs := make([]common.Bytes, 0, maxNumTxs)
	for i := 0; i < maxNumTxs; i++ {
		if mp.candidateTxs.IsEmpty() {
			break
		}
		txGroup := mp.candidateTxs.Pop().(*mempoolTransactionGroup)
		rawTx, txInfo := txGroup.PopTx()

		// Check for outdated txs
		txHash := getTransactionHash(rawTx)
		_, exists := mp.txBookeepper.getStatus(txHash)
		if exists {
			// Only add back Txs that has not been removed from bookkeeper due to timeout
			txs = append(txs, rawTx)
		}

		if txGroup.IsEmpty() {
			delete(mp.addressToTxGroup, txGroup.address)
		} else {
			mp.candidateTxs.Push(txGroup)
		}

		logger.Debugf("Reap tx: %v, txInfo: %v",
			hex.EncodeToString(rawTx), txInfo)
	}

	mp.size -= len(txs)

	return txs
}

// Update removes the committed transactions from the transaction candidate list
// RUNTIME COMPLEXITY: O(k + n), where k is the number committed raw transactions,
// and n is the number of transactions in the candidate pool.
func (mp *Mempool) Update(committedRawTxs []common.Bytes) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()
	mp.UpdateUnsafe(committedRawTxs)
}

// UpdateUnsafe is the non-locking version of Update. Caller must call Mempool.Lock() before
// calling this method.
func (mp *Mempool) UpdateUnsafe(committedRawTxs []common.Bytes) {
	start := time.Now()
	mp.removeTxs(committedRawTxs)
	removeCommittedTxTime := time.Since(start)

	// Remove Txs that have become obsolete.
	start = time.Now()
	count := 0
	invalidTxs := []common.Bytes{}
	txGroups := mp.candidateTxs.ElementList()
	for _, txGroupEl := range *txGroups {
		txGroup := txGroupEl.(*mempoolTransactionGroup)
		txs := txGroup.txs.ElementList()
		for _, txEl := range *txs {
			count++

			mempoolTx := txEl.(*mempoolTransaction)

			// Check for outdated txs
			txHash := getTransactionHash(mempoolTx.rawTransaction)
			_, exists := mp.txBookeepper.getStatus(txHash)
			if !exists {
				// Tx has been removed from bookkeeper due to timeout
				invalidTxs = append(invalidTxs, mempoolTx.rawTransaction)
				continue
			}

			checkTxRes := mp.ledger.ScreenTxUnsafe(mempoolTx.rawTransaction)
			if !checkTxRes.IsOK() {
				invalidTxs = append(invalidTxs, mempoolTx.rawTransaction)
				mp.txBookeepper.markAbandoned(mempoolTx.rawTransaction)
			}
		}
	}
	screenTxTime := time.Since(start)

	start = time.Now()
	mp.removeTxs(invalidTxs)
	removeInvalidTxTime := time.Since(start)

	logger.Debugf("UpdateUnsafe: %d tx screened in %v, removeCommittedTxTime = %v, removed %d obsolete Txs in %v: %v,", count, screenTxTime, removeCommittedTxTime, len(invalidTxs), removeInvalidTxTime, invalidTxs)
}

func (mp *Mempool) removeTxs(committedRawTxs []common.Bytes) {
	committedRawTxMap := make(map[string]bool)
	for _, rawtx := range committedRawTxs {
		committedRawTxMap[string(rawtx)] = true
	}

	elementList := mp.candidateTxs.ElementList()
	elemsTobeRemoved := []pqueue.Element{}
	for _, elem := range *elementList {
		txGroup := elem.(*mempoolTransactionGroup)
		numRemoved := txGroup.RemoveTxs(committedRawTxMap)
		mp.size -= numRemoved
		if txGroup.IsEmpty() {
			delete(mp.addressToTxGroup, txGroup.address)
			elemsTobeRemoved = append(elemsTobeRemoved, txGroup)
		}
	}

	// Note after each iteration, the indices of the elems in the priority queue
	// could change. So we need elem.GetIndex() to return the updated index
	for _, elem := range elemsTobeRemoved {
		mp.candidateTxs.Remove(elem.GetIndex())
	}
}

func (mp *Mempool) GetTransactionStatus(hash string) (TxStatus, bool) {
	return mp.txBookeepper.getStatus(hash)
}

// GetCandidateTransactions returns all the currently candidate transactions
func (mp *Mempool) GetCandidateTransactionHashes() []string {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	txHashes := []string{}
	txgElemList := mp.candidateTxs.ElementList()
	for _, txgElem := range *txgElemList {
		txg := txgElem.(*mempoolTransactionGroup)
		txElemList := txg.txs.ElementList()
		for _, txElem := range *txElemList {
			tx := txElem.(*mempoolTransaction)
			rawTx := tx.rawTransaction
			txHash := "0x" + getTransactionHash(rawTx)
			txHashes = append(txHashes, txHash)
		}
	}

	return txHashes
}

// Flush removes all transactions from the Mempool and the transactionBookkeeper
func (mp *Mempool) Flush() {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.txBookeepper.reset()

	for !mp.candidateTxs.IsEmpty() {
		mp.candidateTxs.Pop()
	}
	mp.size = 0
}

// BroadcastTx broadcast given raw transaction to the network
func (mp *Mempool) BroadcastTx(tx common.Bytes) {
	mp.mutex.Lock()
	defer mp.mutex.Unlock()

	mp.BroadcastTxUnsafe(tx)
}

// BroadcastTxUnsafe is the non-locking version of BroadcastTx
func (mp *Mempool) BroadcastTxUnsafe(tx common.Bytes) {
	data := dp.DataResponse{
		ChannelID: common.ChannelIDTransaction,
		Payload:   tx,
	}

	peerIDs := []string{}
	mp.dispatcher.SendData(peerIDs, data)
}
